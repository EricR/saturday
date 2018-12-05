package solver

import (
	"fmt"
	"github.com/ericr/saturday/lit"
	"github.com/ericr/saturday/tribool"
	"log"
)

const (
	VersionMajor = 1
	VersionMinor = 0
)

// Solver is the SAT solver.
type Solver struct {
	// logger logs messages produced by the solver.
	logger *log.Logger

	// Model Database Fields

	// userVars keeps a map of user-defined variables to internal variables.
	userVars map[int]int
	// internalVars keeps a map of internal variables to user-defined variables.
	internalVars map[int]int
	// model stores a discovered model.
	model map[int]bool
	// status is eventually set to true on sat, false on unsat.
	status tribool.Tribool

	// Constraint Database Fields

	// constrs is a list of problem constraints.
	constrs []*Clause
	// learnts is a list of learnt clauses.
	learnts []*Clause
	// claInc is the clause activity increment.
	claInc float64
	// claDeacy is the decay factor for clause activity.
	claDecay float64

	// Variable Order Fields
	//
	// activity is a heuristic measurement of the activity of a variable.
	activity []float64
	// varInc is the variable activity increment.
	varInc float64
	// varDecay is the decay factor for variable activity.
	varDecay float64
	// order keeps track of dynamic variable ordering.
	order *varOrder

	// Propagation Fields

	// watches contains each literal and a list of constraints watching it.
	watches map[lit.Lit][]*Clause
	// propQ is the propagation queue.
	propQ *lit.Queue

	// Assignment Fields

	// assigns contains the solver's current assignments indexed on variables.
	assigns []tribool.Tribool
	// trail is a list of assignments in chronological order.
	trail []lit.Lit
	// trailLim is a list of separator indices for different decision levels in
	// the trail.
	trailLim []int
	// reason is a list of each variable's constraint that implied its value.
	reason []*Clause
	// level is a list of each variable's decision level at which it was assigned.
	level []int
	// rootLevel separates incremental and search assumptions.
	rootLevel int
}

// New returns a new initialized solver.
func New(logger *log.Logger) *Solver {
	s := &Solver{
		logger:       logger,
		status:       tribool.Undef,
		userVars:     map[int]int{},
		internalVars: map[int]int{},
		model:        map[int]bool{},
		learnts:      []*Clause{},
		activity:     []float64{},
		watches:      map[lit.Lit][]*Clause{},
		propQ:        lit.NewQueue(),
		assigns:      []tribool.Tribool{},
		trail:        []lit.Lit{},
		trailLim:     []int{},
		reason:       []*Clause{},
		level:        []int{},
	}
	s.order = newVarOrder(&s.assigns, &s.activity)

	return s
}

// Version returns the version of the solver.
func Version() string {
	return fmt.Sprintf("%d.%d", VersionMajor, VersionMinor)
}

// Solve accepts a list of constraints and solves the SAT problem, returning
// true when satisfactory and false when unsatisfactory.
func (s *Solver) Solve(ps []int) bool {
	assumps := []lit.Lit{}
	params := searchParams{0.95, 0.999}
	maxConflicts := 100.0
	maxLearnts := float64(s.nConstraints() / 3)

	for _, p := range ps {
		assumps = append(assumps, s.newVar(lit.NewFromInt(p)))
	}

	s.logger.Printf("Starting solver with %d assumptions", len(assumps))

	for i := 0; i < len(assumps); i++ {
		if !s.assume(assumps[i]) || s.propagate() != nil {
			s.cancelUntil(0)

			return false
		}
	}
	s.rootLevel = s.decisionLevel()

	for s.status.Undef() {
		s.status = s.search(int(maxConflicts), int(maxLearnts), params)
		maxConflicts *= 1.5
		maxLearnts *= 1.1
	}
	return s.status.True()
}

// AddClause adds a new clause to the solver.
func (s *Solver) AddClause(ps []int) bool {
	lits := []lit.Lit{}

	for _, p := range ps {
		lits = append(lits, s.newVar(lit.NewFromInt(p)))
	}
	success, c := newClause(s, lits, false)
	if success {
		s.constrs = append(s.constrs, c)
	}
	return success
}

// Answer returns the model.
func (s *Solver) Answer() map[int]bool {
	return s.model
}

// newVar adds a new variable to the solver, referenced thereafter by its index.
func (s *Solver) newVar(p lit.Lit) lit.Lit {
	if _, ok := s.userVars[p.Var()]; !ok {
		s.logger.Printf("Internally assigning %d = %d", p.Var(), s.nVars()+1)

		s.userVars[p.Var()] = s.nVars()
		s.internalVars[s.nVars()] = p.Var()
		s.watches[lit.New(s.nVars()+1, false)] = []*Clause{}
		s.watches[lit.New(s.nVars()+1, true)] = []*Clause{}
		s.reason = append(s.reason, nil)
		s.assigns = append(s.assigns, tribool.Undef)
		s.level = append(s.level, -1)
		s.activity = append(s.activity, float64(0))
	}
	return lit.New(s.userVars[p.Var()], p.Sign())
}

// simplifyDB can be called before solve() and simplifies the constraint
// database. If a top-level conflict is found, returns false.
func (s *Solver) simplifyDB() bool {
	s.logger.Print("Simplifying constraints DB")

	if s.propagate() != nil {
		return false
	}
	for t := 0; t < 2; t++ {
		var cs []*Clause
		if t > 0 {
			cs = s.learnts
		} else {
			cs = s.constrs
		}
		j := 0
		for i := 0; i < len(cs); i++ {
			if cs[i].simplify() {
				cs[i].remove()
			} else {
				cs[j] = cs[i]
				j++
			}
		}
		cs = cs[:len(cs)-j]
	}
	return true
}

// reduceDB removes half of the learnt clauses minus some locked clauses.
func (s *Solver) reduceDB() {
	s.logger.Print("Reducing constraints DB")

	lim := s.claInc / float64(s.nLearnts())
	i := 0
	j := 0

	for i = 0; i < s.nLearnts()/2; i++ {
		if !s.learnts[i].locked() {
			s.learnts[i].remove()
		} else {
			j++
			s.learnts[j] = s.learnts[i]
		}
	}
	for ; i < s.nLearnts(); i++ {
		if !s.learnts[i].locked() && s.learnts[i].activity < lim {
			s.learnts[i].remove()
		} else {
			j++
			s.learnts[j] = s.learnts[i]
		}
	}
	s.learnts = s.learnts[:i-j]
}

// enqueue puts a new fact, p, into the propagation queue.
func (s *Solver) enqueue(p lit.Lit, from *Clause) bool {
	// Check if the fact isn't new first.
	if s.litValue(p) != tribool.Undef {
		if s.litValue(p).False() {
			// Conflicting assignment.
			s.logger.Printf("Conflicting assignment: %s = false", p)
			return false
		} else {
			// Consistent assignment already exists.
			s.logger.Printf("Consistent assignment already exists: %s = true", p)
			return true
		}
	}

	// Fact is new, store and enqueue it.
	s.assigns[p.Index()] = tribool.NewFromBool(!p.Sign())
	s.level[p.Index()] = s.decisionLevel()
	s.reason[p.Index()] = from
	s.trail = append(s.trail, p)
	s.propQ.Insert(p)

	return true
}

// propagate propagates all enqueued facts.
func (s *Solver) propagate() *Clause {
	for s.propQ.Size() > 0 {
		p := s.propQ.Dequeue()

		s.logger.Printf("Propagating %s", p)

		tmp := s.watches[p]
		s.watches[p] = []*Clause{}

		for i := 0; i < len(tmp); i++ {
			// Check for conflict.
			if !(tmp[i].propagate(p)) {
				for j := i + 1; j < len(tmp); j++ {
					s.watches[p] = append(s.watches[p], tmp[j])
				}
				s.propQ.Clear()

				return tmp[i]
			}
		}
	}
	return nil
}

// analyze performs analysis on a conflict.
func (s *Solver) analyze(confl *Clause) ([]lit.Lit, int) {
	seen := make([]bool, s.nVars())
	counter := 0
	p := lit.Undef
	learnts := []lit.Lit{lit.Undef}
	btLevel := 0

	for {
		pReason := confl.calcReason(p)
		// Trace reason for p.
		for j := 0; j < len(pReason); j++ {
			q := pReason[j]

			if !seen[q.Index()] {
				seen[q.Index()] = true
				level := s.level[q.Index()]

				switch {
				case level == s.decisionLevel():
					counter++
				case level > 0:
					learnts = append(learnts, q.Not())

					if level > btLevel {
						btLevel = level
					}
				}
			}
		}
		// Select next literal to look at.
		for {
			p = s.trail[len(s.trail)-1]
			confl = s.reason[p.Index()]
			s.undoOne()

			if seen[p.Index()] {
				break
			}
		}
		counter--
		if counter == 0 {
			break
		}
	}
	learnts[0] = p.Not()

	return learnts, btLevel
}

// search assumes and propagates until a conflict is found. When this happens,
// the conflict is learnt and backtracking is performed until the search can
// continue.
func (s *Solver) search(maxConflicts, maxLearnts int,
	params searchParams) tribool.Tribool {
	// Update decay from search params.
	s.varDecay = 1 / params.varDecay
	s.claDecay = 1 / params.claDecay

	// Clear the model.
	s.model = map[int]bool{}

	nConflicts := 0

	for {
		s.logger.Printf("Searching with params var_decay=%f, cla_decay=%f, "+
			"max_conflicts=%d, max_learnts=%d", s.varDecay, s.claDecay, maxConflicts,
			maxLearnts)

		confl := s.propagate()

		if confl != nil {
			// Conflict detected.
			nConflicts++

			if s.decisionLevel() == s.rootLevel {
				return tribool.False
			}
			learntClause, backtrackLevel := s.analyze(confl)
			if backtrackLevel > s.rootLevel {
				s.cancelUntil(backtrackLevel)
			} else {
				s.cancelUntil(s.rootLevel)
			}
			s.record(learntClause)
			s.decayActivities()
		} else {
			// No conflict detected.
			s.logger.Print("No conflict detected")

			if s.decisionLevel() == 0 {
				s.simplifyDB()
			}
			if s.nLearnts()-s.nAssigns() >= maxLearnts {
				s.reduceDB()
			}
			if s.nAssigns() == s.nVars() {
				// Model found.
				s.logger.Print("Found a model")
				for i := 0; i < s.nVars(); i++ {
					s.model[s.internalVars[i]] = s.assigns[i] == tribool.True
				}
				s.cancelUntil(s.rootLevel)

				return tribool.True
			}
			if nConflicts >= maxConflicts {
				// Reached bound on number of conflicts, so force a restart.
				s.logger.Print("Forcing restart")

				s.cancelUntil(s.rootLevel)

				return tribool.Undef
			}
			// Decide on a new variable.
			s.logger.Print("Deciding on new variable")

			p := lit.New(s.order.choose(), false)
			s.assume(p)
		}
	}
}

// record records a new learnt clause.
func (s *Solver) record(lits []lit.Lit) {
	_, c := newClause(s, lits, true)
	s.enqueue(lits[0], c)

	if c != nil {
		s.learnts = append(s.learnts, c)
	}
}

// assume assumes a literal, returning false if immediate conflict.
func (s *Solver) assume(p lit.Lit) bool {
	s.logger.Printf("Assuming %s", p)

	s.trailLim = append(s.trailLim, s.nAssigns())

	return s.enqueue(p, nil)
}

// undoOne unbinds the last assigned variable.
func (s *Solver) undoOne() {
	s.logger.Print("Undoing")

	p := s.trail[len(s.trail)-1]
	s.assigns[p.Index()] = tribool.Undef
	s.reason[p.Index()] = nil
	s.level[p.Index()] = -1
	s.trail = s.trail[:s.nAssigns()-1]
}

// cancel reverts all variable assignments since the last decision level.
func (s *Solver) cancel() {
	s.logger.Print("Canceling")

	c := s.nAssigns() - s.trailLim[len(s.trailLim)-1]
	for c != 0 {
		s.undoOne()
		c--
	}
	s.trailLim = s.trailLim[:len(s.trailLim)-1]
}

// cancelUntil cancels all variable assignments since the referenced level.
func (s *Solver) cancelUntil(level int) {
	for s.decisionLevel() > level {
		s.cancel()
	}
}

// varBumpActivity bumps a variable's activity.
func (s *Solver) varBumpActivity(p lit.Lit) {
	s.logger.Printf("Bumping var activity for %s", p)

	s.activity[p.Index()] += s.varInc

	if s.activity[p.Index()] > 1e100 {
		s.varRescaleActivity()
	}
}

// varDecayActivity applies decay to varInc.
func (s *Solver) varDecayActivity() {
	s.varInc *= s.varDecay
}

// varRescaleActivity rescales var activity.
func (s *Solver) varRescaleActivity() {
	for i := 0; i < s.nVars(); i++ {
		s.activity[i] *= 1e-100
	}
	s.varInc *= 1e-100
}

// claBumpActivity bumps a clause's activity.
func (s *Solver) claBumpActivity(c *Clause) {
	c.activity += s.claInc

	if c.activity > 1e100 {
		s.claRescaleActivity()
	}
}

// claDecayActivity applies decay to claInc.
func (s *Solver) claDecayActivity() {
	s.claInc *= s.claDecay
}

// claRescaleActivity rescales clause activity.
func (s *Solver) claRescaleActivity() {
	for i := 0; i < s.nLearnts(); i++ {
		s.learnts[i].activity *= 1e-100
	}
	s.claInc *= 1e-100
}

// decayActivities calls both activity decay functions.
func (s *Solver) decayActivities() {
	s.varDecayActivity()
	s.claDecayActivity()
}

// nVars returns the number of variables.
func (s *Solver) nVars() int {
	return len(s.assigns)
}

// nAssigns returns the number of assignments made.
func (s *Solver) nAssigns() int {
	return len(s.trail)
}

// nLearnts returns the number of learnt clauses.
func (s *Solver) nLearnts() int {
	return len(s.learnts)
}

// nConstraints returns the number of constraints.
func (s *Solver) nConstraints() int {
	return len(s.constrs)
}

// litValue returns p's value.
func (s *Solver) litValue(p lit.Lit) tribool.Tribool {
	if p == lit.Undef {
		return tribool.Undef
	}
	if p.Sign() {
		return s.assigns[p.Index()].Not()
	}
	return s.assigns[p.Index()]
}

// decisionLevel returns a solver's decision level.
func (s *Solver) decisionLevel() int {
	return len(s.trailLim)
}

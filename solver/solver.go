package solver

import (
	"fmt"
	"github.com/ericr/saturday/config"
	"github.com/ericr/saturday/lit"
	"github.com/ericr/saturday/tribool"
	"log"
	"math"
	"sort"
)

const (
	VersionMajor = 1
	VersionMinor = 0
)

// Solver is the SAT solver.
type Solver struct {
	config *config.Config
	logger *log.Logger

	// Model Database Fields

	// userVars keeps a map of user-defined variables to internal variables.
	userVars map[int]int
	// internalVars keeps a map of internal variables to user-defined variables.
	internalVars map[int]int
	// model stores the most recently discovered model.
	model map[int]bool

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

	// Algorithmic Restarts Fields

	// maxLearnts is the maximum number of learnt clauses before reduceDB() gets
	// called.
	maxLearnts float64
	// maxLearntsGrowth is the growth factor for maxLearnts.
	maxLearntsGrowth float64
	// maxLearntsCtr is a counter that controls how often maxLearnts gets
	// increased.
	maxLearntsCtr int
	// maxLearntsCtrInc is the amount to increase maxLearntsCtr once it reaches
	// zero.
	maxLearntsCtrInc float64
	// maxLearntsCtrIncGrowth is the growth factor for maxLearntsCtrInc.
	maxLearntsCtrIncGrowth float64
	// maxConflicts is the maximum number of conflicts before a restart occurs.
	maxConflicts float64
	// maxConflictsGrowthStart is the starting constant for maxConflicts's
	// growth.
	maxConflictsGrowthStart float64
	// maxConflictsGrowth is the base of the growth factor for maxConflicts.
	maxConflictsGrowthBase float64

	// Stats Fields

	// propagations keeps track of how many propagations have occurred.
	propagations int
	// conflicts keeps track of how many conflicts have occurred.
	conflicts int
	// restarts keeps track of how many restarts have occurred.
	restarts int
	// decisions keeps track of how many new variables are decided on.
	decisions int
}

// New returns a new initialized solver.
func New(c *config.Config) *Solver {
	s := &Solver{
		config:       c,
		logger:       c.Logger,
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
	s.logger.Print("Starting solver")

	assumps := []lit.Lit{}
	params := searchParams{0.95, 0.999}
	status := tribool.Undef

	// Set values for activity algorithm.
	s.varInc = 1
	s.claInc = 1

	// Set values for the maxLearnts growth algorithm.
	s.maxLearnts = float64(s.NConstrs()) / 3.0
	s.maxLearntsGrowth = 1.1
	s.maxLearntsCtrInc = 100.0
	s.maxLearntsCtr = int(s.maxLearntsCtrInc)
	s.maxLearntsCtrIncGrowth = 1.5

	// Set values for the maxConflicts growth algorithm.
	s.maxConflictsGrowthStart = 100.0
	s.maxConflictsGrowthBase = 2.0

	if !s.simplifyDB() {
		return false
	}
	s.order.init()

	for _, p := range ps {
		assump := lit.NewFromInt(p)

		if _, ok := s.userVars[assump.Var()]; !ok {
			// Illegal assumption.
			return false
		}
		assumps = append(assumps, s.newVar(assump))
	}
	for i := 0; i < len(assumps); i++ {
		if !s.assume(assumps[i]) || s.propagate() != nil {
			s.cancelUntil(0)

			return false
		}
	}
	s.rootLevel = s.decisionLevel()

	for status.Undef() {
		s.maxConflicts = s.maxConflictsGrowthStart *
			math.Pow(s.maxConflictsGrowthBase, float64(s.restarts))
		status = s.search(params)
		s.restarts++
	}
	s.cancelUntil(0)

	return status.True()
}

func (s *Solver) SolveMany(ps []int, mCount uint) [][]int {
	models := [][]int{}

	for i := 0; i < int(mCount); i++ {
		if s.Solve(ps) {
			s.logger.Printf("Found %d/%d models", i+1, mCount)

			models = append(models, s.Answer())
			constrs := s.constrs

			s = New(s.config)

			for _, c := range constrs {
				s.AddClause(c.asInts())
			}
			for _, model := range models {
				newConstr := []int{}

				for _, l := range model {
					newConstr = append(newConstr, -l)
				}
				s.AddClause(newConstr)
			}
		} else {
			s.logger.Printf("No more models exist")
			break
		}
	}
	return models
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

// Answer returns the model as CNF.
func (s *Solver) Answer() []int {
	ps := []int{}

	for p, val := range s.model {
		if val {
			ps = append(ps, p)
		} else {
			ps = append(ps, -p)
		}
	}
	sort.Slice(ps, func(i, j int) bool {
		i, j = ps[i], ps[j]

		if i < 0 {
			i = -i
		}
		if j < 0 {
			j = -j
		}
		return i < j
	})
	return ps
}

// NVars returns the number of variables.
func (s *Solver) NVars() int {
	return len(s.assigns)
}

// NAssigns returns the number of assignments made.
func (s *Solver) NAssigns() int {
	return len(s.trail)
}

// NLearnts returns the number of learnt clauses.
func (s *Solver) NLearnts() int {
	return len(s.learnts)
}

// NConstraints returns the number of constraints.
func (s *Solver) NConstrs() int {
	return len(s.constrs)
}

// NPropagations returns the number of propagations that have occurred.
func (s *Solver) NPropagations() int {
	return s.propagations
}

// NConflicts returns the number of conflicts that have occurred.
func (s *Solver) NConflicts() int {
	return s.conflicts
}

// NRestarts returns the number of restarts that have occurred.
func (s *Solver) NRestarts() int {
	return s.restarts
}

// NDecisions returns the number of variable choosing decisions made.
func (s *Solver) NDecisions() int {
	return s.decisions
}

// newVar adds a new variable to the solver, referenced thereafter by its index.
func (s *Solver) newVar(p lit.Lit) lit.Lit {
	if _, ok := s.userVars[p.Var()]; !ok {
		s.userVars[p.Var()] = s.NVars()
		s.internalVars[s.NVars()] = p.Var()
		s.watches[lit.New(s.NVars()+1, false)] = []*Clause{}
		s.watches[lit.New(s.NVars()+1, true)] = []*Clause{}
		s.reason = append(s.reason, nil)
		s.assigns = append(s.assigns, tribool.Undef)
		s.level = append(s.level, -1)
		s.activity = append(s.activity, float64(0))
		s.order.newVar()
	}
	return lit.New(s.userVars[p.Var()], p.Sign())
}

// search assumes and propagates until a conflict is found. When this happens,
// the conflict is learnt and backtracking is performed until the search can
// continue.
func (s *Solver) search(params searchParams) tribool.Tribool {
	// Update decay vars from search params.
	s.varDecay = 1 / params.varDecay
	s.claDecay = 1 / params.claDecay

	// Reset model and number of conflicts.
	s.model = map[int]bool{}
	nConflicts := 0

	for {
		if confl := s.propagate(); confl != nil {
			// Conflict detected.
			nConflicts++
			s.conflicts++

			// No more decisions can be made.
			if s.decisionLevel() == s.rootLevel {
				return tribool.False
			}

			// Analyze the conflict and produce a learnt clause.
			learntClause, backtrackLevel := s.analyze(confl)

			// Perform backtracking.
			if backtrackLevel > s.rootLevel {
				s.cancelUntil(backtrackLevel)
			} else {
				s.cancelUntil(s.rootLevel)
			}

			// Record new learnt clause.
			s.record(learntClause)

			// Update heuristics.
			s.decayActivities()
			s.maxLearntsCtr -= 1
			if s.maxLearntsCtr == 0 {
				s.maxLearntsCtrInc *= s.maxLearntsCtrIncGrowth
				s.maxLearntsCtr = int(s.maxLearntsCtrInc)
				s.maxLearnts *= s.maxLearntsGrowth
			}
		} else {
			// No conflict detected.
			if s.NAssigns() == s.NVars() {
				// All vars are assigned with no conflicts, so we know we have a model.
				for i := 0; i < s.NVars(); i++ {
					s.model[s.internalVars[i]] = s.assigns[i] == tribool.True
				}
				s.cancelUntil(s.rootLevel)

				return tribool.True
			}

			// Simplify problem clauses.
			if s.decisionLevel() == 0 {
				s.simplifyDB()
			}

			// Check if maxLearnts has been reached, and if so reduce the DB.
			if s.NLearnts()-s.NAssigns() >= int(s.maxLearnts) {
				s.reduceDB()
			}

			// Force a restart if max conflicts is reached, else decide on a new var.
			if nConflicts >= int(s.maxConflicts) {
				s.cancelUntil(s.rootLevel)

				return tribool.Undef
			} else {
				s.assume(lit.NewFromInt(s.order.Choose()))
				s.decisions++
			}
		}
	}
}

// simplifyDB can be called before solve() and simplifies the constraint
// database. If a top-level conflict is found, returns false.
func (s *Solver) simplifyDB() bool {
	if s.propagate() != nil {
		return false
	}
	j := 0
	for i := 0; i < s.NLearnts(); i++ {
		if s.learnts[i].simplify() {
			s.learnts[i].remove()
		} else {
			s.learnts[j] = s.learnts[i]
			j++
		}
	}
	s.learnts = s.learnts[:j]

	return true
}

// reduceDB removes half of the learnt clauses minus some locked clauses.
func (s *Solver) reduceDB() {
	i := 0
	j := 0
	lim := s.claInc / float64(s.NLearnts())

	s.sortLearnts()

	for i, j = 0, 0; i < s.NLearnts(); i++ {
		c := s.learnts[i]

		if c.Len() > 2 && !c.locked() && (i < s.NLearnts()/2 || c.activity < lim) {
			c.remove()
		} else {
			s.learnts[j] = s.learnts[i]
			j++
		}
	}
	s.learnts = s.learnts[:j]
}

// enqueue puts a new fact, p, into the propagation queue.
func (s *Solver) enqueue(p lit.Lit, from *Clause) bool {
	// Check if the fact isn't new first.
	if s.litValue(p) != tribool.Undef {
		if s.litValue(p).False() {
			// Conflicting assignment.
			return false
		} else {
			// Consistent assignment already exists.
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

		tmp := s.watches[p]
		s.watches[p] = []*Clause{}
		s.propagations++

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

// analyze performs analysis on a conflict, returning the reason and the level
// to backtrack to (highest level in conflict clause).
func (s *Solver) analyze(confl *Clause) ([]lit.Lit, int) {
	seen := make([]bool, s.NVars())
	p := lit.Undef
	learnts := []lit.Lit{lit.Undef}
	counter := 0
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
					learnts = append(learnts, q)

					// Keep track of highest level to return.
					if level > btLevel {
						btLevel = level
					}
				}
			}
		}
		// Select the next literal to look at.
		for {
			p = s.trail[s.NAssigns()-1]

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
	s.trailLim = append(s.trailLim, s.NAssigns())

	return s.enqueue(p, nil)
}

// undoOne unbinds the last assigned variable.
func (s *Solver) undoOne() {
	p := s.trail[s.NAssigns()-1]

	s.assigns[p.Index()] = tribool.Undef
	s.reason[p.Index()] = nil
	s.level[p.Index()] = -1
	s.trail = s.trail[:s.NAssigns()-1]
	s.order.push(p.Index())
}

// cancel reverts all variable assignments since the last decision level.
func (s *Solver) cancel() {
	c := s.NAssigns() - s.trailLim[s.decisionLevel()-1]
	for ; c > 0; c-- {
		s.undoOne()
	}
	s.trailLim = s.trailLim[:s.decisionLevel()-1]
}

// cancelUntil cancels all variable assignments since the referenced level.
func (s *Solver) cancelUntil(level int) {
	for s.decisionLevel() > level {
		s.cancel()
	}
}

// varBumpActivity bumps a variable's activity.
func (s *Solver) varBumpActivity(p lit.Lit) {
	s.activity[p.Index()] += s.varInc

	if s.activity[p.Index()] > 1e100 {
		s.varRescaleActivity()
	}
	s.order.fix(p.Index())
}

// varDecayActivity applies decay to varInc.
func (s *Solver) varDecayActivity() {
	s.varInc *= s.varDecay
}

// varRescaleActivity rescales var activity.
func (s *Solver) varRescaleActivity() {
	for i := 0; i < s.NVars(); i++ {
		s.activity[i] *= 1e-100
	}
	s.varInc *= 1e-100
}

// claBumpActivity bumps a clause's activity.
func (s *Solver) claBumpActivity(c *Clause) {
	c.activity += s.claInc

	if c.activity+s.claInc > 1e20 {
		s.claRescaleActivity()
	}
}

// claDecayActivity applies decay to claInc.
func (s *Solver) claDecayActivity() {
	s.claInc *= s.claDecay
}

// claRescaleActivity rescales clause activity.
func (s *Solver) claRescaleActivity() {
	for i := 0; i < s.NLearnts(); i++ {
		s.learnts[i].activity *= 1e-20
	}
	s.claInc *= 1e-20
}

// decayActivities calls both activity decay functions.
func (s *Solver) decayActivities() {
	s.varDecayActivity()
	s.claDecayActivity()
}

// sortLearnts sorts learnts by activity.
func (s *Solver) sortLearnts() {
	sort.Slice(s.learnts, func(i, j int) bool {
		return s.learnts[i].activity < s.learnts[i].activity
	})
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

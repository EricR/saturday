package solver

import (
	"fmt"
	"github.com/ericr/saturday/config"
	"github.com/ericr/saturday/lit"
	"github.com/ericr/saturday/order"
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
	// config is the solver's configuration
	config *config.Config
	// logger is the solver's logger
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
	order *order.Order

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
	s.order = order.New(&s.assigns, &s.activity)

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
	params := searchParams{s.config.VarDecay, s.config.ClaDecay}
	status := tribool.Undef

	// Set values for activity algorithm.
	s.varInc = 1.0
	s.claInc = 1.0

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
	s.order.Init()

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
		s.order.NewVar()
	}
	return lit.New(s.userVars[p.Var()], p.Sign())
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

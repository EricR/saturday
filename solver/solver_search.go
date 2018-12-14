package solver

import (
	"github.com/ericr/saturday/lit"
	"github.com/ericr/saturday/tribool"
)

// searchParams are params supported by search.
type searchParams struct {
	varDecay float64
	claDecay float64
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
	s.order.Push(p.Index())
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

// decisionLevel returns a solver's decision level.
func (s *Solver) decisionLevel() int {
	return len(s.trailLim)
}

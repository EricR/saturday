package solver

import (
	"github.com/ericr/saturday/lit"
	"sort"
)

// varBumpActivity bumps a variable's activity.
func (s *Solver) varBumpActivity(p lit.Lit) {
	s.activity[p.Index()] += s.varInc

	if s.activity[p.Index()] > 1e100 {
		s.varRescaleActivity()
	}
	s.order.Fix(p.Index())
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

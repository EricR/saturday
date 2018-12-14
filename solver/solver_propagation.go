package solver

import (
	"github.com/ericr/saturday/lit"
	"github.com/ericr/saturday/tribool"
)

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

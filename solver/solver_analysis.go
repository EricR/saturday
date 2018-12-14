package solver

import "github.com/ericr/saturday/lit"

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

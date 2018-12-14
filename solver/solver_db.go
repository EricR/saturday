package solver

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

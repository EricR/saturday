package solver

// simplify attempts to simplify the clause.
func (c *Clause) simplify() bool {
	j := 0
	for i := 0; i < c.Len(); i++ {
		// Constraint is already satisfied.
		if c.solver.litValue(c.lits[i]).True() {
			return true
		}
		// Don't copy undefined literals.
		if c.solver.litValue(c.lits[i]).Undef() {
			c.lits[j] = c.lits[i]
			j++
		}
	}
	c.lits = c.lits[:j]

	return false
}

package solver

import "github.com/ericr/saturday/lit"

// propagate attempts to infer additional unit info and, if found, adds it to
// the propagation queue.
func (c *Clause) propagate(p lit.Lit) bool {
	// Make sure the false literal is lits[1].
	if c.lits[0] == p.Not() {
		c.lits[0], c.lits[1] = c.lits[1], p.Not()
	}
	// If 0th watch is true, then the clause is already satisfied. We just need
	// to reinsert it into the watch list.
	if c.solver.litValue(c.lits[0]).True() {
		c.addToWatcher(p)

		return true
	}
	// Look for a new literal to watch and insert this clause into its watch list.
	for i := 2; i < c.Len(); i++ {
		if !c.solver.litValue(c.lits[i]).False() {
			c.lits[1], c.lits[i] = c.lits[i], p.Not()
			c.addToWatcher(c.lits[1].Not())

			return true
		}
	}
	// Clause is unit under assignment.
	c.addToWatcher(p)

	return c.solver.enqueue(c.lits[0], c)
}

// calcReason returns the reason p was propagated.
func (c *Clause) calcReason(p lit.Lit) []lit.Lit {
	outReason := []lit.Lit{}
	offset := 1
	if c.solver.litValue(p).Undef() {
		offset = 0
	}
	for i := offset; i < c.Len(); i++ {
		outReason = append(outReason, c.lits[i].Not())
	}
	if c.learnt {
		c.solver.claBumpActivity(c)
	}
	return outReason
}

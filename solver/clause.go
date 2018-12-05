package solver

import (
	"github.com/ericr/saturday/lit"
	"sort"
	"strings"
)

// Clause is a CNF clause.
type Clause struct {
	solver   *Solver
	lits     []lit.Lit
	learnt   bool
	activity float64
}

// newClause returns a new initialized clause or false on top-level conflict.
func newClause(s *Solver, lits []lit.Lit, learnt bool) (bool, *Clause) {
	c := &Clause{
		solver: s,
		lits:   lits,
		learnt: learnt,
	}
	sort.Sort(c)

	if !learnt {
		idx := 0
		last := lit.Undef

		for _, p := range c.lits {
			switch {
			case s.litValue(p).True():
				// Return on literal already true.
				c.solver.logger.Printf("Literal %s already true", p)
				return true, nil
			case p == last.Not():
				// Return on tautology.
				c.solver.logger.Printf("Tautology detected for %s", p)
				return true, nil
			case s.litValue(p).False():
				// Remove false literal.
				c.solver.logger.Printf("Skipping false literal %s", p)
				continue
			}
			c.lits[idx] = p
			last = p
			idx++
		}
		c.lits = c.lits[:idx]
	}

	switch c.Len() {
	case 0:
		// Return with conflict on empty clause.
		return false, nil
	case 1:
		// Unit detected, enqueue it.
		c.solver.logger.Print("Unit detected")

		return s.enqueue(c.lits[0], c), nil
	}

	if learnt {
		// Pick a second literal to watch.
		idx := c.highestDecisionLevelIdx()
		c.lits[1], c.lits[idx] = c.lits[idx], c.lits[1]

		// Newly learnt clauses are considered active.
		c.solver.claBumpActivity(c)

		for i := 0; i < c.Len(); i++ {
			c.solver.varBumpActivity(c.lits[i])
		}
	}

	c.addToWatcher(c.lits[0].Not())
	c.addToWatcher(c.lits[1].Not())

	return true, c
}

// locked returns true if the clause is locked.
func (c *Clause) locked() bool {
	return c.solver.reason[c.lits[0]] == c
}

// remove removes the clause from the solver.
func (c *Clause) remove() {
	c.removeFromWatcher(c.lits[0].Not())
	c.removeFromWatcher(c.lits[1].Not())
}

// simplify attempts to simplify the clause.
func (c *Clause) simplify() bool {
	j := 0
	for i := 0; i < c.Len(); i++ {
		// Constraint is already satisfied.
		if c.solver.litValue(c.lits[i]).True() {
			return true
		}
		// Don't copy false literals
		if c.solver.litValue(c.lits[i]).Undef() {
			c.lits[j] = c.lits[i]
			j++
		}
	}
	c.lits = c.lits[:j]

	return false
}

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
		c.solver.logger.Printf("Clause already satisfied: %s", c)
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
	c.solver.logger.Printf("Clause is unit: %s", c)
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

// addToWatcher adds this clause to p's watch list.
func (c *Clause) addToWatcher(p lit.Lit) {
	c.solver.watches[p] = append(c.solver.watches[p], c)
}

// removeFromWatcher removes this clause to p's watch list.
func (c *Clause) removeFromWatcher(p lit.Lit) {
	for idx, clause := range c.solver.watches[p] {
		if clause == c {
			nWatches := len(c.solver.watches[p])
			c.solver.watches[p][idx] = c.solver.watches[p][nWatches-1]
			c.solver.watches[p] = c.solver.watches[p][:nWatches-1]
		}
	}
}

// highestDecisionLevelIdx returns the clause index of p with the highest
// decision level.
func (c *Clause) highestDecisionLevelIdx() int {
	max := 0
	maxiIdx := 0

	for idx, p := range c.lits {
		dl := c.solver.level[p.Index()]

		if dl > max {
			maxiIdx = idx
			max = dl
		}
	}
	return maxiIdx
}

// asStrings returns a clause as an array of strings.
func (c *Clause) asStrings() []string {
	litStrs := []string{}

	for _, lit := range c.lits {
		litStrs = append(litStrs, lit.String())
	}
	return litStrs
}

// String implements the Stringer interface.
func (c *Clause) String() string {
	return strings.Join(c.asStrings(), ",")
}

// Len returns the length of the clause.
func (c *Clause) Len() int {
	return len(c.lits)
}

// Swap swaps two literals within the clause.
func (c *Clause) Swap(i, j int) {
	c.lits[i], c.lits[j] = c.lits[j], c.lits[i]
}

// Less compares two literals within the clause.
func (c *Clause) Less(i, j int) bool {
	return c.lits[i] < c.lits[j]
}

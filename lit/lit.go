package lit

import "fmt"

const Undef = Lit(-1)

// Lit is a literal represented by an integer. The sign of the literal is
// represented by the least significant bit, and the value is obtained by
// performing a right bit shift. This encoding makes L and ~L adjacent when
// sorted.
//
// An unknown literal is denoted as -1.
type Lit int

// New returns a new literal given a 0-index variable, v, and whether the
// literal is negative.
func New(v int, neg bool) Lit {
	if neg {
		return Lit(v + v + 1)
	}
	return Lit(v + v)
}

// NewFromInt returns a new literal with a variable equal to i.
func NewFromInt(i int) Lit {
	if i < 0 {
		return New(-i-1, true)
	}
	return New(i-1, false)
}

// Not negates a literal.
func (l Lit) Not() Lit {
	return Lit(l ^ 1)
}

// Sign returns true if the literal is negative.
func (l Lit) Sign() bool {
	return l&1 == 1
}

// Index returns the literal's index.
func (l Lit) Index() int {
	return int(l >> 1)
}

// Var returns the literal's variable.
func (l Lit) Var() int {
	return int(l>>1) + 1
}

// Int returns the literal as an integer.
func (l Lit) Int() int {
	if l.Sign() {
		return -l.Var()
	}
	return l.Var()
}

// String implements the Stringer interface.
func (l Lit) String() string {
	if l.Sign() {
		return fmt.Sprintf("~%d", l.Var())
	}
	return fmt.Sprintf("%d", l.Var())
}

package solver

import (
	"github.com/ericr/saturday/config"
	"github.com/ericr/saturday/lit"
	"github.com/ericr/saturday/tribool"
	"testing"
)

func TestDetectClauseTrue(t *testing.T) {
	conf := config.New()
	s := New(conf)

	lits := []lit.Lit{lit.New(0, false), lit.New(0, false)}
	addLits(s, lits)
	s.assigns[0] = tribool.True

	if valid, _ := newClause(s, lits, false); valid != true {
		t.Fatalf("Did not detect already true clause")
	}
}

func TestDetectClauseTautology(t *testing.T) {
	conf := config.New()
	s := New(conf)

	lits := []lit.Lit{lit.New(0, false), lit.New(0, false)}
	addLits(s, lits)

	if valid, _ := newClause(s, lits, false); valid != true {
		t.Fatalf("Did not detect tautology")
	}
}

func TestDetectClauseEmpty(t *testing.T) {
	conf := config.New()
	s := New(conf)

	lits := []lit.Lit{}
	
	if valid, _ := newClause(s, lits, false); valid != false {
		t.Fatalf("Did not detect empty clause")
	}
}

func TestDetectClauseFalseLits(t *testing.T) {
	conf := config.New()
	s := New(conf)

	lits := []lit.Lit{lit.New(0, false), lit.New(1, false), lit.New(2, true)}
	addLits(s, lits)
	s.assigns[1] = tribool.False

	if _, c := newClause(s, lits, false); c.Len() != 2 {
		t.Fatalf("Did not remove false literals")
	}
}

func TestDetectClauseDuplicates(t *testing.T) {
	conf := config.New()
	s := New(conf)

	lits := []lit.Lit{lit.New(0, false), lit.New(0, false), lit.New(1, true)}
	addLits(s, lits)

	if _, c := newClause(s, lits, false); c.Len() != 2 {
		t.Fatalf("Did not remove duplicates")
	}
}

func addLits(s *Solver, lits []lit.Lit) {
	for _, l := range lits {
		s.newVar(l)
	}
}
package lit

import "testing"

func TestNewFromInt(t *testing.T) {
	if lit := NewFromInt(12); lit.Var() != 12 {
		t.Fatalf("TestNewFromInt() failed, got: %d", lit.Var())
	}
	if lit := NewFromInt(-12); lit.Var() != 12 {
		t.Fatalf("TestNewFromInt() failed, got: %d", lit.Var())
	}
}

func TestNot(t *testing.T) {
	if lit := New(12, false).Not(); lit != New(12, true) {
		t.Fatalf("TestNot() failed, got: %d", lit.Var())
	}
}

func TestSign(t *testing.T) {
	if lit := New(12, true); lit.Sign() != true {
		t.Fatalf("TestSign() failed, got: %d", lit.Var())
	}
	if lit := New(12, false); lit.Sign() != false {
		t.Fatalf("TestSign() failed, got: %d", lit.Var())
	}
}

func TestVar(t *testing.T) {
	if lit := New(23, false); lit.Var() != 24 {
		t.Fatalf("TestVar() failed: %d", lit.Var())
	}
	if lit := New(23, true); lit.Var() != 24 {
		t.Fatalf("TestVar() failed: %d", lit.Var())
	}
}

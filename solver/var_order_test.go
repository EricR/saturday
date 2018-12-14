package solver

import (
	"github.com/ericr/saturday/tribool"
	"testing"
)

func TestVarOrderPush(t *testing.T) {
	assigns := []tribool.Tribool{tribool.True, tribool.False}
	activity := []float64{1, 2}

	order := newVarOrder(&assigns, &activity)
	order.newVar()
	order.newVar()

	if order.vars[1] != 1 {
		t.Fatalf("Second element var order is wrong: %v", order.vars[1])
	}
}

func TestVarOrderPop(t *testing.T) {
	assigns := []tribool.Tribool{tribool.True, tribool.False}
	activity := []float64{1, 2}

	order := newVarOrder(&assigns, &activity)
	order.newVar()
	order.newVar()

	if v := order.pop(); v != 0 {
		t.Fatalf("Popped var order is wrong: %v", v)
	}
}

package order

import (
	"github.com/ericr/saturday/tribool"
	"testing"
)

func TestOrderPush(t *testing.T) {
	assigns := []tribool.Tribool{tribool.True, tribool.False}
	activity := []float64{1, 2}

	ord := New(&assigns, &activity)
	ord.NewVar()
	ord.NewVar()

	if ord.vars[1] != 1 {
		t.Fatalf("Second element var order is wrong: %v", ord.vars[1])
	}
}

func TestOrderPop(t *testing.T) {
	assigns := []tribool.Tribool{tribool.True, tribool.False}
	activity := []float64{1, 2}

	ord := New(&assigns, &activity)
	ord.NewVar()
	ord.NewVar()

	if v := ord.pop(); v != 0 {
		t.Fatalf("Popped var order is wrong: %v", v)
	}
}

package solver

import "github.com/ericr/saturday/tribool"

// varOrder assists with dynamic variable ordering.
type varOrder struct {
	assigns  *[]tribool.Tribool
	activity *[]float64
}

// newVarOrder returns a new varOrder.
func newVarOrder(assigns *[]tribool.Tribool, activity *[]float64) *varOrder {
	return &varOrder{
		assigns:  assigns,
		activity: activity,
	}
}

// choose returns the most active unassigned variable.
func (vo *varOrder) choose() int {
	activity := *vo.activity
	assigns := *vo.assigns
	max := float64(0)
	idx := 0

	for i, a := range activity {
		if assigns[i].Undef() && a >= max {
			max = a
			idx = i
		}
	}
	return idx
}

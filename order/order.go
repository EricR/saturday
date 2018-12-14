package order

import (
	"github.com/ericr/saturday/lit"
	"github.com/ericr/saturday/tribool"
)

// Order assists with dynamic variable ordering.
type Order struct {
	vars     []int
	indices  map[int]int
	assigns  *[]tribool.Tribool
	activity *[]float64
}

// New returns a new Order.
func New(assigns *[]tribool.Tribool, activity *[]float64) *Order {
	return &Order{
		vars:     []int{},
		indices:  map[int]int{},
		assigns:  assigns,
		activity: activity,
	}
}

// Init initializes the order's heap.
func (o *Order) Init() {
	n := o.len()
	for i := n/2 - 1; i >= 0; i-- {
		o.down(i, n)
	}
}

// NewVar adds a new var to the order.
func (o *Order) NewVar() {
	n := len(o.vars)
	o.vars = append(o.vars, n)
	o.indices[n] = n
}

// Choose returns an unbound variable with the highest activity, or the integer
// value of lit.Undef when there are no vars left to choose from.
func (o *Order) Choose() int {
	v := 0
	a := *o.assigns

	for {
		if v = o.pop(); a[v].Undef() {
			return v + 1
		}
	}
	return int(lit.Undef)
}

// Push pushes an element onto the heap.
func (o *Order) Push(v int) {
	if o.indices[v] != -1 {
		return
	}
	o.indices[v] = len(o.vars)
	o.vars = append(o.vars, v)
	o.up(o.len() - 1)
}

// Fix fixes ordering of the order's heap at a given index.
func (o *Order) Fix(v int) {
	if o.indices[v] == -1 {
		return
	}
	v = o.indices[v]

	o.down(v, o.len())
	o.up(v)
}

// len implements the sort interface.
func (o *Order) len() int {
	return len(o.vars)
}

// less implements the sort interface.
func (o *Order) less(i, j int) bool {
	return (*o.activity)[i] < (*o.activity)[j]
}

// swap implements the sort interface.
func (o *Order) swap(i, j int) {
	k, l := o.vars[i], o.vars[j]

	o.vars[i], o.vars[j] = l, k
	o.indices[k], o.indices[l] = j, i
}

// pop pops an element off of the order's heap.
func (o *Order) pop() int {
	n := len(o.vars) - 1
	o.swap(0, n)
	o.down(0, n)
	v := o.vars[n]
	o.vars = o.vars[:n]
	o.indices[v] = -1

	return v
}

// up percolates an element from the heap up, as adopted from Go's
// container/heap package.
func (o *Order) up(j int) {
	for {
		i := (j - 1) / 2
		if i == j || !o.less(j, i) {
			break
		}
		o.swap(i, j)
		j = i
	}
}

// down percolates an element from the heap down, as adopted from Go's
// container/heap package.
func (o *Order) down(i0, n int) bool {
	i := i0
	for {
		j1 := 2*i + 1
		if j1 >= n || j1 < 0 {
			break
		}
		j := j1
		if j2 := j1 + 1; j2 < n && !o.less(j2, j1) {
			j = j2
		}
		if !o.less(j, i) {
			break
		}
		o.swap(i, j)
		i = j
	}
	return i > i0
}

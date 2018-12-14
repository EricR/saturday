package solver

import "github.com/ericr/saturday/tribool"

// varOrder assists with dynamic variable ordering.
type varOrder struct {
	vars     []int
	indices  map[int]int
	assigns  *[]tribool.Tribool
	activity *[]float64
}

// newVarOrder returns a new varOrder.
func newVarOrder(assigns *[]tribool.Tribool, activity *[]float64) *varOrder {
	return &varOrder{
		vars:     []int{},
		indices:  map[int]int{},
		assigns:  assigns,
		activity: activity,
	}
}

// Choose returns an unbound variable with the highest activity.
func (vo *varOrder) Choose() int {
	v := 0
	a := *vo.assigns

	for {
		if v = vo.pop(); a[v].Undef() {
			return v + 1
		}
	}
	return -1
}

// newVar adds a new var to varOrder.
func (vo *varOrder) newVar() {
	n := len(vo.vars)
	vo.vars = append(vo.vars, n)
	vo.indices[n] = n
}

// len implements the sort interface.
func (vo *varOrder) len() int {
	return len(vo.vars)
}

// less implements the sort interface.
func (vo *varOrder) less(i, j int) bool {
	return (*vo.activity)[i] < (*vo.activity)[j]
}

// swap implements the sort interface.
func (vo *varOrder) swap(i, j int) {
	k, l := vo.vars[i], vo.vars[j]
	vo.vars[i], vo.vars[j] = l, k
	vo.indices[k], vo.indices[l] = j, i
}

// init initializes the heap.
func (vo *varOrder) init() {
	n := vo.len()
	for i := n/2 - 1; i >= 0; i-- {
		vo.down(i, n)
	}
}

// pop pops an element off of the heap.
func (vo *varOrder) pop() int {
	n := len(vo.vars) - 1
	vo.swap(0, n)
	vo.down(0, n)
	v := vo.vars[n]
	vo.vars = vo.vars[:n]
	vo.indices[v] = -1

	return v
}

// push pushes an element onto the heap.
func (vo *varOrder) push(v int) {
	if vo.indices[v] != -1 {
		return
	}
	vo.indices[v] = len(vo.vars)
	vo.vars = append(vo.vars, v)
	vo.up(vo.len() - 1)
}

// fix fixes ordering of the heap at a given index.
func (vo *varOrder) fix(v int) {
	if vo.indices[v] == -1 {
		return
	}
	v = vo.indices[v]

	vo.down(v, vo.len())
	vo.up(v)
}

// up percolates an element up.
func (vo *varOrder) up(j int) {
	for {
		i := (j - 1) / 2
		if i == j || !vo.less(j, i) {
			break
		}
		vo.swap(i, j)
		j = i
	}
}

// down percolates an element down.
func (vo *varOrder) down(i0, n int) bool {
	i := i0
	for {
		j1 := 2*i + 1
		if j1 >= n || j1 < 0 {
			break
		}
		j := j1
		if j2 := j1 + 1; j2 < n && !vo.less(j2, j1) {
			j = j2
		}
		if !vo.less(j, i) {
			break
		}
		vo.swap(i, j)
		i = j
	}
	return i > i0
}

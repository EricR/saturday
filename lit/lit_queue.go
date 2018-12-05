package lit

// Queue for literals. Note that this is not async-safe.
type Queue struct {
	items []Lit
}

// NewQueue returns a new queue.
func NewQueue() *Queue {
	return &Queue{
		items: []Lit{},
	}
}

// Insert inserts a new lit into the queue.
func (q *Queue) Insert(l Lit) {
	q.items = append(q.items, l)
}

// Dequeue pops the first lit off the queue.
func (q *Queue) Dequeue() Lit {
	if len(q.items) == 0 {
		return Undef
	}
	first := q.items[0]
	q.items = q.items[1:len(q.items)]

	return first
}

// Clear clears the queue.
func (q *Queue) Clear() {
	q.items = []Lit{}
}

// Size returns the size of the queue.
func (q *Queue) Size() int {
	return len(q.items)
}

package lit

import "testing"

func TestQueueInsert(t *testing.T) {
	q := NewQueue()

	if q.Insert(New(0, false)); len(q.items) != 1 {
		t.Fatalf("TestQueueInsert() failed, got: %d", len(q.items))
	}
}

func TestQueueDequeue(t *testing.T) {
	q := NewQueue()
	lit1 := New(0, false)
	lit2 := New(1, false)
	lit3 := New(2, true)

	q.Insert(lit1)
	q.Insert(lit2)
	q.Insert(lit3)

	if o := q.Dequeue(); o != lit1 {
		t.Fatalf("TestQueueInsert() failed, got: %s", o)
	}
	if o := q.Dequeue(); o != lit2 {
		t.Fatalf("TestQueueInsert() failed, got: %s", o)
	}
	if o := q.Dequeue(); o != lit3 {
		t.Fatalf("TestQueueInsert() failed, got: %s", o)
	}
	if len(q.items) != 0 {
		t.Fatalf("TestQueueInsert() failed: didn't remove items")
	}
}

func TestQueueClear(t *testing.T) {
	q := NewQueue()
	q.Insert(New(0, false))
	q.Insert(New(1, false))

	if q.Clear(); len(q.items) != 0 {
		t.Fatalf("TestQueueClear() failed, got: %d", len(q.items))
	}
}

func TestQueueSize(t *testing.T) {
	q := NewQueue()
	q.Insert(New(0, false))
	q.Insert(New(1, false))

	if q.Size() != 2 {
		t.Fatalf("TestQueueSize() failed, got: %d", q.Size())
	}
}

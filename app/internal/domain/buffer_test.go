package domain

import "testing"

func TestBuffer_Logic(t *testing.T) {
	t.Parallel()
	buf := NewBuffer(3)
	buf.PutAt(new(Request), 1)
	if buf.FindOccupiedSlot(0) != 1 {
		t.Fatalf("invalid occupied index, expected: %d, got: %d", 1, buf.FindOccupiedSlot(0))
	}

	_ = buf.RemoveAt(1)
	if buf.FindFreeSlot(1) != 1 {
		t.Fatalf("invalid free index, expected: %d, got: %d", 1, buf.FindFreeSlot(1))
	}
}

func TestBuffer_FindFreeSlot(t *testing.T) {
	t.Run("start = 0", func(t *testing.T) {
		t.Parallel()

		buf := NewBuffer(4)
		buf.slots[1] = &Request{ID: 1}
		buf.slots[2] = &Request{ID: 2}
		buf.slots[3] = &Request{ID: 3}

		i := buf.FindFreeSlot(0)
		if i != 0 {
			t.Fatalf("invalid index, expected: %d, got: %d", 0, i)
		}
	})

	t.Run("start = 2(after all)", func(t *testing.T) {
		t.Parallel()

		buf := NewBuffer(4)
		buf.slots[1] = &Request{ID: 1}
		buf.slots[2] = &Request{ID: 2}
		buf.slots[3] = &Request{ID: 3}

		i := buf.FindFreeSlot(2)
		if i != 0 {
			t.Fatalf("invalid index, expected: %d, got: %d", 0, i)
		}
	})

	t.Run("buffer is full", func(t *testing.T) {
		t.Parallel()

		buf := NewBuffer(3)
		buf.slots[0] = &Request{ID: 1}
		buf.slots[1] = &Request{ID: 2}
		buf.slots[2] = &Request{ID: 3}

		i := buf.FindFreeSlot(1)
		if i != -1 {
			t.Fatalf("invalid index, expected: %d, got: %d", -1, i)
		}
	})
}

func TestBuffer_FindOccupiedSlot(t *testing.T) {
	t.Run("empty", func(t *testing.T) {
		t.Parallel()

		buf := NewBuffer(10)
		index := buf.FindOccupiedSlot(1)
		if index != -1 {
			t.Fatalf("invalid index, expected: %d, got: %d", -1, index)
		}
	})

	t.Run("not empty", func(t *testing.T) {
		t.Parallel()

		buf := NewBuffer(10)
		buf.slots[3] = &Request{}
		index := buf.FindOccupiedSlot(2)
		if index != 3 {
			t.Fatalf("invalid index, expected: %d, got: %d", 3, index)
		}

		index = buf.FindOccupiedSlot(4)
		if index != 3 {
			t.Fatalf("invalid index, expected: %d, got: %d", 3, index)
		}
	})
}

func TestBuffer_PutAt(t *testing.T) {
	t.Parallel()
	buf := NewBuffer(3)
	buf.PutAt(&Request{}, 1)
	if !(buf.slots[1] != nil) {
		t.Fatalf("failed to put at index: %d", 1)
	}
}

func TestBuffer_RemoveAt(t *testing.T) {
	t.Parallel()
	buf := NewBuffer(3)
	buf.PutAt(new(Request), 1)
	if !(buf.slots[1] != nil) {
		t.Fatalf("failed to put at index: %d", 1)
	}

	req := buf.RemoveAt(1)
	if req == nil {
		t.Fatalf("nil req")
	}

	if buf.slots[1] != nil {
		t.Fatalf("failed to remove at index: %d", 1)
	}
}

func TestBuffer_FindOldestSlot(t *testing.T) {
	t.Run("cool", func(t *testing.T) {
		t.Parallel()

		b := NewBuffer(3)
		r1 := &Request{}
		r1.MarkBuffered(8.00)
		b.PutAt(r1, 0)
		r2 := &Request{}
		r2.MarkBuffered(7.20)
		b.PutAt(r2, 1)
		r3 := &Request{}
		r3.MarkBuffered(9.50)
		b.PutAt(r3, 2)
		if b.FindOldestSlot() != 1 {
			t.Fatalf("invalid oldest index, expected: %d, got: %d", 1, b.FindOldestSlot())
		}
	})

	t.Run("empty buffer", func(t *testing.T) {
		t.Parallel()
		b := NewBuffer(3)
		if b.FindOldestSlot() != -1 {
			t.Fatalf("invalid oldest slot, expected: %d, got: %d", -1, b.FindOldestSlot())
		}
	})
}

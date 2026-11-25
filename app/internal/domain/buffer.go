package domain

import "math"

type Buffer struct {
	slots []*Request
}

func NewBuffer(capacity int) *Buffer {
	return &Buffer{slots: make([]*Request, capacity)}
}

func (b *Buffer) FindFreeSlot(start int) int {
	n := len(b.slots)
	for i := range n {
		index := (start + i) % n
		if b.slots[index] == nil {
			return index
		}
	}

	return -1
}

func (b *Buffer) FindOccupiedSlot(start int) int {
	for i := range len(b.slots) {
		index := (start + i) % len(b.slots)
		if b.slots[index] != nil {
			return index
		}
	}

	return -1
}

func (b *Buffer) PutAt(req *Request, at int) {
	b.slots[at] = req
}

func (b *Buffer) RemoveAt(at int) *Request {
	req := b.slots[at]
	b.slots[at] = nil
	return req
}

func (b *Buffer) FindOldestSlot() int {
	m := math.MaxFloat64
	seenAt := map[float64]int{}
	for i, v := range b.slots {
		if v == nil {
			continue
		}

		m = min(v.BufferedAt, m)
		if _, ok := seenAt[m]; !ok {
			seenAt[m] = i
		}
	}

	if _, ok := seenAt[m]; !ok {
		return -1
	}

	return seenAt[m]
}

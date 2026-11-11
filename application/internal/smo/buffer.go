package smo

import "smo-alarms/internal/model"

// Buffer — очередь тревог на N мест. Хранит только места и даёт примитивы поиска
// с заданной позиции. Указатели УП и УВ принадлежат диспетчерам ДП и ДВ.
// Места нумеруются с 0; «следующее за k» = (k + 1) mod N.
type Buffer struct {
	capacity int
	slots    []*model.Request
	occupied int
}

func NewBuffer(capacity int) *Buffer {
	return &Buffer{capacity: capacity, slots: make([]*model.Request, capacity)}
}

// FindFreeSlotByRing — Д1ОЗ1: первое свободное место, начиная со start, по кольцу.
// Если указатель обошёл кольцо и вернулся на start — мест нет, результат −1.
func (b *Buffer) FindFreeSlotByRing(start int) int {
	for i := 0; i < b.capacity; i++ {
		pos := (start + i) % b.capacity
		if b.slots[pos] == nil {
			return pos
		}
	}
	return -1
}

// FindOccupiedSlotByRing — Д2Б3: первое занятое место, начиная со start, по кольцу; −1, если буфер пуст.
func (b *Buffer) FindOccupiedSlotByRing(start int) int {
	for i := 0; i < b.capacity; i++ {
		pos := (start + i) % b.capacity
		if b.slots[pos] != nil {
			return pos
		}
	}
	return -1
}

// FindOldestSlot — Д1ОО3: место заявки с минимальным временем постановки; −1, если буфер пуст.
func (b *Buffer) FindOldestSlot() int {
	best := -1
	for i, r := range b.slots {
		if r == nil {
			continue
		}
		if best < 0 || r.BufferPutTime() < b.slots[best].BufferPutTime() {
			best = i
		}
	}
	return best
}

func (b *Buffer) Put(slot int, request *model.Request) {
	if b.slots[slot] != nil {
		panic("buffer: место уже занято")
	}
	b.slots[slot] = request
	b.occupied++
}

func (b *Buffer) RemoveAt(slot int) *model.Request {
	r := b.slots[slot]
	if r == nil {
		panic("buffer: место пустое")
	}
	b.slots[slot] = nil
	b.occupied--
	return r
}

func (b *Buffer) IsFull() bool     { return b.occupied == b.capacity }
func (b *Buffer) IsEmpty() bool    { return b.occupied == 0 }
func (b *Buffer) Len() int         { return b.occupied }
func (b *Buffer) GetCapacity() int { return b.capacity }

// Snapshot — копия мест буфера для пошагового режима.
func (b *Buffer) Snapshot() []*model.Request {
	out := make([]*model.Request, b.capacity)
	copy(out, b.slots)
	return out
}

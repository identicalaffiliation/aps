// Package engine — календарь особых событий.
package engine

import (
	"container/heap"
	"fmt"
	"sort"
)

// EventType — тип будущего события в календаре.
type EventType int

const (
	ServiceEnd EventType = iota // прибор закончил проверку
	Generation                  // источник прислал тревогу
)

func (t EventType) String() string {
	if t == ServiceEnd {
		return "SERVICE_END"
	}
	return "GENERATION"
}

// Event — особое событие. Для Generation заполнен SourceID, для ServiceEnd — DeviceID.
type Event struct {
	Time     float64
	Type     EventType
	SourceID int
	DeviceID int
	seq      int64
}

// Less — порядок событий: по времени; при равном времени SERVICE_END раньше GENERATION,
// чтобы освободившийся прибор успел забрать заявку до прихода новой.
func (e Event) Less(o Event) bool {
	if e.Time != o.Time {
		return e.Time < o.Time
	}
	if e.Type != o.Type {
		return e.Type < o.Type
	}
	return e.seq < o.seq
}

func (e Event) String() string {
	if e.Type == Generation {
		return fmt.Sprintf("%.3f GENERATION И%d", e.Time, e.SourceID)
	}
	return fmt.Sprintf("%.3f SERVICE_END П%d", e.Time, e.DeviceID)
}

type eventHeap []Event

func (h eventHeap) Len() int           { return len(h) }
func (h eventHeap) Less(i, j int) bool { return h[i].Less(h[j]) }
func (h eventHeap) Swap(i, j int)      { h[i], h[j] = h[j], h[i] }
func (h *eventHeap) Push(x any)        { *h = append(*h, x.(Event)) }
func (h *eventHeap) Pop() any {
	old := *h
	n := len(old)
	x := old[n-1]
	*h = old[:n-1]
	return x
}

// EventCalendar — очередь будущих событий с приоритетом по времени.
type EventCalendar struct {
	queue eventHeap
	seq   int64
}

func NewEventCalendar() *EventCalendar { return &EventCalendar{} }

func (c *EventCalendar) Schedule(e Event) {
	c.seq++
	e.seq = c.seq
	heap.Push(&c.queue, e)
}

func (c *EventCalendar) PollNext() Event { return heap.Pop(&c.queue).(Event) }
func (c *EventCalendar) IsEmpty() bool   { return len(c.queue) == 0 }

// View — отсортированная копия календаря для пошагового режима ОД1.
func (c *EventCalendar) View() []Event {
	out := make([]Event, len(c.queue))
	copy(out, c.queue)
	sort.Slice(out, func(i, j int) bool { return out[i].Less(out[j]) })
	return out
}

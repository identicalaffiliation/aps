// Package dispatch — диспетчеры ДП (постановка) и ДВ (выбор).
package dispatch

import (
	"smo-alarms/internal/model"
	"smo-alarms/internal/smo"
	"smo-alarms/internal/stats"
)

// PlacementResult — результат работы ДП: куда поставили заявку и кого выбили (или nil).
type PlacementResult struct {
	Slot    int
	Evicted *model.Request
}

func (p PlacementResult) HasEviction() bool { return p.Evicted != nil }

// PlacementDispatcher — ДП, компонент приёма тревог. Срабатывает на GENERATION.
// Классический вариант: работает только с буфером и ничего не знает о приборах.
type PlacementDispatcher struct {
	buffer     *smo.Buffer
	statistics *stats.StatisticsCollector
	putPointer int // УП
}

func NewPlacementDispatcher(buffer *smo.Buffer, statistics *stats.StatisticsCollector) *PlacementDispatcher {
	return &PlacementDispatcher{buffer: buffer, statistics: statistics}
}

// Place ставит заявку в буфер.
//
//	Д1ОЗ1: k = поиск свободного места по кольцу с УП;
//	Д1ОО3: если мест нет — выбить самую старую и встать на её место;
//	УП := (k + 1) mod N.
func (d *PlacementDispatcher) Place(request *model.Request, currentTime float64) PlacementResult {
	res := PlacementResult{Slot: d.buffer.FindFreeSlotByRing(d.putPointer)}
	if res.Slot < 0 {
		// Буфер полон ⇒ по инварианту все приборы заняты.
		res.Slot, res.Evicted = d.evictOldest(currentTime)
	}
	d.buffer.Put(res.Slot, request)
	request.MarkBuffered(currentTime)
	d.advancePointer(res.Slot)
	return res
}

// evictOldest — Д1ОО3: удалить самую старую заявку, засчитать отказ, вернуть освободившееся место.
func (d *PlacementDispatcher) evictOldest(currentTime float64) (int, *model.Request) {
	s := d.buffer.FindOldestSlot()
	old := d.buffer.RemoveAt(s)
	old.Reject(currentTime)
	d.statistics.RecordRejected(old)
	return s, old
}

func (d *PlacementDispatcher) advancePointer(slot int) {
	d.putPointer = (slot + 1) % d.buffer.GetCapacity()
}

// GetPutPointer — положение УП для пошагового режима.
func (d *PlacementDispatcher) GetPutPointer() int { return d.putPointer }

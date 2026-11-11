package dispatch

import (
	"math"
	"testing"

	"smo-alarms/internal/dist"
	"smo-alarms/internal/engine"
	"smo-alarms/internal/model"
	"smo-alarms/internal/smo"
	"smo-alarms/internal/stats"
)

func near(a, b float64) bool { return math.Abs(a-b) < 1e-9 }

func req(src, num int, t float64) *model.Request {
	return model.NewRequest(int64(src*100+num), src, num, model.ThresholdExceeded, "obj", t)
}

func putAt(b *smo.Buffer, slot int, r *model.Request, t float64) {
	b.Put(slot, r)
	r.MarkBuffered(t)
}

func labels(b *smo.Buffer) []string {
	var out []string
	for _, r := range b.Snapshot() {
		if r == nil {
			out = append(out, "—")
		} else {
			out = append(out, r.Label())
		}
	}
	return out
}

func eqStrings(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// Сиквенс «Отказ через переполнение буфера при занятых приборах и последующая
// постановка одной заявки на обработку» (sequence_overflow_to_service), те же числа.
func TestOverflowThenOneRequestToService(t *testing.T) {
	buf := smo.NewBuffer(3)
	p1 := smo.NewDevice(1, dist.NewFixed(2.40))
	p2 := smo.NewDevice(2, dist.NewFixed(1.10, 1.80))
	devices := []*smo.Device{p1, p2}
	cal := engine.NewEventCalendar()
	st := stats.NewStatisticsCollector(3, devices)
	dp := NewPlacementDispatcher(buf, st)
	dv := NewSelectionDispatcher(buf, devices, cal, st)

	// Исходное состояние, t = 10,00: буфер [И2-5 (7,20) | И1-7 (8,00) | И3-4 (9,50)], УП = 1, УВ = 1.
	putAt(buf, 0, req(2, 5, 7.2), 7.2)
	putAt(buf, 1, req(1, 7, 8.0), 8.0)
	putAt(buf, 2, req(3, 4, 9.5), 9.5)
	p1.StartService(req(3, 3, 6), 10.0) // П1 занят до 12,40
	p2.StartService(req(2, 4, 6), 10.0) // П2 занят до 11,10

	// t = 10,00: пришла И1-8 → Д1ОЗ1 мест не нашёл → Д1ОО3 выбивает И2-5 из места 1.
	newReq := req(1, 8, 10.0)
	res := dp.Place(newReq, 10.0)
	if !res.HasEviction() || res.Evicted.Label() != "И2-5" || res.Slot != 0 {
		t.Fatalf("ожидали выбивание И2-5 из места 1, получили %+v", res)
	}
	if res.Evicted.Status() != model.Rejected || st.Rejected(2) != 1 {
		t.Fatalf("И2-5 должна получить REJECTED и попасть в статистику отказов")
	}
	if got := labels(buf); !eqStrings(got, []string{"И1-8", "И1-7", "И3-4"}) {
		t.Fatalf("буфер после выбивания: %v", got)
	}
	if dp.GetPutPointer() != 1 {
		t.Fatalf("УП должен стать 2 (индекс 1), получили %d", dp.GetPutPointer()+1)
	}
	// Триггер ДВ «заявка в буфере»: все приборы заняты — назначений нет.
	if as := dv.OnRequestBuffered(10.0); len(as) != 0 {
		t.Fatalf("приборы заняты, назначений быть не должно: %v", as)
	}

	// t = 11,10: П2 освободился → Д2П1 выбирает П2 → Д2Б3 с УВ = 1 берёт И1-8.
	p2.FinishService(11.1)
	as := dv.OnDeviceReleased(11.1)
	if len(as) != 1 {
		t.Fatalf("ожидали одно назначение, получили %d", len(as))
	}
	a := as[0]
	if a.Request.Label() != "И1-8" || a.Device.ID() != 2 || a.Slot != 0 || !near(a.EndTime, 12.9) {
		t.Fatalf("ожидали И1-8 из места 1 на П2 до 12,90, получили %s из %d на П%d до %.2f",
			a.Request.Label(), a.Slot+1, a.Device.ID(), a.EndTime)
	}
	if !near(a.Request.TimeInBuffer(), 1.1) {
		t.Fatalf("Тбп(И1-8) должно быть 1,10, получили %.3f", a.Request.TimeInBuffer())
	}
	if dv.GetTakePointer() != 1 {
		t.Fatalf("УВ должен стать 2")
	}
	if got := labels(buf); !eqStrings(got, []string{"—", "И1-7", "И3-4"}) {
		t.Fatalf("буфер в итоге: %v", got)
	}
	if e := cal.PollNext(); e.Type != engine.ServiceEnd || e.DeviceID != 2 || !near(e.Time, 12.9) {
		t.Fatalf("в календаре должно быть SERVICE_END П2 в 12,90, а есть %v", e)
	}
}

// Краевой кейс 1: постановка с переходом через конец кольца (Д1ОЗ1).
func TestPlacementWrapsAroundRing(t *testing.T) {
	buf := smo.NewBuffer(4)
	st := stats.NewStatisticsCollector(3, nil)
	dp := NewPlacementDispatcher(buf, st)
	putAt(buf, 1, req(1, 3, 1), 1)
	putAt(buf, 2, req(2, 2, 2), 2)
	putAt(buf, 3, req(3, 1, 3), 3)
	dp.putPointer = 1 // УП = 2

	res := dp.Place(req(2, 3, 5), 5)
	if res.Slot != 0 || res.HasEviction() {
		t.Fatalf("ожидали место 1 без выбивания, получили %+v", res)
	}
	if dp.GetPutPointer() != 1 {
		t.Fatalf("УП должен стать 2")
	}
}

// Краевой кейс 2: выбивается самая старая, а не заявка под указателем (Д1ОО3 ≠ Д1ОО1).
func TestEvictsOldestNotUnderPointer(t *testing.T) {
	buf := smo.NewBuffer(4)
	st := stats.NewStatisticsCollector(3, nil)
	dp := NewPlacementDispatcher(buf, st)
	putAt(buf, 0, req(2, 3, 5.1), 5.1)
	putAt(buf, 1, req(1, 3, 2.0), 2.0)
	putAt(buf, 2, req(2, 2, 3.4), 3.4)
	putAt(buf, 3, req(3, 1, 4.0), 4.0)
	dp.putPointer = 1 // УП = 2

	res := dp.Place(req(3, 2, 6.0), 6.0)
	if res.Evicted.Label() != "И1-3" || res.Slot != 1 {
		t.Fatalf("ожидали выбивание И1-3 из места 2, получили %s из %d", res.Evicted.Label(), res.Slot+1)
	}
	if dp.GetPutPointer() != 2 {
		t.Fatalf("УП должен стать 3")
	}
}

// Краевой кейс 3: выборка по кольцу с переходом через конец (Д2Б3).
func TestSelectionWrapsAroundRing(t *testing.T) {
	buf := smo.NewBuffer(4)
	devices := []*smo.Device{smo.NewDevice(1, dist.NewFixed(9)), smo.NewDevice(2, dist.NewFixed(9)), smo.NewDevice(3, dist.NewFixed(1))}
	cal := engine.NewEventCalendar()
	st := stats.NewStatisticsCollector(3, devices)
	dv := NewSelectionDispatcher(buf, devices, cal, st)
	devices[0].StartService(req(1, 1, 0), 0)
	devices[1].StartService(req(1, 2, 0), 0)
	putAt(buf, 0, req(2, 3, 1), 1)
	putAt(buf, 3, req(3, 1, 2), 2)
	dv.takePointer = 1 // УВ = 2

	as := dv.OnDeviceReleased(3)
	if len(as) != 1 || as[0].Request.Label() != "И3-1" || as[0].Slot != 3 || as[0].Device.ID() != 3 {
		t.Fatalf("ожидали И3-1 из места 4 на П3, получили %+v", as)
	}
	if dv.GetTakePointer() != 0 {
		t.Fatalf("УВ должен перейти на место 1")
	}
}

// Краевой кейс 4: при нескольких свободных приборах выбирается П1 (Д2П1).
func TestDevicePriority(t *testing.T) {
	buf := smo.NewBuffer(4)
	devices := []*smo.Device{smo.NewDevice(1, dist.NewFixed(1)), smo.NewDevice(2, dist.NewFixed(9)), smo.NewDevice(3, dist.NewFixed(1))}
	cal := engine.NewEventCalendar()
	st := stats.NewStatisticsCollector(3, devices)
	dv := NewSelectionDispatcher(buf, devices, cal, st)
	devices[1].StartService(req(2, 1, 0), 0) // П2 занят
	putAt(buf, 0, req(1, 9, 1), 1)

	as := dv.OnRequestBuffered(1)
	if len(as) != 1 || as[0].Device.ID() != 1 {
		t.Fatalf("ожидали постановку на П1, получили %+v", as)
	}
	if !devices[2].IsFree() {
		t.Fatalf("П3 не должен рассматриваться")
	}
}

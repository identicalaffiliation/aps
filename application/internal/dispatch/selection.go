package dispatch

import (
	"smo-alarms/internal/engine"
	"smo-alarms/internal/model"
	"smo-alarms/internal/smo"
	"smo-alarms/internal/stats"
)

// Assignment — факт постановки заявки на прибор (для журнала пошагового режима).
type Assignment struct {
	Request *model.Request
	Device  *smo.Device
	Slot    int     // из какого места буфера взята
	EndTime float64 // когда прибор закончит
}

// SelectionDispatcher — ДВ, планировщик проверки тревог.
// Триггеры: «заявка поставлена в буфер» и «прибор освободился».
type SelectionDispatcher struct {
	buffer      *smo.Buffer
	devices     []*smo.Device
	calendar    *engine.EventCalendar
	statistics  *stats.StatisticsCollector
	takePointer int // УВ
}

func NewSelectionDispatcher(buffer *smo.Buffer, devices []*smo.Device,
	calendar *engine.EventCalendar, statistics *stats.StatisticsCollector) *SelectionDispatcher {
	return &SelectionDispatcher{buffer: buffer, devices: devices, calendar: calendar, statistics: statistics}
}

// OnRequestBuffered — триггер №1: ДП поставил заявку в буфер.
func (d *SelectionDispatcher) OnRequestBuffered(currentTime float64) []Assignment {
	return d.dispatch(currentTime)
}

// OnDeviceReleased — триггер №2: прибор освободился (SERVICE_END).
func (d *SelectionDispatcher) OnDeviceReleased(currentTime float64) []Assignment {
	return d.dispatch(currentTime)
}

// dispatch — общий цикл: пока есть свободный прибор и буфер не пуст,
// взять прибор по Д2П1, заявку по Д2Б3 и поставить заявку на прибор.
func (d *SelectionDispatcher) dispatch(currentTime float64) []Assignment {
	var out []Assignment
	for {
		dev := d.findFreeDeviceByPriority()
		if dev == nil {
			break // все приборы заняты — заявки ждут SERVICE_END
		}
		req, slot := d.takeRequestByRing()
		if req == nil {
			break // буфер пуст — прибор простаивает
		}
		d.statistics.RecordServiceStart(req, currentTime)
		end := dev.StartService(req, currentTime) // ПЗ2
		d.calendar.Schedule(engine.Event{Time: end, Type: engine.ServiceEnd, DeviceID: dev.ID()})
		out = append(out, Assignment{Request: req, Device: dev, Slot: slot, EndTime: end})
	}
	return out
}

// findFreeDeviceByPriority — Д2П1: перебор всегда с П1, первый свободный.
func (d *SelectionDispatcher) findFreeDeviceByPriority() *smo.Device {
	for _, dev := range d.devices {
		if dev.IsFree() {
			return dev
		}
	}
	return nil
}

// takeRequestByRing — Д2Б3: первое занятое место по кольцу с УВ; УВ := (q + 1) mod N.
func (d *SelectionDispatcher) takeRequestByRing() (*model.Request, int) {
	q := d.buffer.FindOccupiedSlotByRing(d.takePointer)
	if q < 0 {
		return nil, -1
	}
	r := d.buffer.RemoveAt(q)
	d.takePointer = (q + 1) % d.buffer.GetCapacity()
	return r, q
}

// GetTakePointer — положение УВ для пошагового режима.
func (d *SelectionDispatcher) GetTakePointer() int { return d.takePointer }

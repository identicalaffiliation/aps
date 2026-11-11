package smo

import (
	"smo-alarms/internal/dist"
	"smo-alarms/internal/model"
)

// Device — прибор: рабочий процесс анализа тревог. Обслуживает одну заявку за раз.
type Device struct {
	id            int
	serviceTime   dist.Distribution // ПЗ2
	current       *model.Request
	serviceStart  float64
	plannedEnd    float64
	busyTimeTotal float64
}

func NewDevice(id int, serviceTime dist.Distribution) *Device {
	return &Device{id: id, serviceTime: serviceTime}
}

func (d *Device) IsFree() bool { return d.current == nil }

// StartService ставит заявку на проверку и возвращает момент окончания (ПЗ2).
func (d *Device) StartService(request *model.Request, currentTime float64) float64 {
	if d.current != nil {
		panic("device: прибор занят")
	}
	d.current = request
	d.serviceStart = currentTime
	d.plannedEnd = currentTime + d.serviceTime.Next()
	request.MarkInService(currentTime)
	return d.plannedEnd
}

// FinishService завершает проверку и возвращает обслуженную заявку.
func (d *Device) FinishService(currentTime float64) *model.Request {
	r := d.current
	if r == nil {
		panic("device: прибор свободен")
	}
	d.busyTimeTotal += currentTime - d.serviceStart
	r.MarkServed(currentTime)
	d.current = nil
	return r
}

func (d *Device) GetCurrent() *model.Request { return d.current }
func (d *Device) PlannedEnd() float64        { return d.plannedEnd }
func (d *Device) GetBusyTimeTotal() float64  { return d.busyTimeTotal }
func (d *Device) ID() int                    { return d.id }

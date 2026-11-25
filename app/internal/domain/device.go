package domain

type Device struct {
	ID    int // number of device
	Curr  *Request
	EndAt float64
}

func NewDevice(id int) *Device {
	return &Device{ID: id, Curr: nil, EndAt: 0}
}

func (d *Device) IsFree() bool {
	return d.Curr == nil
}

func (d *Device) Start(r *Request, now, duration float64) float64 {
	r.MarkInService(now)
	d.Curr = r
	d.EndAt = now + duration
	return d.EndAt
}

func (d *Device) Finish(now float64) *Request {
	d.Curr.MarkServed(now)
	t := d.Curr
	d.Curr = nil
	return t
}

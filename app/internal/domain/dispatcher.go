package domain

import "sort"

type PlacementDispatcher struct {
	buf     *Buffer
	pointer int // put pointer
}

func NewPlacementDispatcher(b *Buffer) *PlacementDispatcher {
	return &PlacementDispatcher{buf: b, pointer: 0}
}

func (pd *PlacementDispatcher) Place(r *Request, t float64) (int, *Request) {
	freeAt := pd.buf.FindFreeSlot(pd.pointer)
	var evicted *Request
	if freeAt == -1 {
		freeAt = pd.buf.FindOldestSlot()
		evicted = pd.buf.RemoveAt(freeAt)
		evicted.Reject(t)
	}

	// [1, 2, 3]
	// [1, nil, 3]
	// [1, 4, 3]

	r.MarkBuffered(t)
	pd.buf.PutAt(r, freeAt)
	pd.pointer = (freeAt + 1) % len(pd.buf.slots)
	return freeAt, evicted
}

type SelectionDispatcher struct {
	Devices []*Device
	Buf     *Buffer
	Pointer int // take pointer
}

func NewSelectionDispatcher(buf *Buffer, devices ...*Device) *SelectionDispatcher {
	d := append(make([]*Device, 0, len(devices)), devices...)
	sort.Slice(d, func(i, j int) bool {
		return d[i].ID < d[j].ID
	})

	return &SelectionDispatcher{Devices: d, Pointer: 0, Buf: buf}
}

func (sd *SelectionDispatcher) DispatchOne(now, duration float64) (*Device, float64) {
	for _, device := range sd.Devices {
		if device.IsFree() {
			index := sd.Buf.FindOccupiedSlot(sd.Pointer)
			if index == -1 {
				return nil, 0
			}

			req := sd.Buf.RemoveAt(index)
			sd.Pointer = (index + 1) % len(sd.Buf.slots)
			return device, device.Start(req, now, duration)
		}
	}

	return nil, 0
}

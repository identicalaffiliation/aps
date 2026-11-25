package domain

import (
	"testing"
)

func TestSelectionDispatcher_DispatchOne(t *testing.T) {
	t.Parallel()

	buf := NewBuffer(3)
	buf.slots[1] = new(Request)
	sd := NewSelectionDispatcher(buf, NewDevice(1), NewDevice(2))
	device, end := sd.DispatchOne(3.35, 1.43) // 4.78 end
	if device != nil && device.ID != 1 {
		t.Fatalf("invalid device id, expected: %d, got: %d", 1, device.ID)
	}

	if sd.Buf.slots[0] != nil {
		t.Fatal("not empty slot")
	}

	if sd.Pointer != 2 {
		t.Fatalf("invalid take pointer, expected: %d, got: %d", 2, sd.Pointer)
	}

	if end != 4.78 {
		t.Fatal("invalid end time")
	}
}

func TestPlacementDispatcher_Place(t *testing.T) {
	t.Parallel()

	buf := loadData(t)
	dp := NewPlacementDispatcher(buf)

	r := &Request{ID: 1, Status: Created}
	i, e := dp.Place(r, 10.00)
	if dp.pointer != 2 {
		t.Fatalf("invalid dp put pointer, expected: %d, got: %d", 2, dp.pointer)
	}

	if i != 1 {
		t.Fatalf("invalid put at index, expected: %d, got: %d", 1, i)
	}

	if e.FinishedAt != 10.00 {
		t.Fatalf("invalid finished at of evicted req, expected: %f, got: %f", 10.00, e.FinishedAt)
	}

	if dp.buf.slots[1].ID != r.ID {
		t.Fatalf("invalid id of put at req, expected: %d, got: %d", r.ID, dp.buf.slots[1].ID)
	}
}

func TestSelectionDispatcher_New(t *testing.T) {
	t.Parallel()

	d1 := NewDevice(2)
	d2 := NewDevice(0)

	sp := NewSelectionDispatcher(NewBuffer(0), d1, d2)
	if sp.Devices[0].ID != 0 {
		t.Fatalf("invalid id of device, expected: %d, got: %d", 0, sp.Devices[0].ID)
	}

	if sp.Devices[1].ID != 2 {
		t.Fatalf("invalid id of device, expected: %d, got: %d", 2, sp.Devices[0].ID)
	}
}

func loadData(t *testing.T) *Buffer {
	t.Helper()
	b := NewBuffer(3)
	b.PutAt(&Request{BufferedAt: 8.00}, 0)
	b.PutAt(&Request{BufferedAt: 7.20}, 1)
	b.PutAt(&Request{BufferedAt: 9.50}, 2)
	return b
}

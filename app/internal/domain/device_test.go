package domain

import "testing"

func TestDevice_HappyFlow(t *testing.T) {
	t.Parallel()

	device := NewDevice(1)
	request := &Request{
		Status: Created,
	}

	request.MarkBuffered(0.84)
	if device.Start(request, 0.84, 2.70) != 3.54 {
		t.Fatalf("invalid end time, expected: %f, got: %f", 3.54, device.EndAt)
	}

	if device.IsFree() {
		t.Fatal("device is not available, but available")
	}

	if request.Status != InService {
		t.Fatalf("invalid request status, expected: %d, got: %d", InService, request.Status)
	}

	free := device.Finish(3.54)
	if free.Status != Served && request.FinishedAt == 3.54 {
		t.Fatalf("invalid status, expected: %d, got: %d", Served, free.Status)
	}
}

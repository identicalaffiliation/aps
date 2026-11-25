package domain

import "testing"

func TestRequestServiceLifecycle(t *testing.T) {
	t.Parallel()

	r := &Request{}

	r.MarkBuffered(0.84)
	if r.BufferedAt != 0.84 {
		t.Fatal("invalid buffered at")
	}

	if r.Status != InBuffer {
		t.Fatal("invalid status")
	}

	r.MarkInService(0.84)
	if r.ServiceStartAt != 0.84 {
		t.Fatal("invalid service at")
	}

	if r.Status != InService {
		t.Fatal("invalid status")
	}

	r.MarkServed(3.54)
	if r.FinishedAt != 3.54 {
		t.Fatal("invalid finished at")
	}

	if r.Status != Served {
		t.Fatal("invalid status")
	}
}

func TestRequestRejectedFromBuffer(t *testing.T) {
	t.Parallel()

	r := &Request{}

	r.MarkBuffered(7.20)

	r.Reject(10.00)
	if r.FinishedAt != 10.00 && r.BufferedAt != 7.20 {
		t.Fatal("invalid finished at")
	}

	if r.Status != Rejected {
		t.Fatal("invalid status")
	}
}

package domain

const (
	Created Status = iota
	InBuffer
	InService
	Served
	Rejected
)

type Status int

type Request struct {
	ID       int
	SourceID int
	Number   int

	Type     string
	ObjectID string

	GeneratedAt    float64
	BufferedAt     float64
	ServiceStartAt float64
	FinishedAt     float64

	Status Status
}

func (r *Request) MarkBuffered(t float64) {
	r.Status = InBuffer
	r.BufferedAt = t
}

func (r *Request) MarkInService(t float64) {
	r.Status = InService
	r.ServiceStartAt = t
}

func (r *Request) MarkServed(t float64) {
	r.Status = Served
	r.FinishedAt = t
}

func (r *Request) Reject(t float64) {
	r.Status = Rejected
	r.FinishedAt = t
}

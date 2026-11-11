// Package model — модель предметной области: заявка = тревожное сообщение.
package model

import "fmt"

// RequestStatus — жизненный цикл заявки:
// CREATED → IN_BUFFER → IN_SERVICE → SERVED, либо IN_BUFFER → REJECTED (Д1ОО3).
type RequestStatus int

const (
	Created RequestStatus = iota
	InBuffer
	InService
	Served
	Rejected
)

func (s RequestStatus) String() string {
	return [...]string{"CREATED", "IN_BUFFER", "IN_SERVICE", "SERVED", "REJECTED"}[s]
}

// AlarmType — тип тревоги в бизнес-домене.
type AlarmType int

const (
	ThresholdExceeded AlarmType = iota // превышение порога (датчики)
	EquipmentFault                     // неисправность (контроллеры линий)
	AccessViolation                    // нарушение режима (СКУД)
)

func (a AlarmType) String() string {
	return [...]string{"превышение порога", "неисправность", "нарушение доступа"}[a]
}

// Request — заявка (тревожное сообщение).
type Request struct {
	id               int64
	sourceID         int
	number           int // порядковый номер у своего источника: И<sourceID>-<number>
	alarmType        AlarmType
	objectID         string
	generationTime   float64
	bufferPutTime    float64
	serviceStartTime float64
	finishTime       float64
	status           RequestStatus
}

// NewRequest создаёт тревогу в момент генерации.
func NewRequest(id int64, sourceID, number int, alarmType AlarmType, objectID string, generationTime float64) *Request {
	return &Request{
		id: id, sourceID: sourceID, number: number, alarmType: alarmType,
		objectID: objectID, generationTime: generationTime, status: Created,
	}
}

func (r *Request) MarkBuffered(t float64) {
	r.bufferPutTime = t
	r.status = InBuffer
}

func (r *Request) MarkInService(t float64) {
	r.serviceStartTime = t
	r.status = InService
}

func (r *Request) MarkServed(t float64) {
	r.finishTime = t
	r.status = Served
}

// Reject — заявка вытеснена из буфера без обслуживания (Д1ОО3).
func (r *Request) Reject(t float64) {
	r.finishTime = t
	r.status = Rejected
}

// TimeInBuffer — Тбп. Для обслуженной: до начала обслуживания, для отказанной: до выбивания.
func (r *Request) TimeInBuffer() float64 {
	switch r.status {
	case InService, Served:
		return r.serviceStartTime - r.bufferPutTime
	case Rejected:
		return r.finishTime - r.bufferPutTime
	}
	return 0
}

// ServiceTime — Тобсл (только для обслуженных).
func (r *Request) ServiceTime() float64 {
	if r.status != Served {
		return 0
	}
	return r.finishTime - r.serviceStartTime
}

// TimeInSystem — Тпреб = момент выхода − момент генерации.
func (r *Request) TimeInSystem() float64 { return r.finishTime - r.generationTime }

func (r *Request) ID() int64               { return r.id }
func (r *Request) SourceID() int           { return r.sourceID }
func (r *Request) AlarmType() AlarmType    { return r.alarmType }
func (r *Request) ObjectID() string        { return r.objectID }
func (r *Request) GenerationTime() float64 { return r.generationTime }
func (r *Request) BufferPutTime() float64  { return r.bufferPutTime }
func (r *Request) Status() RequestStatus   { return r.status }

// Label — короткое имя для вывода, как на диаграммах: «И1-8».
func (r *Request) Label() string { return fmt.Sprintf("И%d-%d", r.sourceID, r.number) }

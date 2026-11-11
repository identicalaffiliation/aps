// Package stats — сбор статистики и сводная таблица ОР1.
package stats

import (
	"smo-alarms/internal/model"
	"smo-alarms/internal/smo"
)

// SourceStats — строка сводной таблицы по источнику.
type SourceStats struct {
	SourceID          int
	Generated         int64
	Served            int64
	Rejected          int64
	RejectProbability float64 // Pотк
	AvgTimeInSystem   float64 // Тпреб = Тбп + Тобсл
	AvgTimeInBuffer   float64 // Тбп
	AvgServiceTime    float64 // Тобсл
	VarTimeInBuffer   float64 // Дбп
	VarServiceTime    float64 // Добсл
}

// DeviceStats — строка сводной таблицы по прибору.
type DeviceStats struct {
	DeviceID    int
	Utilization float64 // Кисп
}

// ResultsTable — ОР1: сводная таблица результатов.
type ResultsTable struct {
	SourceRows []SourceStats
	DeviceRows []DeviceStats
	TotalTime  float64
	Total      SourceStats // итог по всем источникам (SourceID = 0)
}

// StatisticsCollector накапливает показатели. Вызывается в строго определённых точках:
// генерация, выбивание, постановка на прибор, окончание обслуживания.
type StatisticsCollector struct {
	generated, rejected, served []int64
	// Тбп копится и для обслуженных, и для отказанных: обе заявки реально стояли в буфере.
	sumTimeInBuffer, sumSqTimeInBuffer []float64
	waited                             []int64
	sumServiceTime, sumSqServiceTime   []float64
	devices                            []*smo.Device
}

func NewStatisticsCollector(sources int, devices []*smo.Device) *StatisticsCollector {
	return &StatisticsCollector{
		generated: make([]int64, sources), rejected: make([]int64, sources), served: make([]int64, sources),
		sumTimeInBuffer: make([]float64, sources), sumSqTimeInBuffer: make([]float64, sources),
		waited:         make([]int64, sources),
		sumServiceTime: make([]float64, sources), sumSqServiceTime: make([]float64, sources),
		devices: devices,
	}
}

func (s *StatisticsCollector) RecordGenerated(r *model.Request) { s.generated[r.SourceID()-1]++ }

// RecordRejected вызывается ДП при выбивании (Д1ОО3).
func (s *StatisticsCollector) RecordRejected(r *model.Request) {
	i := r.SourceID() - 1
	s.rejected[i]++
	s.addWait(i, r.TimeInBuffer())
}

// RecordServiceStart вызывается ДВ при постановке заявки на прибор.
func (s *StatisticsCollector) RecordServiceStart(r *model.Request, time float64) {
	s.addWait(r.SourceID()-1, time-r.BufferPutTime())
}

// RecordServed вызывается при окончании обслуживания (SERVICE_END).
func (s *StatisticsCollector) RecordServed(r *model.Request) {
	i := r.SourceID() - 1
	t := r.ServiceTime()
	s.served[i]++
	s.sumServiceTime[i] += t
	s.sumSqServiceTime[i] += t * t
}

func (s *StatisticsCollector) addWait(i int, w float64) {
	s.waited[i]++
	s.sumTimeInBuffer[i] += w
	s.sumSqTimeInBuffer[i] += w * w
}

func (s *StatisticsCollector) GetTotalGenerated() int64     { return sum(s.generated) }
func (s *StatisticsCollector) GetTotalRejected() int64      { return sum(s.rejected) }
func (s *StatisticsCollector) GetTotalServed() int64        { return sum(s.served) }
func (s *StatisticsCollector) Generated(sourceID int) int64 { return s.generated[sourceID-1] }
func (s *StatisticsCollector) Rejected(sourceID int) int64  { return s.rejected[sourceID-1] }

// GetRejectProbability — текущая общая Pотк.
func (s *StatisticsCollector) GetRejectProbability() float64 {
	g := s.GetTotalGenerated()
	if g == 0 {
		return 0
	}
	return float64(s.GetTotalRejected()) / float64(g)
}

// BuildResults формирует сводную таблицу ОР1. totalTime — время реализации.
func (s *StatisticsCollector) BuildResults(totalTime float64) *ResultsTable {
	t := &ResultsTable{TotalTime: totalTime}
	for i := range s.generated {
		t.SourceRows = append(t.SourceRows, row(i+1, s.generated[i], s.served[i], s.rejected[i],
			s.waited[i], s.sumTimeInBuffer[i], s.sumSqTimeInBuffer[i], s.sumServiceTime[i], s.sumSqServiceTime[i]))
	}
	t.Total = row(0, sum(s.generated), sum(s.served), sum(s.rejected), sum(s.waited),
		sumF(s.sumTimeInBuffer), sumF(s.sumSqTimeInBuffer), sumF(s.sumServiceTime), sumF(s.sumSqServiceTime))
	for _, d := range s.devices {
		u := 0.0
		if totalTime > 0 {
			u = d.GetBusyTimeTotal() / totalTime
		}
		t.DeviceRows = append(t.DeviceRows, DeviceStats{DeviceID: d.ID(), Utilization: u})
	}
	return t
}

// AvgUtilization — средний Кисп по приборам.
func (t *ResultsTable) AvgUtilization() float64 {
	if len(t.DeviceRows) == 0 {
		return 0
	}
	s := 0.0
	for _, d := range t.DeviceRows {
		s += d.Utilization
	}
	return s / float64(len(t.DeviceRows))
}

func row(id int, gen, served, rej, waited int64, sw, sw2, ss, ss2 float64) SourceStats {
	r := SourceStats{SourceID: id, Generated: gen, Served: served, Rejected: rej}
	if gen > 0 {
		r.RejectProbability = float64(rej) / float64(gen)
	}
	r.AvgTimeInBuffer, r.VarTimeInBuffer = meanVar(sw, sw2, waited)
	r.AvgServiceTime, r.VarServiceTime = meanVar(ss, ss2, served)
	// Тпреб = Тбп + Тобсл (как в методичке).
	r.AvgTimeInSystem = r.AvgTimeInBuffer + r.AvgServiceTime
	return r
}

// meanVar — M[x] и D = M[x²] − (M[x])².
func meanVar(s, s2 float64, n int64) (float64, float64) {
	if n == 0 {
		return 0, 0
	}
	m := s / float64(n)
	d := s2/float64(n) - m*m
	if d < 0 {
		d = 0
	}
	return m, d
}

func sum(a []int64) (s int64) {
	for _, v := range a {
		s += v
	}
	return
}

func sumF(a []float64) (s float64) {
	for _, v := range a {
		s += v
	}
	return
}

// Package sim — движок метода особых событий.
package sim

import (
	"fmt"
	"math/rand"

	"smo-alarms/internal/dispatch"
	"smo-alarms/internal/dist"
	"smo-alarms/internal/engine"
	"smo-alarms/internal/smo"
	"smo-alarms/internal/stats"
)

// SimulationConfig — входные параметры модели.
type SimulationConfig struct {
	SourceLambdas  []float64 // λi для ИЗ1
	DeviceCount    int       // m
	ServiceMin     float64   // a для ПЗ2
	ServiceMax     float64   // b для ПЗ2
	BufferCapacity int       // N мест
	RequestsLimit  int64     // сколько заявок сгенерировать
	Seed           int64
}

func (c SimulationConfig) Validate() error {
	switch {
	case len(c.SourceLambdas) == 0:
		return fmt.Errorf("нужен хотя бы один источник")
	case c.DeviceCount <= 0:
		return fmt.Errorf("число приборов должно быть > 0")
	case c.BufferCapacity <= 0:
		return fmt.Errorf("размер буфера должен быть > 0")
	case c.ServiceMin < 0 || c.ServiceMax < c.ServiceMin:
		return fmt.Errorf("нужно 0 ≤ a ≤ b для ПЗ2")
	case c.RequestsLimit <= 0:
		return fmt.Errorf("число заявок должно быть > 0")
	}
	for _, l := range c.SourceLambdas {
		if l <= 0 {
			return fmt.Errorf("интенсивность λ должна быть > 0")
		}
	}
	return nil
}

// DeviceState — состояние прибора в снимке ОД1.
type DeviceState struct {
	ID      int
	Request string // "" — свободен
	Until   float64
}

// StepSnapshot — одна строка пошагового режима ОД1.
type StepSnapshot struct {
	Step          int
	Time          float64
	Event         engine.Event
	Actions       []string // что произошло: сгенерирована, в буфер, выбита, выбрана, на прибор, обработана
	Calendar      []engine.Event
	BufferSlots   []string // "" — свободно
	PutPointer    int
	TakePointer   int
	Devices       []DeviceState
	Generated     int64
	Rejected      int64
	RejectPercent float64
}

// Simulator — владеет всеми объектами модели и передаёт события диспетчерам.
type Simulator struct {
	config              SimulationConfig
	currentTime         float64
	calendar            *engine.EventCalendar
	sources             []*smo.Source
	buffer              *smo.Buffer
	devices             []*smo.Device
	placementDispatcher *dispatch.PlacementDispatcher
	selectionDispatcher *dispatch.SelectionDispatcher
	statistics          *stats.StatisticsCollector
	steps               int
}

// New собирает модель с распределениями ИЗ1 и ПЗ2.
func New(cfg SimulationConfig) (*Simulator, error) {
	if err := cfg.Validate(); err != nil {
		return nil, err
	}
	rng := rand.New(rand.NewSource(cfg.Seed))
	intervals := make([]dist.Distribution, len(cfg.SourceLambdas))
	for i, l := range cfg.SourceLambdas {
		intervals[i] = dist.NewExponential(l, rng)
	}
	services := make([]dist.Distribution, cfg.DeviceCount)
	for j := range services {
		services[j] = dist.NewUniform(cfg.ServiceMin, cfg.ServiceMax, rng)
	}
	return NewWithDistributions(cfg, intervals, services), nil
}

// NewWithDistributions — сборка с заданными распределениями (используется в тестах).
func NewWithDistributions(cfg SimulationConfig, intervals, services []dist.Distribution) *Simulator {
	s := &Simulator{config: cfg, calendar: engine.NewEventCalendar(), buffer: smo.NewBuffer(cfg.BufferCapacity)}
	ids := &smo.IDGenerator{}
	for i, d := range intervals {
		s.sources = append(s.sources, smo.NewSource(i+1, d, ids))
	}
	for j, d := range services {
		s.devices = append(s.devices, smo.NewDevice(j+1, d))
	}
	s.statistics = stats.NewStatisticsCollector(len(s.sources), s.devices)
	s.placementDispatcher = dispatch.NewPlacementDispatcher(s.buffer, s.statistics)
	s.selectionDispatcher = dispatch.NewSelectionDispatcher(s.buffer, s.devices, s.calendar, s.statistics)
	s.Init()
	return s
}

// Init — t = 0; для каждого источника запланировать первое GENERATION по ИЗ1.
func (s *Simulator) Init() {
	for _, src := range s.sources {
		s.calendar.Schedule(engine.Event{Time: src.NextArrivalInterval(), Type: engine.Generation, SourceID: src.ID()})
	}
}

func (s *Simulator) IsFinished() bool { return s.calendar.IsEmpty() }

// Step обрабатывает ближайшее особое событие и возвращает снимок ОД1.
func (s *Simulator) Step() *StepSnapshot {
	if s.IsFinished() {
		return nil
	}
	e := s.calendar.PollNext()
	s.currentTime = e.Time
	s.steps++
	var actions []string
	switch e.Type {
	case engine.Generation:
		actions = s.handleGeneration(e)
	case engine.ServiceEnd:
		actions = s.handleServiceEnd(e)
	}
	return s.snapshot(e, actions)
}

// RunAuto — автоматический режим: прогон до пустого календаря, затем ОР1.
func (s *Simulator) RunAuto() *stats.ResultsTable {
	for !s.IsFinished() {
		e := s.calendar.PollNext()
		s.currentTime = e.Time
		s.steps++
		if e.Type == engine.Generation {
			s.handleGeneration(e)
		} else {
			s.handleServiceEnd(e)
		}
	}
	return s.Results()
}

func (s *Simulator) Results() *stats.ResultsTable { return s.statistics.BuildResults(s.currentTime) }

// handleGeneration — триггер ДП: сгенерирована тревога.
func (s *Simulator) handleGeneration(e engine.Event) []string {
	if s.statistics.GetTotalGenerated() >= s.config.RequestsLimit {
		return []string{fmt.Sprintf("И%d: лимит %d заявок достигнут, генерация остановлена", e.SourceID, s.config.RequestsLimit)}
	}
	src := s.sources[e.SourceID-1]
	req := src.Generate(s.currentTime)
	s.statistics.RecordGenerated(req)
	actions := []string{fmt.Sprintf("сгенерирована %s (%s)", req.Label(), req.AlarmType())}

	if s.statistics.GetTotalGenerated() < s.config.RequestsLimit { // ИБ: планируем следующую по ИЗ1
		s.calendar.Schedule(engine.Event{Time: s.currentTime + src.NextArrivalInterval(), Type: engine.Generation, SourceID: src.ID()})
	}

	res := s.placementDispatcher.Place(req, s.currentTime)
	if res.HasEviction() {
		actions = append(actions, fmt.Sprintf("буфер полон: выбита самая старая %s из места %d (ОТКАЗ, Д1ОО3)", res.Evicted.Label(), res.Slot+1))
	}
	actions = append(actions, fmt.Sprintf("%s поставлена в буфер, место %d (Д1ОЗ1), УП → %d", req.Label(), res.Slot+1, s.placementDispatcher.GetPutPointer()+1))

	assigns := s.selectionDispatcher.OnRequestBuffered(s.currentTime)
	return append(actions, describe(assigns)...)
}

// handleServiceEnd — прибор закончил проверку, триггер ДВ «прибор освободился».
func (s *Simulator) handleServiceEnd(e engine.Event) []string {
	dev := s.devices[e.DeviceID-1]
	req := dev.FinishService(s.currentTime)
	s.statistics.RecordServed(req)
	actions := []string{fmt.Sprintf("П%d обработал %s, Тпреб = %.3f", dev.ID(), req.Label(), req.TimeInSystem())}
	assigns := s.selectionDispatcher.OnDeviceReleased(s.currentTime)
	if len(assigns) == 0 {
		actions = append(actions, fmt.Sprintf("буфер пуст: П%d простаивает", dev.ID()))
	}
	return append(actions, describe(assigns)...)
}

func describe(as []dispatch.Assignment) []string {
	var out []string
	for _, a := range as {
		out = append(out,
			fmt.Sprintf("выбрана %s из места %d (Д2Б3)", a.Request.Label(), a.Slot+1),
			fmt.Sprintf("%s отправлена на П%d (Д2П1), окончание в %.3f (ПЗ2)", a.Request.Label(), a.Device.ID(), a.EndTime))
	}
	return out
}

func (s *Simulator) snapshot(e engine.Event, actions []string) *StepSnapshot {
	snap := &StepSnapshot{
		Step: s.steps, Time: s.currentTime, Event: e, Actions: actions,
		Calendar:    s.calendar.View(),
		PutPointer:  s.placementDispatcher.GetPutPointer(),
		TakePointer: s.selectionDispatcher.GetTakePointer(),
		Generated:   s.statistics.GetTotalGenerated(),
		Rejected:    s.statistics.GetTotalRejected(),
	}
	snap.RejectPercent = 100 * s.statistics.GetRejectProbability()
	for _, r := range s.buffer.Snapshot() {
		if r == nil {
			snap.BufferSlots = append(snap.BufferSlots, "")
		} else {
			snap.BufferSlots = append(snap.BufferSlots, r.Label())
		}
	}
	for _, d := range s.devices {
		st := DeviceState{ID: d.ID()}
		if r := d.GetCurrent(); r != nil {
			st.Request, st.Until = r.Label(), d.PlannedEnd()
		}
		snap.Devices = append(snap.Devices, st)
	}
	return snap
}

// Доступ для тестов и отображения.
func (s *Simulator) Buffer() *smo.Buffer                    { return s.buffer }
func (s *Simulator) Devices() []*smo.Device                 { return s.devices }
func (s *Simulator) Statistics() *stats.StatisticsCollector { return s.statistics }
func (s *Simulator) CurrentTime() float64                   { return s.currentTime }
func (s *Simulator) Config() SimulationConfig               { return s.config }
func (s *Simulator) SourceCount() int                       { return len(s.sources) }

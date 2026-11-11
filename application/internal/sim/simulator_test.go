package sim

import (
	"math"
	"testing"

	"smo-alarms/internal/dist"
	"smo-alarms/internal/engine"
)

func near(a, b float64) bool { return math.Abs(a-b) < 1e-9 }

// Сиквенс Happy Flow (sequence_happy_flow): буфер пуст, прибор свободен; те же числа.
func TestHappyFlow(t *testing.T) {
	cfg := SimulationConfig{SourceLambdas: []float64{1, 1, 1}, DeviceCount: 2, ServiceMin: 1, ServiceMax: 3,
		BufferCapacity: 3, RequestsLimit: 100}
	intervals := []dist.Distribution{dist.NewFixed(4.20), dist.NewFixed(0.84, 3.10), dist.NewFixed(5.05)}
	services := []dist.Distribution{dist.NewFixed(2.70), dist.NewFixed(2.0)}
	s := NewWithDistributions(cfg, intervals, services)

	// t = 0,84: GENERATION И2 → место 1 → сразу на П1 до 3,54.
	st := s.Step()
	if st.Event.Type != engine.Generation || st.Event.SourceID != 2 || !near(st.Time, 0.84) {
		t.Fatalf("первым должно быть GENERATION И2 в 0,84, а не %v", st.Event)
	}
	if st.Devices[0].Request != "И2-1" || !near(st.Devices[0].Until, 3.54) || st.Devices[1].Request != "" {
		t.Fatalf("И2-1 должна быть на П1 до 3,54, П2 свободен: %+v", st.Devices)
	}
	if st.PutPointer != 1 || st.TakePointer != 1 || st.BufferSlots[0] != "" {
		t.Fatalf("после шага буфер пуст, УП = УВ = 2: %+v", st)
	}
	if e := st.Calendar; len(e) != 4 || !near(e[0].Time, 3.54) || e[0].Type != engine.ServiceEnd || !near(e[1].Time, 3.94) {
		t.Fatalf("календарь: ожидали SERVICE_END П1 3,54, затем GENERATION И2 3,94: %v", e)
	}

	// t = 3,54: SERVICE_END П1 → буфер пуст, П1 простаивает; Тпреб = 2,70, Тбп = 0.
	st = s.Step()
	if st.Event.Type != engine.ServiceEnd || st.Devices[0].Request != "" {
		t.Fatalf("П1 должен освободиться: %+v", st)
	}
	r := s.Results().Total
	if r.Served != 1 || !near(r.AvgTimeInBuffer, 0) || !near(r.AvgServiceTime, 2.70) {
		t.Fatalf("ожидали 1 обслуженную, Тбп = 0, Тобсл = 2,70: %+v", r)
	}
}

// Инварианты модели на случайном прогоне с перегрузкой (много отказов).
func TestInvariants(t *testing.T) {
	cfg := SimulationConfig{SourceLambdas: []float64{1.5, 1.2, 1.0}, DeviceCount: 2, ServiceMin: 0.5, ServiceMax: 2.5,
		BufferCapacity: 4, RequestsLimit: 5000, Seed: 7}
	s, err := New(cfg)
	if err != nil {
		t.Fatal(err)
	}
	prev := 0.0
	for !s.IsFinished() {
		st := s.Step()
		if st.Time < prev {
			t.Fatalf("время пошло назад: %.3f < %.3f", st.Time, prev)
		}
		prev = st.Time
		// Буфер не пуст ⇒ все приборы заняты.
		if !s.Buffer().IsEmpty() {
			for _, d := range s.Devices() {
				if d.IsFree() {
					t.Fatalf("шаг %d: в буфере есть заявки, а П%d свободен", st.Step, d.ID())
				}
			}
		}
	}
	res := s.Results()
	if res.Total.Generated != cfg.RequestsLimit {
		t.Fatalf("сгенерировано %d, ожидали %d", res.Total.Generated, cfg.RequestsLimit)
	}
	if res.Total.Served+res.Total.Rejected != res.Total.Generated {
		t.Fatalf("обслужено + отказано ≠ сгенерировано: %+v", res.Total)
	}
	if res.Total.Rejected == 0 {
		t.Fatalf("при перегрузке должны быть отказы")
	}
	for _, d := range res.DeviceRows {
		if d.Utilization <= 0 || d.Utilization > 1 {
			t.Fatalf("Кисп вне (0; 1]: %+v", d)
		}
	}
	// Д2П1: П1 загружен не меньше П2.
	if res.DeviceRows[0].Utilization < res.DeviceRows[1].Utilization {
		t.Fatalf("по Д2П1 Кисп П1 ≥ Кисп П2: %+v", res.DeviceRows)
	}
}

// Одинаковый seed — одинаковый результат.
func TestDeterministic(t *testing.T) {
	cfg := SimulationConfig{SourceLambdas: []float64{1, 0.8, 0.6}, DeviceCount: 3, ServiceMin: 0.5, ServiceMax: 1.5,
		BufferCapacity: 5, RequestsLimit: 2000, Seed: 42}
	a, _ := New(cfg)
	b, _ := New(cfg)
	ra, rb := a.RunAuto(), b.RunAuto()
	if ra.Total != rb.Total || ra.TotalTime != rb.TotalTime {
		t.Fatalf("результаты при одном seed различаются")
	}
}

func TestConfigValidation(t *testing.T) {
	bad := SimulationConfig{SourceLambdas: []float64{1}, DeviceCount: 0, BufferCapacity: 1, RequestsLimit: 1}
	if _, err := New(bad); err == nil {
		t.Fatal("ожидали ошибку при 0 приборов")
	}
}

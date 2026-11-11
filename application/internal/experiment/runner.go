// Package experiment — подбор числа реализаций и сравнение конфигураций.
package experiment

import (
	"math"

	"smo-alarms/internal/sim"
	"smo-alarms/internal/stats"
)

const (
	TAlpha = 1.643  // α = 0,9
	Delta  = 0.1    // относительная точность 10 %
	N0     = 100    // пристрелка
	MaxN   = 100000 // потолок N: при p < 0,3 % формула требует миллионы заявок, а на выводы это не влияет
)

// ExperimentRunner — серия прогонов модели.
type ExperimentRunner struct {
	baseConfig sim.SimulationConfig
}

func NewExperimentRunner(base sim.SimulationConfig) *ExperimentRunner {
	return &ExperimentRunner{baseConfig: base}
}

// RequiredN — N = tα²·(1 − p)/(p·δ²).
func RequiredN(p float64) int64 {
	if p <= 0 {
		return 0
	}
	return int64(math.Ceil(TAlpha * TAlpha * (1 - p) / (p * Delta * Delta)))
}

// Iteration — один шаг подбора N.
type Iteration struct {
	N int64
	P float64
}

// FindRequiredRequests: прогон с N₀ = 100 → оценка p → новое N → ... пока p не стабилизируется
// (|p − pпред| < δ·pпред). Если отказов нет, N удваивается до MaxN.
func (r *ExperimentRunner) FindRequiredRequests() (int64, []Iteration) {
	var hist []Iteration
	n, prev := int64(N0), -1.0
	for i := 0; i < 25; i++ {
		res := r.run(n)
		p := res.Total.RejectProbability
		hist = append(hist, Iteration{N: n, P: p})
		if p == 0 {
			// Отказов не было — p не оценить. Удваиваем N; если отказов нет и при MaxN, их практически нет.
			if n >= MaxN {
				return n, hist
			}
			n = min(2*n, MaxN)
			continue
		}
		if prev >= 0 && math.Abs(p-prev) < Delta*prev {
			return n, hist
		}
		next := RequiredN(p)
		if next < N0 {
			next = N0
		}
		if next > MaxN {
			next = MaxN
		}
		if next == n { // упёрлись в потолок или N уже достаточен
			return n, hist
		}
		prev, n = p, next
	}
	return n, hist
}

// CompareConfigs прогоняет каждую конфигурацию в автоматическом режиме.
func (r *ExperimentRunner) CompareConfigs(configs []sim.SimulationConfig) []*stats.ResultsTable {
	out := make([]*stats.ResultsTable, 0, len(configs))
	for _, c := range configs {
		s, err := sim.New(c)
		if err != nil {
			out = append(out, nil)
			continue
		}
		out = append(out, s.RunAuto())
	}
	return out
}

func (r *ExperimentRunner) run(n int64) *stats.ResultsTable {
	c := r.baseConfig
	c.RequestsLimit = n
	s, _ := sim.New(c)
	return s.RunAuto()
}

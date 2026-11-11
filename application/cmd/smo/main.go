// Имитационная модель СМО, вариант 3: система обработки промышленных тревог.
// ИБ · ИЗ1 · ПЗ2 · Д1ОЗ1 · Д1ОО3 · Д2П1 · Д2Б3 · ОР1 · ОД1
package main

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"strconv"
	"strings"
	"text/tabwriter"

	"smo-alarms/internal/experiment"
	"smo-alarms/internal/sim"
	"smo-alarms/internal/ui"
)

// ExperimentMinN — минимальное число заявок на конфигурацию в режиме experiment.
const ExperimentMinN = 10000

func main() {
	mode := flag.String("mode", "step", "режим: step (ОД1), auto (ОР1), experiment (подбор конфигурации)")
	lambdas := flag.String("lambdas", "1.0,0.8,0.6", "интенсивности источников λ через запятую (ИЗ1)")
	devices := flag.Int("devices", 3, "число приборов m")
	a := flag.Float64("a", 0.5, "минимум времени обслуживания (ПЗ2)")
	b := flag.Float64("b", 1.5, "максимум времени обслуживания (ПЗ2)")
	buffer := flag.Int("buffer", 5, "размер буфера N")
	n := flag.Int64("n", 1000, "число генерируемых заявок")
	seed := flag.Int64("seed", 42, "seed генератора случайных чисел")
	steps := flag.Int("steps", 0, "step: вывести первые K шагов без интерактива (0 — интерактивно)")
	deviceCost := flag.Float64("device-cost", 10, "experiment: стоимость прибора за единицу модельного времени")
	rejectCost := flag.Float64("reject-cost", 50, "experiment: стоимость потерянной тревоги")
	flag.Parse()

	cfg := sim.SimulationConfig{
		SourceLambdas: parseLambdas(*lambdas), DeviceCount: *devices,
		ServiceMin: *a, ServiceMax: *b, BufferCapacity: *buffer,
		RequestsLimit: *n, Seed: *seed,
	}
	if err := cfg.Validate(); err != nil {
		fmt.Fprintln(os.Stderr, "ошибка параметров:", err)
		os.Exit(2)
	}
	printConfig(cfg)

	switch *mode {
	case "step":
		runStep(cfg, *steps)
	case "auto":
		runAuto(cfg)
	case "experiment":
		runExperiment(cfg, *deviceCost, *rejectCost)
	default:
		fmt.Fprintln(os.Stderr, "неизвестный режим:", *mode)
		os.Exit(2)
	}
}

func runStep(cfg sim.SimulationConfig, k int) {
	s, _ := sim.New(cfg)
	in := bufio.NewReader(os.Stdin)
	interactive := k == 0
	if interactive {
		fmt.Println("Пошаговый режим (ОД1). Enter — следующий шаг, a — досчитать до конца, q — выход.")
	}
	for i := 0; !s.IsFinished(); i++ {
		if !interactive && i >= k {
			break
		}
		ui.PrintStep(os.Stdout, s.Step())
		if interactive {
			fmt.Print("\n> ")
			line, err := in.ReadString('\n')
			cmd := strings.TrimSpace(line)
			if err != nil || cmd == "q" {
				return
			}
			if cmd == "a" {
				ui.PrintResults(os.Stdout, s.RunAuto())
				return
			}
		}
	}
	if s.IsFinished() {
		ui.PrintResults(os.Stdout, s.Results())
	}
}

func runAuto(cfg sim.SimulationConfig) {
	r := experiment.NewExperimentRunner(cfg)
	need, hist := r.FindRequiredRequests()
	fmt.Println("\nПодбор числа заявок: N = tα²·(1 − p)/(p·δ²), tα = 1,643, δ = 0,1")
	for _, h := range hist {
		fmt.Printf("  N = %-7d → p = %.4f\n", h.N, h.P)
	}
	if need > cfg.RequestsLimit {
		fmt.Printf("Заданного N = %d мало, используется N = %d\n", cfg.RequestsLimit, need)
		cfg.RequestsLimit = need
	}
	s, _ := sim.New(cfg)
	ui.PrintResults(os.Stdout, s.RunAuto())
}

// runExperiment перебирает число приборов и размер буфера и сравнивает конфигурации
// по требованиям задания: Pотк ≤ 10 %, загрузка приборов около 90 %, минимальное Тпреб.
func runExperiment(base sim.SimulationConfig, deviceCost, rejectCost float64) {
	type row struct {
		m, n              int
		p, k, tpreb, cost float64
		ok                bool
	}
	var rows []row
	for m := 1; m <= base.DeviceCount+3; m++ {
		for _, buf := range []int{2, 3, 5, 8, 12} {
			c := base
			c.DeviceCount, c.BufferCapacity = m, buf
			need, _ := experiment.NewExperimentRunner(c).FindRequiredRequests()
			// При перегрузке отказы идут пачками и формула для N их недооценивает,
			// поэтому для сравнения конфигураций берём не меньше ExperimentMinN заявок.
			need = max(need, ExperimentMinN)
			if need > c.RequestsLimit {
				c.RequestsLimit = need
			}
			s, _ := sim.New(c)
			t := s.RunAuto()
			// Стоимость за единицу времени: содержание приборов + потерянные тревоги.
			cost := float64(m)*deviceCost + rejectCost*float64(t.Total.Rejected)/t.TotalTime
			rows = append(rows, row{m, buf, t.Total.RejectProbability, t.AvgUtilization(), t.Total.AvgTimeInSystem, cost,
				t.Total.RejectProbability <= 0.10})
		}
	}
	fmt.Printf("\nСравнение конфигураций (стоимость прибора %.1f/ед. времени, потерянной тревоги %.1f)\n\n", deviceCost, rejectCost)
	tw := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', tabwriter.AlignRight)
	fmt.Fprintln(tw, "Приборов\tБуфер\tPотк\tКисп\tТпреб\tЗатраты/ед.t\tPотк≤10%\t")
	best := -1
	for i, r := range rows {
		mark := "нет"
		if r.ok {
			mark = "да"
			if best < 0 || r.cost < rows[best].cost {
				best = i
			}
		}
		fmt.Fprintf(tw, "%d\t%d\t%.4f\t%.3f\t%.3f\t%.2f\t%s\t\n", r.m, r.n, r.p, r.k, r.tpreb, r.cost, mark)
	}
	tw.Flush()
	if best >= 0 {
		r := rows[best]
		fmt.Printf("\nСамая дешёвая конфигурация с Pотк ≤ 10%%: приборов — %d, мест в буфере — %d — Pотк = %.2f%%, Кисп = %.1f%%, Тпреб = %.3f\n",
			r.m, r.n, 100*r.p, 100*r.k, r.tpreb)
	}
}

func parseLambdas(s string) []float64 {
	var out []float64
	for _, p := range strings.Split(s, ",") {
		if v, err := strconv.ParseFloat(strings.TrimSpace(p), 64); err == nil {
			out = append(out, v)
		}
	}
	return out
}

func printConfig(c sim.SimulationConfig) {
	fmt.Println("СМО «Обработка промышленных тревог», вариант 3: ИБ ИЗ1 ПЗ2 Д1ОЗ1 Д1ОО3 Д2П1 Д2Б3 ОР1 ОД1")
	fmt.Printf("Источников: %d, λ = %v · приборов: %d, ПЗ2 на [%.2f; %.2f] · буфер: %d · заявок: %d · seed: %d\n",
		len(c.SourceLambdas), c.SourceLambdas, c.DeviceCount, c.ServiceMin, c.ServiceMax, c.BufferCapacity, c.RequestsLimit, c.Seed)
}

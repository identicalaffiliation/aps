// Package ui — текстовое отображение: ОД1 (пошаговый режим) и ОР1 (сводная таблица).
package ui

import (
	"fmt"
	"io"
	"strings"
	"text/tabwriter"

	"smo-alarms/internal/sim"
	"smo-alarms/internal/stats"
)

// PrintStep — ОД1: календарь событий, буфер с указателями, приборы, % отказа.
func PrintStep(w io.Writer, s *sim.StepSnapshot) {
	fmt.Fprintf(w, "\n══════ Шаг %d · t = %.3f · событие: %s ══════\n", s.Step, s.Time, s.Event)
	for _, a := range s.Actions {
		fmt.Fprintf(w, "  • %s\n", a)
	}

	fmt.Fprintln(w, "\n  Календарь событий:")
	tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
	fmt.Fprintln(tw, "    №\tВремя\tСобытие\tОбъект")
	for i, e := range s.Calendar {
		obj := fmt.Sprintf("И%d", e.SourceID)
		if e.SourceID == 0 {
			obj = fmt.Sprintf("П%d", e.DeviceID)
		}
		fmt.Fprintf(tw, "    %d\t%.3f\t%s\t%s\n", i+1, e.Time, e.Type, obj)
	}
	if len(s.Calendar) == 0 {
		fmt.Fprintln(tw, "    —\tпуст\t\t")
	}
	tw.Flush()

	fmt.Fprintln(w, "\n  Буфер:")
	tw = tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
	fmt.Fprintln(tw, "    Место\tЗаявка\tУказатели")
	for i, r := range s.BufferSlots {
		if r == "" {
			r = "—"
		}
		var p []string
		if i == s.PutPointer {
			p = append(p, "← УП")
		}
		if i == s.TakePointer {
			p = append(p, "← УВ")
		}
		fmt.Fprintf(tw, "    %d\t%s\t%s\n", i+1, r, strings.Join(p, " "))
	}
	tw.Flush()

	fmt.Fprintln(w, "\n  Приборы:")
	tw = tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
	fmt.Fprintln(tw, "    Прибор\tСостояние\tЗаявка\tОкончание")
	for _, d := range s.Devices {
		if d.Request == "" {
			fmt.Fprintf(tw, "    П%d\tсвободен\t—\t—\n", d.ID)
		} else {
			fmt.Fprintf(tw, "    П%d\tзанят\t%s\t%.3f\n", d.ID, d.Request, d.Until)
		}
	}
	tw.Flush()
	fmt.Fprintf(w, "\n  Сгенерировано: %d · отказов: %d · %% отказа: %.2f%%\n", s.Generated, s.Rejected, s.RejectPercent)
}

// PrintResults — ОР1: сводная таблица по источникам и приборам.
func PrintResults(w io.Writer, t *stats.ResultsTable) {
	fmt.Fprintf(w, "\nСводная таблица (ОР1). Время реализации T = %.3f\n\n", t.TotalTime)
	tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', tabwriter.AlignRight)
	fmt.Fprintln(tw, "Источник\tЗаявок\tОбслуж.\tОтказов\tPотк\tТпреб\tТбп\tТобсл\tДбп\tДобсл\t")
	for _, r := range t.SourceRows {
		writeRow(tw, fmt.Sprintf("И%d", r.SourceID), r)
	}
	writeRow(tw, "Всего", t.Total)
	tw.Flush()

	fmt.Fprintln(w)
	tw = tabwriter.NewWriter(w, 0, 0, 2, ' ', tabwriter.AlignRight)
	fmt.Fprintln(tw, "Прибор\tКисп\t")
	for _, d := range t.DeviceRows {
		fmt.Fprintf(tw, "П%d\t%.3f\t\n", d.DeviceID, d.Utilization)
	}
	fmt.Fprintf(tw, "Среднее\t%.3f\t\n", t.AvgUtilization())
	tw.Flush()
}

func writeRow(tw io.Writer, name string, r stats.SourceStats) {
	fmt.Fprintf(tw, "%s\t%d\t%d\t%d\t%.4f\t%.3f\t%.3f\t%.3f\t%.3f\t%.3f\t\n",
		name, r.Generated, r.Served, r.Rejected, r.RejectProbability,
		r.AvgTimeInSystem, r.AvgTimeInBuffer, r.AvgServiceTime, r.VarTimeInBuffer, r.VarServiceTime)
}

// Package smo — элементы СМО: источник, буфер, прибор.
package smo

import (
	"fmt"

	"smo-alarms/internal/dist"
	"smo-alarms/internal/model"
)

// IDGenerator выдаёт сквозные идентификаторы заявок для всех источников.
type IDGenerator struct{ next int64 }

func (g *IDGenerator) Next() int64 { g.next++; return g.next }

// Source — бесконечный источник (ИБ): датчик / контроллер линии / СКУД.
type Source struct {
	id         int
	interval   dist.Distribution // ИЗ1
	nextNumber int
	ids        *IDGenerator
}

func NewSource(id int, interval dist.Distribution, ids *IDGenerator) *Source {
	return &Source{id: id, interval: interval, ids: ids}
}

// Generate формирует новую тревогу в момент currentTime.
func (s *Source) Generate(currentTime float64) *model.Request {
	s.nextNumber++
	alarm := model.AlarmType((s.id - 1) % 3) // 1 — датчики, 2 — контроллеры, 3 — СКУД
	object := fmt.Sprintf("объект-%d.%d", s.id, s.nextNumber)
	return model.NewRequest(s.ids.Next(), s.id, s.nextNumber, alarm, object, currentTime)
}

// NextArrivalInterval — интервал до следующей тревоги по ИЗ1.
func (s *Source) NextArrivalInterval() float64 { return s.interval.Next() }

func (s *Source) ID() int { return s.id }

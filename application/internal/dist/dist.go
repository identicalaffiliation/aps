// Package dist — законы распределения варианта: ИЗ1 (экспоненциальный) и ПЗ2 (равномерный).
package dist

import (
	"math"
	"math/rand"
)

// Distribution — источник случайных интервалов.
type Distribution interface {
	Next() float64
}

// Exponential — ИЗ1, пуассоновский поток: τ = −ln(r)/λ.
type Exponential struct {
	lambda float64
	random *rand.Rand
}

func NewExponential(lambda float64, random *rand.Rand) *Exponential {
	return &Exponential{lambda: lambda, random: random}
}

func (e *Exponential) Next() float64 {
	// rand.Float64() ∈ [0;1), поэтому 1 − r ∈ (0;1] и логарифм всегда определён.
	r := 1 - e.random.Float64()
	return -math.Log(r) / e.lambda
}

// Uniform — ПЗ2: Тобсл = a + (b − a)·r.
type Uniform struct {
	a, b   float64
	random *rand.Rand
}

func NewUniform(a, b float64, random *rand.Rand) *Uniform {
	return &Uniform{a: a, b: b, random: random}
}

func (u *Uniform) Next() float64 { return u.a + (u.b-u.a)*u.random.Float64() }

// Fixed — заранее заданная последовательность значений. Нужна для тестов,
// чтобы воспроизвести сценарии с сиквенс-диаграмм на конкретных числах.
type Fixed struct {
	values []float64
	pos    int
}

func NewFixed(values ...float64) *Fixed { return &Fixed{values: values} }

func (f *Fixed) Next() float64 {
	v := f.values[f.pos%len(f.values)]
	f.pos++
	return v
}

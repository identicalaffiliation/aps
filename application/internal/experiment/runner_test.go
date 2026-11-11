package experiment

import "testing"

func TestRequiredN(t *testing.T) {
	// p = 0,1: 1,643²·0,9 / (0,1·0,01) = 2429,5 → 2430
	if n := RequiredN(0.1); n != 2430 {
		t.Fatalf("N(0,1) = %d, ожидали 2430", n)
	}
	if n := RequiredN(0); n != 0 {
		t.Fatalf("при p = 0 формула неприменима")
	}
}

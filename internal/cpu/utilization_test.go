package cpu

import (
	"math"
	"testing"
)

func TestUtilization(t *testing.T) {
	delta := Delta{
		User:    20,
		Nice:    5,
		System:  10,
		Idle:    50,
		IOWait:  5,
		IRQ:     2,
		SoftIRQ: 3,
		Steal:   5,
	}

	got, err := Utilization(delta)
	if err != nil {
		t.Fatalf("Utilization() returned unexpected error: %v", err)
	}

	want := 45.0

	if math.Abs(got-want) > 1e-9 {
		t.Fatalf("Utilization() = %v, want %v", got, want)
	}
}

func TestUtilizationRejectsZeroTotal(t *testing.T) {
	_, err := Utilization(Delta{})
	if err == nil {
		t.Fatal("Utilization() expected error for zero total delta")
	}
}

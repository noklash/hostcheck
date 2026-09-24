package process

import (
	"testing"
	"time"
)

func TestUtilization(t *testing.T) {
	delta := Delta{
		PID:            1234,
		CPUTimeTicks:   160,
		CPUTimeSeconds: 1.6,
		Elapsed:        2 * time.Second,
	}

	utilization, err := Utilization(delta)
	if err != nil {
		t.Fatalf("Utilization() error = %v", err)
	}

	want := 0.8

	if utilization != want {
		t.Errorf("Utilization() = %f, want %f", utilization, want)
	}
}

func TestUtilizationFullCPU(t *testing.T) {
	delta := Delta{
		PID:            1234,
		CPUTimeSeconds: 2,
		Elapsed:        2 * time.Second,
	}

	utilization, err := Utilization(delta)
	if err != nil {
		t.Fatalf("Utilization() error = %v", err)
	}

	if utilization != 1.0 {
		t.Errorf("Utilization() = %f, want 1.0", utilization)
	}
}

func TestUtilizationHalfCPU(t *testing.T) {
	delta := Delta{
		PID:            1234,
		CPUTimeSeconds: 1,
		Elapsed:        2 * time.Second,
	}

	utilization, err := Utilization(delta)
	if err != nil {
		t.Fatalf("Utilization() error = %v", err)
	}

	if utilization != 0.5 {
		t.Errorf("Utilization() = %f, want 0.5", utilization)
	}
}

func TestUtilizationRejectsZeroElapsed(t *testing.T) {
	delta := Delta{
		PID:            1234,
		CPUTimeSeconds: 1,
	}

	_, err := Utilization(delta)
	if err == nil {
		t.Fatal("Utilization() error = nil, want error")
	}
}

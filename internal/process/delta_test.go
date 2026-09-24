package process

import (
	"strings"
	"testing"
	"time"
)

func TestDeltaSamples(t *testing.T) {
	start := time.Unix(100, 0)

	previous := Sample{
		PID:        1234,
		StartTime:  5000,
		UTime:      100,
		STime:      20,
		ObservedAt: start,
	}

	current := Sample{
		PID:        1234,
		StartTime:  5000,
		UTime:      250,
		STime:      30,
		ObservedAt: start.Add(2 * time.Second),
	}

	delta, err := DeltaSamples(previous, current)
	if err != nil {
		t.Fatalf("DeltaSamples() error = %v", err)
	}

	if delta.PID != 1234 {
		t.Errorf("PID = %d, want 1234", delta.PID)
	}

	if delta.CPUTimeTicks != 160 {
		t.Errorf("CPUTimeTicks = %d, want 160", delta.CPUTimeTicks)
	}

	if delta.CPUTimeSeconds != 1.6 {
		t.Errorf("CPUTimeSeconds = %f, want 1.6", delta.CPUTimeSeconds)
	}

	if delta.Elapsed != 2*time.Second {
		t.Errorf("Elapsed = %v, want 2s", delta.Elapsed)
	}
}

func TestDeltaSamplesRejectsPIDChange(t *testing.T) {
	previous := Sample{
		PID:        1234,
		StartTime:  5000,
		UTime:      100,
		ObservedAt: time.Unix(100, 0),
	}

	current := previous
	current.PID = 5678
	current.ObservedAt = previous.ObservedAt.Add(time.Second)

	_, err := DeltaSamples(previous, current)
	if err == nil {
		t.Fatal("DeltaSamples() error = nil, want error")
	}

	if !strings.Contains(err.Error(), "pid changed") {
		t.Fatalf("error = %q, want PID change error", err)
	}
}

func TestDeltaSamplesRejectsPIDReuse(t *testing.T) {
	previous := Sample{
		PID:        1234,
		StartTime:  5000,
		UTime:      100,
		STime:      20,
		ObservedAt: time.Unix(100, 0),
	}

	current := previous
	current.StartTime = 9000
	current.UTime = 10
	current.STime = 2
	current.ObservedAt = previous.ObservedAt.Add(time.Second)

	_, err := DeltaSamples(previous, current)
	if err == nil {
		t.Fatal("DeltaSamples() error = nil, want PID reuse error")
	}

	if !strings.Contains(err.Error(), "process identity changed") {
		t.Fatalf("error = %q, want identity change error", err)
	}
}

func TestDeltaSamplesRejectsCounterRollback(t *testing.T) {
	previous := Sample{
		PID:        1234,
		StartTime:  5000,
		UTime:      100,
		STime:      20,
		ObservedAt: time.Unix(100, 0),
	}

	current := previous
	current.UTime = 90
	current.ObservedAt = previous.ObservedAt.Add(time.Second)

	_, err := DeltaSamples(previous, current)
	if err == nil {
		t.Fatal("DeltaSamples() error = nil, want error")
	}

	if !strings.Contains(err.Error(), "utime counter moved backwards") {
		t.Fatalf("error = %q, want utime rollback error", err)
	}
}

func TestDeltaSamplesRejectsNonPositiveElapsed(t *testing.T) {
	previous := Sample{
		PID:        1234,
		StartTime:  5000,
		UTime:      100,
		ObservedAt: time.Unix(100, 0),
	}

	current := previous
	current.UTime = 110

	_, err := DeltaSamples(previous, current)
	if err == nil {
		t.Fatal("DeltaSamples() error = nil, want error")
	}
}

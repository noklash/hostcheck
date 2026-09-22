package cpu

import "testing"

func TestStatDelta(t *testing.T) {
	previous := Stat{
		User:      100,
		Nice:      20,
		System:    30,
		Idle:      400,
		IOWait:    10,
		IRQ:       5,
		SoftIRQ:   8,
		Steal:     2,
		Guest:     4,
		GuestNice: 1,
	}

	current := Stat{
		User:      150,
		Nice:      25,
		System:    45,
		Idle:      500,
		IOWait:    13,
		IRQ:       7,
		SoftIRQ:   11,
		Steal:     3,
		Guest:     6,
		GuestNice: 2,
	}

	got, err := current.Delta(previous)
	if err != nil {
		t.Fatalf("Delta() error = %v", err)
	}

	want := Delta{
		User:      50,
		Nice:      5,
		System:    15,
		Idle:      100,
		IOWait:    3,
		IRQ:       2,
		SoftIRQ:   3,
		Steal:     1,
		Guest:     2,
		GuestNice: 1,
	}

	if got != want {
		t.Fatalf("Delta() = %+v, want %+v", got, want)
	}
}

func TestStatDeltaRejectsCounterRegression(t *testing.T) {
	previous := Stat{
		User:      100,
		Nice:      20,
		System:    30,
		Idle:      400,
		IOWait:    10,
		IRQ:       5,
		SoftIRQ:   8,
		Steal:     2,
		Guest:     4,
		GuestNice: 1,
	}

	current := previous
	current.User = previous.User - 1

	got, err := current.Delta(previous)
	if err == nil {
		t.Fatal("Delta() error = nil, want counter regression error")
	}

	if got != (Delta{}) {
		t.Fatalf("Delta() = %+v, want zero delta", got)
	}
}

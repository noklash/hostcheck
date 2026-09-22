package cpu

import (
	"strings"
	"testing"
)

func TestReadStatFindsAggregateCPURecord(t *testing.T) {
	input := `cpu0  100 20 30 400 10 5 8 2 4 1
cpu  109 50 37 1 3 2 4 5 6 7
`

	got, err := readStat(strings.NewReader(input))
	if err != nil {
		t.Fatalf("readStat() error = %v", err)
	}

	want := Stat{
		User:      109,
		Nice:      50,
		System:    37,
		Idle:      1,
		IOWait:    3,
		IRQ:       2,
		SoftIRQ:   4,
		Steal:     5,
		Guest:     6,
		GuestNice: 7,
	}

	if got != want {
		t.Fatalf("readStat() = %+v, want %+v", got, want)
	}
}

func TestReadStatRejectsMissingAggregateCPURecord(t *testing.T) {
	input := `cpu0 100 20 30 400 10 5 8 2 4 1
cpu1 100 20 30 400 10 5 8 2 4 1
`

	_, err := readStat(strings.NewReader(input))
	if err == nil {
		t.Fatal("readStat() error = nil, want error")
	}
}

func TestReadStatRejectsMalformedAggregateCPURecord(t *testing.T) {
	input := `cpu0 100 20 30 400 10 5 8 2 4 1
cpu  109 50 invalid 1 3 2 4 5 6 7
`

	_, err := readStat(strings.NewReader(input))
	if err == nil {
		t.Fatal("readStat() error = nil, want error")
	}
}

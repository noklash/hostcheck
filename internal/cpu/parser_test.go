package cpu

import "testing"

func TestParseStat(t *testing.T) {
	line := "cpu  109 50 37 1 3 2 4 5 6 7"

	got, err := ParseStat(line)
	if err != nil {
		t.Fatalf("ParseStat() returned unexpected error: %v", err)
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
		t.Fatalf("ParseStat() = %+v, want %+v", got, want)
	}
}

func TestParseStatRejectsWrongFieldCount(t *testing.T) {
	line := "cpu  109 50 37"

	_, err := ParseStat(line)
	if err == nil {
		t.Fatal("ParseStat() expected error for invalid field count")
	}
}

func TestParseStatRejectsWrongRecordType(t *testing.T) {
	line := "cpu0  109 50 37 1 3 2 4 5 6 7"

	_, err := ParseStat(line)
	if err == nil {
		t.Fatal("ParseStat() expected error for non-aggregate CPU record")
	}
}

func TestParseStatRejectsInvalidCounter(t *testing.T) {
	line := "cpu  109 invalid 37 1 3 2 4 5 6 7"

	_, err := ParseStat(line)
	if err == nil {
		t.Fatal("ParseStat() expected error for invalid counter")
	}
}

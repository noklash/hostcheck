package memory

import (
	"testing"
)

func TestParseMemInfo(t *testing.T) {
	input := `MemTotal:        3480368 kB
MemFree:          260380 kB
MemAvailable:    1324752 kB
Buffers:           22772 kB
Cached:          1084488 kB
SwapCached:       342608 kB
Active:          2089956 kB
Inactive:         942296 kB
SwapTotal:       4003836 kB
SwapFree:        2439972 kB
`

	got, err := ParseMemInfo(input)
	if err != nil {
		t.Fatalf("ParseMemInfo() error = %v", err)
	}

	want := MemInfo{
		Total:     3480368,
		Available: 1324752,
		Free:      260380,
		Buffers:   22772,
		Cached:    1084488,
		SwapTotal: 4003836,
		SwapFree:  2439972,
	}

	if got != want {
		t.Fatalf("ParseMemInfo() = %+v, want %+v", got, want)
	}
}

func TestParseMemInfoIgnoresUnknownFields(t *testing.T) {
	input := `MemTotal:       1000 kB
MemFree:          100 kB
MemAvailable:     500 kB
Buffers:           50 kB
Cached:           200 kB
SwapTotal:        800 kB
SwapFree:         600 kB
SomeNewField:      123 kB
`

	_, err := ParseMemInfo(input)
	if err != nil {
		t.Fatalf("ParseMemInfo() error = %v", err)
	}
}

func TestParseMemInfoRejectsInvalidValue(t *testing.T) {
	input := `MemTotal:       invalid kB
MemFree:          100 kB
MemAvailable:     500 kB
Buffers:           50 kB
Cached:           200 kB
SwapTotal:        800 kB
SwapFree:         600 kB
`

	_, err := ParseMemInfo(input)
	if err == nil {
		t.Fatal("ParseMemInfo() expected error for invalid value")
	}
}

func TestParseMemInfoRejectsInvalidUnit(t *testing.T) {
	input := `MemTotal:       1000 MB
MemFree:          100 kB
MemAvailable:     500 kB
Buffers:           50 kB
Cached:           200 kB
SwapTotal:        800 kB
SwapFree:         600 kB
`

	_, err := ParseMemInfo(input)
	if err == nil {
		t.Fatal("ParseMemInfo() expected error for invalid unit")
	}
}

func TestParseMemInfoRejectsMalformedRecognizedLine(t *testing.T) {
	input := `MemTotal:       1000 kB extra
MemFree:          100 kB
MemAvailable:     500 kB
Buffers:           50 kB
Cached:           200 kB
SwapTotal:        800 kB
SwapFree:         600 kB
`

	_, err := ParseMemInfo(input)
	if err == nil {
		t.Fatal("ParseMemInfo() expected error for malformed recognized line")
	}
}

func TestParseMemInfoRejectsMissingField(t *testing.T) {
	input := `MemTotal:       1000 kB
MemFree:          100 kB
MemAvailable:     500 kB
Buffers:           50 kB
Cached:           200 kB
SwapTotal:        800 kB
`

	_, err := ParseMemInfo(input)
	if err == nil {
		t.Fatal("ParseMemInfo() expected error for missing field")
	}
}

func TestParseMemInfoRejectsDuplicateField(t *testing.T) {
	input := `MemTotal:       1000 kB
MemTotal:       2000 kB
MemFree:          100 kB
MemAvailable:     500 kB
Buffers:           50 kB
Cached:           200 kB
SwapTotal:        800 kB
SwapFree:         600 kB
`

	_, err := ParseMemInfo(input)
	if err == nil {
		t.Fatal("ParseMemInfo() expected error for duplicate field")
	}
}

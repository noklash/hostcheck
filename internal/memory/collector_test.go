package memory

import (
	"errors"
	"strings"
	"testing"
)

func TestReadMemInfoReader(t *testing.T) {
	input := `MemTotal:       1000 kB
MemFree:          100 kB
MemAvailable:     500 kB
Buffers:           50 kB
Cached:           200 kB
SwapTotal:        800 kB
SwapFree:         600 kB
`

	got, err := readMemInfo(strings.NewReader(input))
	if err != nil {
		t.Fatalf("readMemInfo() error = %v", err)
	}

	want := MemInfo{
		Total:     1000,
		Available: 500,
		Free:      100,
		Buffers:   50,
		Cached:    200,
		SwapTotal: 800,
		SwapFree:  600,
	}

	if got != want {
		t.Fatalf("readMemInfo() = %+v, want %+v", got, want)
	}
}

func TestReadMemInfoRejectsMalformedData(t *testing.T) {
	input := `MemTotal:       invalid kB
MemFree:          100 kB
MemAvailable:     500 kB
Buffers:           50 kB
Cached:           200 kB
SwapTotal:        800 kB
SwapFree:         600 kB
`

	_, err := readMemInfo(strings.NewReader(input))
	if err == nil {
		t.Fatal("readMemInfo() expected error for malformed data")
	}
}

func TestReadMemInfoPropagatesReaderError(t *testing.T) {
	wantErr := errors.New("reader failure")

	reader := &failingReader{err: wantErr}

	_, err := readMemInfo(reader)
	if err == nil {
		t.Fatal("readMemInfo() expected error")
	}

	if !errors.Is(err, wantErr) {
		t.Fatalf("readMemInfo() error = %v, want wrapped error %v", err, wantErr)
	}
}

type failingReader struct {
	err error
}

func (r *failingReader) Read([]byte) (int, error) {
	return 0, r.err
}

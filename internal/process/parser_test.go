package process

import (
	"strings"
	"testing"
)

func TestParseStat(t *testing.T) {
	line := "46643 (bash) S 11653 46643 46643 34820 46854 4194304 1380 2173 4 0 16 3 2 5 20 0 1 0 1992836 20590592 1437 18446744073709551615 106247936307200 106247937366433 140724419267440 0 0 0 65536 3686404 1266761467 1 0 0 17 1 0 0 0 0 0 106247937595632 106247937644680 106248510603264 140724419270961 140724419270975 140724419270975 140724419272682 0"

	got, err := ParseStat(line)
	if err != nil {
		t.Fatalf("ParseStat() error = %v", err)
	}

	if got.PID != 46643 {
		t.Errorf("PID = %d, want 46643", got.PID)
	}
	if got.Comm != "bash" {
		t.Errorf("Comm = %q, want %q", got.Comm, "bash")
	}
	if got.State != 'S' {
		t.Errorf("State = %q, want %q", got.State, 'S')
	}
	if got.PPID != 11653 {
		t.Errorf("PPID = %d, want 11653", got.PPID)
	}
	if got.UTime != 16 {
		t.Errorf("UTime = %d, want 16", got.UTime)
	}
	if got.STime != 3 {
		t.Errorf("STime = %d, want 3", got.STime)
	}
	if got.Priority != 20 {
		t.Errorf("Priority = %d, want 20", got.Priority)
	}
	if got.Nice != 0 {
		t.Errorf("Nice = %d, want 0", got.Nice)
	}
	if got.Threads != 1 {
		t.Errorf("Threads = %d, want 1", got.Threads)
	}
	if got.StartTime != 1992836 {
		t.Errorf("StartTime = %d, want 1992836", got.StartTime)
	}
	if got.VSize != 20590592 {
		t.Errorf("VSize = %d, want 20590592", got.VSize)
	}
	if got.RSSPages != 1437 {
		t.Errorf("RSSPages = %d, want 1437", got.RSSPages)
	}
}

func TestParseStatCommWithSpaces(t *testing.T) {
	line := "48783 (name with space) S 48748 48783 48748 34820 48784 4194304 1199 0 0 0 4 2 0 0 20 0 1 0 2267789 32198656 2777 18446744073709551615 4333568 8027233 140737217070528 0 0 0 0 16781312 2 1 0 0 17 1 0 0 0 0 0 11038136 11672392 98349056 140737217073657 140737217073667 140737217073667 140737217077223 0"

	got, err := ParseStat(line)
	if err != nil {
		t.Fatalf("ParseStat() error = %v", err)
	}

	if got.Comm != "name with space" {
		t.Errorf("Comm = %q, want %q", got.Comm, "name with space")
	}
}

func TestParseStatCommWithClosingParentheses(t *testing.T) {
	line := "48815 (name)with)paren) S 48748 48815 48748 34819 48822 4194304 1199 0 0 0 4 2 0 0 20 0 1 0 2267789 32198656 2777 18446744073709551615 4333568 8027233 140737217070528 0 0 0 0 16781312 2 1 0 0 17 1 0 0 0 0 0 11038136 11672392 98349056 140737217073657 140737217073667 140737217073667 140737217077223 0"

	got, err := ParseStat(line)
	if err != nil {
		t.Fatalf("ParseStat() error = %v", err)
	}

	if got.Comm != "name)with)paren" {
		t.Errorf("Comm = %q, want %q", got.Comm, "name)with)paren")
	}
	if got.State != 'S' {
		t.Errorf("State = %q, want %q", got.State, 'S')
	}
}

func TestParseStatNegativePriorityAndNice(t *testing.T) {
	line := "1 (test) S 0 1 1 0 0 0 0 0 0 0 10 2 0 0 -20 -5 1 0 100 4096 2 0"

	got, err := ParseStat(line)
	if err != nil {
		t.Fatalf("ParseStat() error = %v", err)
	}

	if got.Priority != -20 {
		t.Errorf("Priority = %d, want -20", got.Priority)
	}
	if got.Nice != -5 {
		t.Errorf("Nice = %d, want -5", got.Nice)
	}
}

func TestParseStatRejectsEmpty(t *testing.T) {
	_, err := ParseStat("")
	if err == nil {
		t.Fatal("ParseStat() error = nil, want error")
	}
}

func TestParseStatRejectsMissingOpeningParenthesis(t *testing.T) {
	_, err := ParseStat("123 bash) S 1 2 3")
	if err == nil {
		t.Fatal("ParseStat() error = nil, want error")
	}
}

func TestParseStatRejectsMissingClosingParenthesis(t *testing.T) {
	_, err := ParseStat("123 (bash S 1 2 3")
	if err == nil {
		t.Fatal("ParseStat() error = nil, want error")
	}
}

func TestParseStatRejectsTooFewFields(t *testing.T) {
	_, err := ParseStat("123 (bash) S 1 2 3")
	if err == nil {
		t.Fatal("ParseStat() error = nil, want error")
	}
}

func TestParseStatRejectsInvalidPID(t *testing.T) {
	line := "abc (bash) S 1 2 3"
	_, err := ParseStat(line)
	if err == nil {
		t.Fatal("ParseStat() error = nil, want error")
	}
}

func TestParseStatRejectsInvalidNumericField(t *testing.T) {
	fields := strings.Fields("123 (bash) S 1 2 3 4 5 6 7 8 9 10 11 12 13 14 15 16 17 18 19 20 21 22")
	fields[3] = "not-a-number"

	line := strings.Join(fields, " ")

	_, err := ParseStat(line)
	if err == nil {
		t.Fatal("ParseStat() error = nil, want error")
	}
}

func TestParseStatRejectsNegativeUnsignedField(t *testing.T) {
	line := "123 (bash) S 1 2 3 4 5 6 7 8 9 10 -11 12 13 14 15 16 17 18 19 20 21 22"

	_, err := ParseStat(line)
	if err == nil {
		t.Fatal("ParseStat() error = nil, want error")
	}
}

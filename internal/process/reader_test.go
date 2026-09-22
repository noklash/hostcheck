package process

import (
	"os"
	"testing"
)

func TestReadStatsSelf(t *testing.T) {
	pid := int64(os.Getpid())

	stats, err := ReadStats(pid)
	if err != nil {
		t.Fatalf("ReadStats() error = %v", err)
	}

	if stats.PID != pid {
		t.Errorf("PID = %d, want %d", stats.PID, pid)
	}

	if stats.Comm == "" {
		t.Error("Comm is empty")
	}

	if stats.Threads == 0 {
		t.Error("Threads = 0, want non-zero")
	}
}

func TestReadStatsMissingProcess(t *testing.T) {
	_, err := ReadStats(999999999)
	if err == nil {
		t.Fatal("ReadStats() error = nil, want error")
	}
}

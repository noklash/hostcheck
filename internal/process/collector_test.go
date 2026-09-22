package process

import (
	"os"
	"testing"
)

func TestCollect(t *testing.T) {
	stats, err := Collect()
	if err != nil {
		t.Fatalf("Collect() error = %v", err)
	}

	if len(stats) == 0 {
		t.Fatal("Collect() returned no processes")
	}

	foundSelf := false

	for _, stat := range stats {
		if stat.PID <= 0 {
			t.Errorf("PID = %d, want positive PID", stat.PID)
		}

		if stat.Comm == "" {
			t.Errorf("PID %d has empty Comm", stat.PID)
		}

		if stat.Threads == 0 {
			t.Errorf("PID %d has zero Threads", stat.PID)
		}

		if stat.PID == int64(os.Getpid()) {
			foundSelf = true
		}
	}

	if !foundSelf {
		t.Errorf("current test process PID %d not found", os.Getpid())
	}
}

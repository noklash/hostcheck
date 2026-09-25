package process

import (
	"os"
	"testing"
)

func TestCollect(t *testing.T) {
	processes, err := Collect()
	if err != nil {
		t.Fatalf("Collect() error = %v", err)
	}

	if len(processes) == 0 {
		t.Fatal("Collect() returned no processes")
	}

	foundSelf := false

	for _, process := range processes {
		stat := process.Stats

		if stat.PID <= 0 {
			t.Errorf("PID = %d, want positive PID", stat.PID)
		}

		if stat.Comm == "" {
			t.Errorf("PID %d has empty Comm", stat.PID)
		}

		if stat.Threads == 0 {
			t.Errorf("PID %d has zero Threads", stat.PID)
		}

		if process.Kthread {
			if process.Memory != nil {
				t.Errorf("PID %d is a kernel thread but has memory data", stat.PID)
			}
		} else {
			if process.Memory == nil {
				t.Errorf("PID %d is a userspace process but has no memory data", stat.PID)
			}

			if process.Memory != nil &&
				process.Memory.ResidentBytes > process.Memory.VirtualBytes {
				t.Errorf(
					"PID %d has resident memory %d greater than virtual memory %d",
					stat.PID,
					process.Memory.ResidentBytes,
					process.Memory.VirtualBytes,
				)
			}
		}

		if stat.PID == int64(os.Getpid()) {
			foundSelf = true

			if process.Kthread {
				t.Error("current test process classified as kernel thread")
			}

			if process.Memory == nil {
				t.Error("current test process has no memory data")
			}
		}
	}

	if !foundSelf {
		t.Errorf("current test process PID %d not found", os.Getpid())
	}
}

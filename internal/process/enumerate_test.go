package process

import (
	"testing"
)

func TestListPIDs(t *testing.T) {
	pids, err := ListPIDs()
	if err != nil {
		t.Fatalf("ListPIDs() error = %v", err)
	}

	if len(pids) == 0 {
		t.Fatal("ListPIDs() returned no processes")
	}

	for _, pid := range pids {
		if pid <= 0 {
			t.Fatalf("ListPIDs() returned invalid PID %d", pid)
		}
	}
}

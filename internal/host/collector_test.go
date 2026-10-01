package host

import (
	"testing"
	"time"
)

func TestCollect(t *testing.T) {
	before := time.Now()

	snapshot := Collect(Config{})

	after := time.Now()

	if snapshot.ObservedAt.Before(before) || snapshot.ObservedAt.After(after) {
		t.Fatalf("ObservedAt = %v, want timestamp between %v and %v",
			snapshot.ObservedAt, before, after)
	}

	if snapshot.CPU == nil {
		t.Fatal("CPU snapshot is nil")
	}

	if snapshot.Memory == nil {
		t.Fatal("memory snapshot is nil")
	}

	if snapshot.Filesystem == nil {
		t.Fatal("filesystem snapshot is nil")
	}

	if snapshot.Processes == nil {
		t.Fatal("process snapshot is nil")
	}

	if snapshot.Network == nil {
		t.Fatal("network snapshot is nil")
	}

	if len(snapshot.Errors) != 0 {
		t.Fatalf("unexpected collection errors: %v", snapshot.Errors)
	}
}

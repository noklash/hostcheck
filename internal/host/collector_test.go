package host

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestCollect(t *testing.T) {
	before := time.Now()

	snapshot := Collect(Config{})

	after := time.Now()

	if snapshot.ObservedAt.Before(before) || snapshot.ObservedAt.After(after) {
		t.Fatalf(
			"ObservedAt = %v, want timestamp between %v and %v",
			snapshot.ObservedAt,
			before,
			after,
		)
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

func TestCollectPreservesSuccessfulObservationsWhenFilesystemCollectionFails(
	t *testing.T,
) {
	missingPath := filepath.Join(t.TempDir(), "missing-filesystem")

	snapshot := Collect(Config{
		FilesystemPath: missingPath,
	})

	if snapshot.CPU == nil {
		t.Error("CPU snapshot is nil after filesystem collection failure")
	}

	if snapshot.Memory == nil {
		t.Error("memory snapshot is nil after filesystem collection failure")
	}

	if snapshot.Processes == nil {
		t.Error("process snapshot is nil after filesystem collection failure")
	}

	if snapshot.Network == nil {
		t.Error("network snapshot is nil after filesystem collection failure")
	}

	if snapshot.Filesystem != nil {
		t.Errorf(
			"filesystem snapshot = %v, want nil after collection failure",
			snapshot.Filesystem,
		)
	}

	var filesystemErrorFound bool

	for _, collectionError := range snapshot.Errors {
		if collectionError.Subsystem != "filesystem" {
			continue
		}

		filesystemErrorFound = true

		if !errors.Is(collectionError.Err, os.ErrNotExist) {
			t.Errorf(
				"filesystem error = %v, want an error wrapping %v",
				collectionError.Err,
				os.ErrNotExist,
			)
		}
	}

	if !filesystemErrorFound {
		t.Fatal("filesystem collection error was not recorded")
	}
}

package filesystem

import (
	"errors"
	"syscall"
	"testing"
)

func TestReadStats(t *testing.T) {
	stats, err := ReadStats("/")
	if err != nil {
		t.Fatalf("ReadStats() error = %v", err)
	}

	if stats.Path != "/" {
		t.Fatalf("Path = %q, want %q", stats.Path, "/")
	}

	if stats.BlockSize == 0 {
		t.Fatal("BlockSize = 0, want non-zero")
	}

	if stats.BlocksTotal == 0 {
		t.Fatal("BlocksTotal = 0, want non-zero")
	}
}

func TestReadStatsReader(t *testing.T) {
	var gotPath string

	stats, err := readStats("/test", func(path string, buf *syscall.Statfs_t) error {
		gotPath = path

		*buf = syscall.Statfs_t{
			Bsize:  4096,
			Blocks: 10000,
			Bfree:  4000,
			Bavail: 3500,
			Files:  5000,
			Ffree:  2000,
		}

		return nil
	})
	if err != nil {
		t.Fatalf("readStats() error = %v", err)
	}

	if gotPath != "/test" {
		t.Fatalf("statfs path = %q, want %q", gotPath, "/test")
	}

	want := Stats{
		Path:            "/test",
		BlockSize:       4096,
		BlocksTotal:     10000,
		BlocksFree:      4000,
		BlocksAvailable: 3500,
		InodesTotal:     5000,
		InodesFree:      2000,
	}

	if stats != want {
		t.Fatalf("readStats() = %+v, want %+v", stats, want)
	}
}

func TestReadStatsReaderError(t *testing.T) {
	wantErr := errors.New("statfs failed")

	_, err := readStats("/test", func(path string, buf *syscall.Statfs_t) error {
		return wantErr
	})

	if !errors.Is(err, wantErr) {
		t.Fatalf("readStats() error = %v, want wrapped %v", err, wantErr)
	}
}

func TestReadStatsRejectsZeroBlockSize(t *testing.T) {
	_, err := readStats("/test", func(path string, buf *syscall.Statfs_t) error {
		*buf = syscall.Statfs_t{
			Bsize:  0,
			Blocks: 100,
		}

		return nil
	})

	if err == nil {
		t.Fatal("readStats() error = nil, want error")
	}
}

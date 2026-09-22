package filesystem

import (
	"fmt"
	"syscall"
)

type statfsFunc func(path string, buf *syscall.Statfs_t) error

func systemStatfs(path string, buf *syscall.Statfs_t) error {
	return syscall.Statfs(path, buf)
}

// ReadStats reads filesystem capacity and inode statistics for path.
func ReadStats(path string) (Stats, error) {
	return readStats(path, systemStatfs)
}

func readStats(path string, statfs statfsFunc) (Stats, error) {
	var raw syscall.Statfs_t

	if err := statfs(path, &raw); err != nil {
		return Stats{}, fmt.Errorf("statfs %q: %w", path, err)
	}

	if raw.Bsize <= 0 {
		return Stats{}, fmt.Errorf("statfs %q returned invalid block size %d", path, raw.Bsize)
	}

	return Stats{
		Path:            path,
		BlockSize:       uint64(raw.Bsize),
		BlocksTotal:     raw.Blocks,
		BlocksFree:      raw.Bfree,
		BlocksAvailable: raw.Bavail,
		InodesTotal:     raw.Files,
		InodesFree:      raw.Ffree,
	}, nil
}

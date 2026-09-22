package process

import (
	"os"
	"strconv"
)

// ListPIDs returns the process IDs currently visible under /proc.
//
// The /proc filesystem is a live kernel interface, so the returned list
// represents what was observed during enumeration and may change immediately
// afterward.
func ListPIDs() ([]int64, error) {
	entries, err := os.ReadDir("/proc")
	if err != nil {
		return nil, err
	}

	pids := make([]int64, 0)

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		pid, err := strconv.ParseInt(entry.Name(), 10, 64)
		if err != nil || pid <= 0 {
			continue
		}

		pids = append(pids, pid)
	}

	return pids, nil
}

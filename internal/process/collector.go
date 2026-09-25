package process

import (
	"errors"
	"fmt"
	"os"
)

// Collect returns process observations for processes visible during
// collection.
//
// The /proc filesystem is live and processes may exit between enumeration
// and individual interface reads. A process that disappears during
// collection is skipped.
//
// Kernel threads are included in the collection. Their Kthread field is true
// and their Memory field is nil because they do not have userspace memory
// accounting.
func Collect() ([]Process, error) {
	pids, err := ListPIDs()
	if err != nil {
		return nil, fmt.Errorf("enumerate processes: %w", err)
	}

	processes := make([]Process, 0, len(pids))

	for _, pid := range pids {
		stats, err := ReadStats(pid)
		if err != nil {
			if errors.Is(err, os.ErrNotExist) {
				continue
			}

			return nil, fmt.Errorf("collect process %d stats: %w", pid, err)
		}

		status, err := ReadStatus(pid)
		if err != nil {
			if errors.Is(err, os.ErrNotExist) {
				continue
			}

			return nil, fmt.Errorf("collect process %d status: %w", pid, err)
		}

		processes = append(processes, Process{
			Stats:   stats,
			Kthread: status.Kthread,
			Memory:  status.Memory,
		})
	}

	return processes, nil
}

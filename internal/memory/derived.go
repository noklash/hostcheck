package memory

import "fmt"

// AvailablePercent derives the percentage of total memory currently available.
func AvailablePercent(mem MemInfo) (float64, error) {
	if mem.Total == 0 {
		return 0, fmt.Errorf("memory total must be greater than zero")
	}

	if mem.Available > mem.Total {
		return 0, fmt.Errorf("available memory cannot exceed total memory")
	}

	return float64(mem.Available) / float64(mem.Total) * 100, nil
}

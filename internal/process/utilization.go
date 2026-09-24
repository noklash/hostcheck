package process

import "fmt"

// Utilization returns process CPU utilization as a fraction of one CPU.
//
// A result of 1.0 means the process consumed approximately one full CPU
// during the observation interval. A result of 0.5 means approximately half
// of one CPU.
func Utilization(delta Delta) (float64, error) {
	if delta.Elapsed <= 0 {
		return 0, fmt.Errorf("elapsed time must be positive")
	}

	utilization := delta.CPUTimeSeconds / delta.Elapsed.Seconds()

	if utilization < 0 {
		return 0, fmt.Errorf("utilization cannot be negative")
	}

	return utilization, nil
}

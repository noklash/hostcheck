package cpu

import "fmt"

// Utilization calculates aggregate CPU utilization from a CPU accounting delta.
//
// The result is expressed as a percentage in the range 0 to 100.
// A zero total delta is invalid because there is no CPU accounting interval
// from which utilization can be calculated.
func Utilization(delta Delta) (float64, error) {
	total := delta.User +
		delta.Nice +
		delta.System +
		delta.Idle +
		delta.IOWait +
		delta.IRQ +
		delta.SoftIRQ +
		delta.Steal

	if total == 0 {
		return 0, fmt.Errorf("cannot calculate CPU utilization: total delta is zero")
	}

	busy := delta.User +
		delta.Nice +
		delta.System +
		delta.IRQ +
		delta.SoftIRQ +
		delta.Steal

	return float64(busy) / float64(total) * 100, nil
}

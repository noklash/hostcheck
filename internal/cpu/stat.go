package cpu

// Stat represents the cumulative CPU accounting counters
// exposed by Linux through /proc/stat.
//
// All counter values are measured in clock ticks.
type Stat struct {
	User      uint64
	Nice      uint64
	System    uint64
	Idle      uint64
	IOWait    uint64
	IRQ       uint64
	SoftIRQ   uint64
	Steal     uint64
	Guest     uint64
	GuestNice uint64
}

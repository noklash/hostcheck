package memory

// MemInfo contains the memory fields collected from /proc/meminfo.
// Values are kept in the units reported by Linux: kB.
type MemInfo struct {
	Total     uint64
	Available uint64
	Free      uint64
	Buffers   uint64
	Cached    uint64
	SwapTotal uint64
	SwapFree  uint64
}

package process

// Memory contains process memory measurements derived from
// /proc/<pid>/status.
//
// All values are expressed in bytes.
type Memory struct {
	VirtualBytes      uint64
	ResidentBytes     uint64
	AnonymousBytes    uint64
	FileBackedBytes   uint64
	SharedMemoryBytes uint64
}

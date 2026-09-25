package process

// Status contains process information derived from /proc/<pid>/status.
//
// Memory is nil for kernel threads because they do not have a userspace
// address space for the memory fields reported by /proc/<pid>/status.
type Status struct {
	Kthread bool
	Memory  *Memory
}

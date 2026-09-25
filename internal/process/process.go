package process

// Process represents a point-in-time process observation collected from
// multiple Linux /proc interfaces.
//
// Memory is nil for kernel threads because they do not have userspace
// memory accounting in /proc/<pid>/status.
type Process struct {
	Stats   Stats
	Kthread bool
	Memory  *Memory
}

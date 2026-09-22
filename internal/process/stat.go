package process

// Stats contains raw process accounting fields reported by
// /proc/<pid>/stat.
//
// Values retain the units used by the Linux kernel interface.
type Stats struct {
	PID       int64
	Comm      string
	State     byte
	PPID      int64
	UTime     uint64
	STime     uint64
	Priority  int64
	Nice      int64
	Threads   uint64
	StartTime uint64
	VSize     uint64
	RSSPages  int64
}

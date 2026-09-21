package cpu

// Delta represents CPU accounting accumulated between two samples.
type Delta struct {
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

// Delta returns the difference between two cumulative CPU accounting samples.
func (s Stat) Delta(previous Stat) Delta {
	return Delta{
		User:      s.User - previous.User,
		Nice:      s.Nice - previous.Nice,
		System:    s.System - previous.System,
		Idle:      s.Idle - previous.Idle,
		IOWait:    s.IOWait - previous.IOWait,
		IRQ:       s.IRQ - previous.IRQ,
		SoftIRQ:   s.SoftIRQ - previous.SoftIRQ,
		Steal:     s.Steal - previous.Steal,
		Guest:     s.Guest - previous.Guest,
		GuestNice: s.GuestNice - previous.GuestNice,
	}
}

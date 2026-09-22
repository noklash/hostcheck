package cpu

import "fmt"

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
//
// An error is returned if any counter regresses between samples.
func (s Stat) Delta(previous Stat) (Delta, error) {
	if s.User < previous.User {
		return Delta{}, fmt.Errorf("CPU user counter regressed: previous=%d current=%d", previous.User, s.User)
	}

	if s.Nice < previous.Nice {
		return Delta{}, fmt.Errorf("CPU nice counter regressed: previous=%d current=%d", previous.Nice, s.Nice)
	}

	if s.System < previous.System {
		return Delta{}, fmt.Errorf("CPU system counter regressed: previous=%d current=%d", previous.System, s.System)
	}

	if s.Idle < previous.Idle {
		return Delta{}, fmt.Errorf("CPU idle counter regressed: previous=%d current=%d", previous.Idle, s.Idle)
	}

	if s.IOWait < previous.IOWait {
		return Delta{}, fmt.Errorf("CPU iowait counter regressed: previous=%d current=%d", previous.IOWait, s.IOWait)
	}

	if s.IRQ < previous.IRQ {
		return Delta{}, fmt.Errorf("CPU irq counter regressed: previous=%d current=%d", previous.IRQ, s.IRQ)
	}

	if s.SoftIRQ < previous.SoftIRQ {
		return Delta{}, fmt.Errorf("CPU softirq counter regressed: previous=%d current=%d", previous.SoftIRQ, s.SoftIRQ)
	}

	if s.Steal < previous.Steal {
		return Delta{}, fmt.Errorf("CPU steal counter regressed: previous=%d current=%d", previous.Steal, s.Steal)
	}

	if s.Guest < previous.Guest {
		return Delta{}, fmt.Errorf("CPU guest counter regressed: previous=%d current=%d", previous.Guest, s.Guest)
	}

	if s.GuestNice < previous.GuestNice {
		return Delta{}, fmt.Errorf("CPU guest_nice counter regressed: previous=%d current=%d", previous.GuestNice, s.GuestNice)
	}

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
	}, nil
}

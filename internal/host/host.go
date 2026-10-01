package host

import (
	"time"

	"github.com/noklash/hostcheck/internal/cpu"
	"github.com/noklash/hostcheck/internal/filesystem"
	"github.com/noklash/hostcheck/internal/memory"
	"github.com/noklash/hostcheck/internal/network"
	"github.com/noklash/hostcheck/internal/process"
)

type Snapshot struct {
	ObservedAt time.Time

	CPU        *cpu.Stat
	Memory     *memory.MemInfo
	Filesystem *filesystem.Stats
	Processes  []process.Process
	Network    *network.Network

	Errors []CollectionError
}

type CollectionError struct {
	Subsystem string
	Err       error
}

package host

import (
	"time"

	"github.com/noklash/hostcheck/internal/cpu"
	"github.com/noklash/hostcheck/internal/filesystem"
	"github.com/noklash/hostcheck/internal/memory"
	"github.com/noklash/hostcheck/internal/network"
	"github.com/noklash/hostcheck/internal/process"
)

type Config struct {
	FilesystemPath string
}

func Collect(config Config) Snapshot {
	snapshot := Snapshot{
		ObservedAt: time.Now(),
		Errors:     make([]CollectionError, 0),
	}

	cpuStat, err := cpu.ReadStat()
	if err != nil {
		snapshot.Errors = append(snapshot.Errors, CollectionError{
			Subsystem: "cpu",
			Err:       err,
		})
	} else {
		snapshot.CPU = &cpuStat
	}

	memInfo, err := memory.ReadMemInfo()
	if err != nil {
		snapshot.Errors = append(snapshot.Errors, CollectionError{
			Subsystem: "memory",
			Err:       err,
		})
	} else {
		snapshot.Memory = &memInfo
	}

	filesystemPath := config.FilesystemPath
	if filesystemPath == "" {
		filesystemPath = "/"
	}

	filesystemStats, err := filesystem.ReadStats(filesystemPath)
	if err != nil {
		snapshot.Errors = append(snapshot.Errors, CollectionError{
			Subsystem: "filesystem",
			Err:       err,
		})
	} else {
		snapshot.Filesystem = &filesystemStats
	}

	processes, err := process.Collect()
	if err != nil {
		snapshot.Errors = append(snapshot.Errors, CollectionError{
			Subsystem: "process",
			Err:       err,
		})
	} else {
		snapshot.Processes = processes
	}

	networkSnapshot, err := network.ReadNetwork()
	if err != nil {
		snapshot.Errors = append(snapshot.Errors, CollectionError{
			Subsystem: "network",
			Err:       err,
		})
	} else {
		snapshot.Network = &networkSnapshot
	}

	return snapshot
}

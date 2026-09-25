package process

import (
	"fmt"
	"os"
)

// ReadStatus reads and parses /proc/<pid>/status.
func ReadStatus(pid int64) (Status, error) {
	if pid <= 0 {
		return Status{}, fmt.Errorf("invalid pid %d", pid)
	}

	path := fmt.Sprintf("/proc/%d/status", pid)

	data, err := os.ReadFile(path)
	if err != nil {
		return Status{}, fmt.Errorf("read %s: %w", path, err)
	}

	status, err := ParseStatus(string(data))
	if err != nil {
		return Status{}, fmt.Errorf("parse %s: %w", path, err)
	}

	return status, nil
}

// ReadMemory reads process memory data from /proc/<pid>/status.
//
// This compatibility helper returns an error when the process does not
// expose userspace memory accounting, such as a kernel thread.
func ReadMemory(pid int64) (Memory, error) {
	status, err := ReadStatus(pid)
	if err != nil {
		return Memory{}, err
	}

	if status.Memory == nil {
		return Memory{}, fmt.Errorf("process %d has no userspace memory accounting", pid)
	}

	return *status.Memory, nil
}

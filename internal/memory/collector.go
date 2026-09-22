package memory

import (
	"fmt"
	"io"
	"os"
)

// ReadMemInfo reads the current memory statistics from /proc/meminfo.
func ReadMemInfo() (MemInfo, error) {
	file, err := os.Open("/proc/meminfo")
	if err != nil {
		return MemInfo{}, fmt.Errorf("open /proc/meminfo: %w", err)
	}
	defer file.Close()

	return readMemInfo(file)
}

func readMemInfo(r io.Reader) (MemInfo, error) {
	data, err := io.ReadAll(r)
	if err != nil {
		return MemInfo{}, fmt.Errorf("read memory statistics: %w", err)
	}

	mem, err := ParseMemInfo(string(data))
	if err != nil {
		return MemInfo{}, fmt.Errorf("parse memory statistics: %w", err)
	}

	return mem, nil
}

package cpu

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"
)

// ReadStat reads the aggregate CPU accounting record from /proc/stat.
func ReadStat() (Stat, error) {
	file, err := os.Open("/proc/stat")
	if err != nil {
		return Stat{}, fmt.Errorf("open /proc/stat: %w", err)
	}
	defer file.Close()

	return readStat(file)
}

func readStat(reader io.Reader) (Stat, error) {
	scanner := bufio.NewScanner(reader)

	for scanner.Scan() {
		fields := strings.Fields(scanner.Text())

		if len(fields) == 0 || fields[0] != "cpu" {
			continue
		}

		stat, err := ParseStat(scanner.Text())
		if err != nil {
			return Stat{}, fmt.Errorf("parse aggregate CPU stat: %w", err)
		}

		return stat, nil
	}

	if err := scanner.Err(); err != nil {
		return Stat{}, fmt.Errorf("read /proc/stat: %w", err)
	}

	return Stat{}, fmt.Errorf("aggregate CPU record not found in /proc/stat")
}

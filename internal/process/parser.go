package process

import (
	"fmt"
	"strconv"
	"strings"
)

// ParseStat parses one complete /proc/<pid>/stat record.
//
// The comm field is enclosed in parentheses and may contain spaces and
// closing parentheses, so the record cannot be parsed with strings.Fields
// from the beginning.
func ParseStat(line string) (Stats, error) {
	line = strings.TrimSpace(line)
	if line == "" {
		return Stats{}, fmt.Errorf("empty stat record")
	}

	open := strings.IndexByte(line, '(')
	if open <= 0 {
		return Stats{}, fmt.Errorf("invalid stat record: missing comm opening parenthesis")
	}

	pidText := strings.TrimSpace(line[:open])
	pid, err := strconv.ParseInt(pidText, 10, 64)
	if err != nil {
		return Stats{}, fmt.Errorf("parse pid %q: %w", pidText, err)
	}

	close := findCommEnd(line, open)
	if close < 0 {
		return Stats{}, fmt.Errorf("invalid stat record: missing comm closing parenthesis")
	}

	comm := line[open+1 : close]

	suffix := strings.TrimSpace(line[close+1:])
	fields := strings.Fields(suffix)

	// The suffix begins at field 3 (state), so fields[0] is field 3.
	// We need through field 24 (rss), which requires 22 suffix fields.
	if len(fields) < 22 {
		return Stats{}, fmt.Errorf(
			"invalid stat record: expected at least 22 fields after comm, got %d",
			len(fields),
		)
	}

	state := fields[0]
	if len(state) != 1 {
		return Stats{}, fmt.Errorf("invalid process state %q", state)
	}

	ppid, err := parseInt64("ppid", fields[1])
	if err != nil {
		return Stats{}, err
	}

	utime, err := parseUint64("utime", fields[11])
	if err != nil {
		return Stats{}, err
	}

	stime, err := parseUint64("stime", fields[12])
	if err != nil {
		return Stats{}, err
	}

	priority, err := parseInt64("priority", fields[15])
	if err != nil {
		return Stats{}, err
	}

	nice, err := parseInt64("nice", fields[16])
	if err != nil {
		return Stats{}, err
	}

	threads, err := parseUint64("num_threads", fields[17])
	if err != nil {
		return Stats{}, err
	}

	startTime, err := parseUint64("starttime", fields[19])
	if err != nil {
		return Stats{}, err
	}

	vsize, err := parseUint64("vsize", fields[20])
	if err != nil {
		return Stats{}, err
	}

	rss, err := parseInt64("rss", fields[21])
	if err != nil {
		return Stats{}, err
	}

	return Stats{
		PID:       pid,
		Comm:      comm,
		State:     state[0],
		PPID:      ppid,
		UTime:     utime,
		STime:     stime,
		Priority:  priority,
		Nice:      nice,
		Threads:   threads,
		StartTime: startTime,
		VSize:     vsize,
		RSSPages:  rss,
	}, nil
}

// findCommEnd finds the closing parenthesis that terminates the comm field.
//
// A comm value may itself contain ')', so the first ')' is not necessarily
// the delimiter. The delimiter is identified by the state field immediately
// following it.
func findCommEnd(line string, open int) int {
	for i := len(line) - 1; i > open; i-- {
		if line[i] != ')' {
			continue
		}

		rest := line[i+1:]
		if len(rest) < 2 || rest[0] != ' ' {
			continue
		}

		fields := strings.Fields(rest)
		if len(fields) == 0 || len(fields[0]) != 1 {
			continue
		}

		return i
	}

	return -1
}

func parseInt64(name, value string) (int64, error) {
	n, err := strconv.ParseInt(value, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("parse %s %q: %w", name, value, err)
	}

	return n, nil
}

func parseUint64(name, value string) (uint64, error) {
	n, err := strconv.ParseUint(value, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("parse %s %q: %w", name, value, err)
	}

	return n, nil
}

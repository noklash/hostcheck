package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/noklash/hostcheck/internal/health"
	"github.com/noklash/hostcheck/internal/host"
)

const (
	degradedThresholdPercent = 20.0
	criticalThresholdPercent = 10.0
)

func main() {
	jsonOutput := flag.Bool(
		"json",
		false,
		"write the health result as JSON",
	)
	flag.Parse()

	snapshot := host.Collect(host.Config{})

	result, err := health.Evaluate(snapshot, defaultPolicy())
	if err != nil {
		fmt.Fprintf(os.Stderr, "health evaluation failed: %v\n", err)
		os.Exit(1)
	}

	if *jsonOutput {
		if err := printJSONResult(os.Stdout, snapshot, result); err != nil {
			fmt.Fprintf(os.Stderr, "JSON output failed: %v\n", err)
			os.Exit(1)
		}
	} else {
		if err := printResult(os.Stdout, snapshot, result); err != nil {
			fmt.Fprintf(os.Stderr, "output failed: %v\n", err)
			os.Exit(1)
		}
	}

	os.Exit(exitCode(result))
}

func defaultPolicy() health.SnapshotPolicy {
	return health.SnapshotPolicy{
		Memory: health.MemoryPolicy{
			DegradedBelowPercent: degradedThresholdPercent,
			CriticalBelowPercent: criticalThresholdPercent,
		},
		Filesystem: health.FilesystemPolicy{
			DegradedBelowPercent: degradedThresholdPercent,
			CriticalBelowPercent: criticalThresholdPercent,
		},
		FilesystemInode: health.FilesystemInodePolicy{
			DegradedBelowPercent: degradedThresholdPercent,
			CriticalBelowPercent: criticalThresholdPercent,
		},
	}
}

func exitCode(result health.Result) int {
	switch result.Coverage {
	case health.Unavailable:
		return 1

	case health.Complete, health.Partial:
		switch result.Status {
		case health.OK:
			return 0
		case health.Degraded:
			return 1
		case health.Critical:
			return 2
		}
	}

	return 1
}

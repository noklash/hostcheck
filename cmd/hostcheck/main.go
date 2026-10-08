package main

import (
	"flag"
	"fmt"
	"os"
	"time"

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
		printResult(snapshot, result)
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

func printResult(snapshot host.Snapshot, result health.Result) {
	fmt.Printf("hostcheck\n")
	fmt.Printf(
		"observed_at: %s\n",
		snapshot.ObservedAt.Format(time.RFC3339),
	)
	fmt.Printf("status: %s\n", result.Status)
	fmt.Printf("coverage: %s\n", result.Coverage)
	fmt.Printf("\n")

	for _, assessment := range result.Assessments {
		if assessment.Availability == health.Unassessable {
			fmt.Printf("[unassessable] %s\n", assessment.Subject)
		} else {
			fmt.Printf("[%s] %s\n", assessment.Status, assessment.Subject)
		}

		fmt.Printf("  %s\n", assessment.Reason)

		for _, evidence := range assessment.Evidence {
			fmt.Printf("  %s\n", evidence)
		}
	}
}

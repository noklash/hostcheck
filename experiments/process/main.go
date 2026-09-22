package main

import (
	"fmt"
	"log"

	"github.com/noklash/hostcheck/internal/process"
)

func main() {
	stats, err := process.Collect()
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("processes=%d\n", len(stats))

	for i, stat := range stats {
		if i >= 20 {
			break
		}

		fmt.Printf(
			"pid=%d comm=%q state=%c ppid=%d threads=%d utime=%d stime=%d\n",
			stat.PID,
			stat.Comm,
			stat.State,
			stat.PPID,
			stat.Threads,
			stat.UTime,
			stat.STime,
		)
	}
}

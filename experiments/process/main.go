package main

import (
	"fmt"
	"log"

	"github.com/noklash/hostcheck/internal/process"
)

func main() {
	processes, err := process.Collect()
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("processes=%d\n", len(processes))

	for i, p := range processes {
		if i >= 20 {
			break
		}

		stat := p.Stats

		fmt.Printf(
			"pid=%d comm=%q state=%c ppid=%d threads=%d utime=%d stime=%d",
			stat.PID,
			stat.Comm,
			stat.State,
			stat.PPID,
			stat.Threads,
			stat.UTime,
			stat.STime,
		)

		if p.Kthread {
			fmt.Printf(" kthread=true memory=unavailable")
		} else {
			fmt.Printf(
				" kthread=false vsize=%d rss=%d anon=%d file=%d shmem=%d",
				p.Memory.VirtualBytes,
				p.Memory.ResidentBytes,
				p.Memory.AnonymousBytes,
				p.Memory.FileBackedBytes,
				p.Memory.SharedMemoryBytes,
			)
		}

		fmt.Println()
	}
}

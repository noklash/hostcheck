package main

import (
	"fmt"
	"log"
	"os"
	"time"

	"github.com/noklash/hostcheck/internal/process"
)

func main() {
	if len(os.Args) != 2 {
		log.Fatalf("usage: go run ./experiments/process-cpu <pid>")
	}

	var pid int64

	if _, err := fmt.Sscanf(os.Args[1], "%d", &pid); err != nil {
		log.Fatalf("invalid pid %q: %v", os.Args[1], err)
	}

	firstStats, err := process.ReadStats(pid)
	if err != nil {
		log.Fatal(err)
	}

	first := process.NewSample(firstStats, time.Now())

	time.Sleep(time.Second)

	secondStats, err := process.ReadStats(pid)
	if err != nil {
		log.Fatal(err)
	}

	second := process.NewSample(secondStats, time.Now())

	delta, err := process.DeltaSamples(first, second)
	if err != nil {
		log.Fatal(err)
	}

	utilization, err := process.Utilization(delta)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("pid=%d\n", delta.PID)
	fmt.Printf("cpu_ticks=%d\n", delta.CPUTimeTicks)
	fmt.Printf("cpu_seconds=%.3f\n", delta.CPUTimeSeconds)
	fmt.Printf("elapsed_seconds=%.3f\n", delta.Elapsed.Seconds())
	fmt.Printf("utilization=%.2f%%\n", utilization*100)
}

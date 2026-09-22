package main

import (
	"fmt"
	"os"
	"time"

	"github.com/noklash/hostcheck/internal/cpu"
)

func main() {
	previous, err := cpu.ReadStat()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	time.Sleep(time.Second)

	current, err := cpu.ReadStat()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	delta, err := current.Delta(previous)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	utilization, err := cpu.Utilization(delta)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	fmt.Printf("CPU utilization: %.2f%%\n", utilization)
}

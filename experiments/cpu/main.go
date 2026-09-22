package main

import (
	"bufio"
	"fmt"
	"os"
	"time"

	"github.com/noklash/hostcheck/internal/cpu"
)

func readCPUStat() (cpu.Stat, error) {
	file, err := os.Open("/proc/stat")
	if err != nil {
		return cpu.Stat{}, fmt.Errorf("open /proc/stat: %w", err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)

	if !scanner.Scan() {
		if err := scanner.Err(); err != nil {
			return cpu.Stat{}, fmt.Errorf("read /proc/stat: %w", err)
		}

		return cpu.Stat{}, fmt.Errorf("/proc/stat is empty")
	}

	stat, err := cpu.ParseStat(scanner.Text())
	if err != nil {
		return cpu.Stat{}, fmt.Errorf("parse CPU stat: %w", err)
	}

	return stat, nil
}

func main() {
	previous, err := readCPUStat()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	time.Sleep(time.Second)

	current, err := readCPUStat()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	delta := current.Delta(previous)

	utilization, err := cpu.Utilization(delta)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	fmt.Printf("CPU utilization: %.2f%%\n", utilization)
}

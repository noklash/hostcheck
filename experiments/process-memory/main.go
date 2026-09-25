package main

import (
	"fmt"
	"log"
	"os"
	"syscall"

	"github.com/noklash/hostcheck/internal/process"
)

const mappingSize = 100 * 1024 * 1024

func main() {
	pid := int64(os.Getpid())
	pageSize := os.Getpagesize()

	before, err := collectSelf(pid)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("== before mmap ==")
	printMemory(before)

	mapping, err := syscall.Mmap(
		-1,
		0,
		mappingSize,
		syscall.PROT_READ|syscall.PROT_WRITE,
		syscall.MAP_PRIVATE|syscall.MAP_ANON,
	)
	if err != nil {
		log.Fatalf("mmap: %v", err)
	}

	defer func() {
		if err := syscall.Munmap(mapping); err != nil {
			log.Printf("munmap: %v", err)
		}
	}()

	afterMap, err := collectSelf(pid)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("\n== after mmap, before page touch ==")
	printMemory(afterMap)

	for offset := 0; offset < len(mapping); offset += pageSize {
		mapping[offset] = 1
	}

	afterTouch, err := collectSelf(pid)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("\n== after touching every page ==")
	printMemory(afterTouch)

	fmt.Println("\n== deltas ==")
	printDelta("mmap reservation", before, afterMap)
	printDelta("page touching", afterMap, afterTouch)
}

func collectSelf(pid int64) (process.Memory, error) {
	processes, err := process.Collect()
	if err != nil {
		return process.Memory{}, err
	}

	for _, p := range processes {
		if p.Stats.PID != pid {
			continue
		}

		if p.Kthread {
			return process.Memory{}, fmt.Errorf("current process classified as kernel thread")
		}

		if p.Memory == nil {
			return process.Memory{}, fmt.Errorf("current process has no memory data")
		}

		return *p.Memory, nil
	}

	return process.Memory{}, fmt.Errorf("current process PID %d not found", pid)
}

func printMemory(memory process.Memory) {
	fmt.Printf("virtual:   %12d bytes (%8.2f MiB)\n",
		memory.VirtualBytes,
		toMiB(memory.VirtualBytes),
	)
	fmt.Printf("resident:  %12d bytes (%8.2f MiB)\n",
		memory.ResidentBytes,
		toMiB(memory.ResidentBytes),
	)
	fmt.Printf("anonymous: %12d bytes (%8.2f MiB)\n",
		memory.AnonymousBytes,
		toMiB(memory.AnonymousBytes),
	)
	fmt.Printf("file-backed:%11d bytes (%8.2f MiB)\n",
		memory.FileBackedBytes,
		toMiB(memory.FileBackedBytes),
	)
	fmt.Printf("shared:    %12d bytes (%8.2f MiB)\n",
		memory.SharedMemoryBytes,
		toMiB(memory.SharedMemoryBytes),
	)
}

func printDelta(label string, before, after process.Memory) {
	fmt.Printf("%s:\n", label)
	fmt.Printf("  virtual:   %+d bytes (%+.2f MiB)\n",
		int64(after.VirtualBytes)-int64(before.VirtualBytes),
		toMiBDelta(after.VirtualBytes, before.VirtualBytes),
	)
	fmt.Printf("  resident:  %+d bytes (%+.2f MiB)\n",
		int64(after.ResidentBytes)-int64(before.ResidentBytes),
		toMiBDelta(after.ResidentBytes, before.ResidentBytes),
	)
	fmt.Printf("  anonymous: %+d bytes (%+.2f MiB)\n",
		int64(after.AnonymousBytes)-int64(before.AnonymousBytes),
		toMiBDelta(after.AnonymousBytes, before.AnonymousBytes),
	)
}

func toMiB(bytes uint64) float64 {
	return float64(bytes) / (1024 * 1024)
}

func toMiBDelta(after, before uint64) float64 {
	return float64(int64(after)-int64(before)) / (1024 * 1024)
}

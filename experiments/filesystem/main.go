package main

import (
	"fmt"
	"log"
	"os"

	"github.com/noklash/hostcheck/internal/filesystem"
)

func main() {
	path := "/"

	if len(os.Args) > 1 {
		path = os.Args[1]
	}

	stats, err := filesystem.ReadStats(path)
	if err != nil {
		log.Fatal(err)
	}

	usedBlocks, err := stats.UsedBlocks()
	if err != nil {
		log.Fatal(err)
	}

	availableBytes, err := stats.AvailableBytes()
	if err != nil {
		log.Fatal(err)
	}

	usedBytes, err := stats.UsedBytes()
	if err != nil {
		log.Fatal(err)
	}

	availablePercent, err := stats.AvailablePercent()
	if err != nil {
		log.Fatal(err)
	}

	usedInodes, err := stats.UsedInodes()
	if err != nil {
		log.Fatal(err)
	}

	availableInodePercent, err := stats.AvailableInodePercent()
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Path: %s\n", stats.Path)
	fmt.Printf("Block size: %d bytes\n", stats.BlockSize)
	fmt.Printf("Total blocks: %d\n", stats.BlocksTotal)
	fmt.Printf("Free blocks: %d\n", stats.BlocksFree)
	fmt.Printf("Available blocks: %d\n", stats.BlocksAvailable)
	fmt.Printf("Used blocks: %d\n", usedBlocks)
	fmt.Printf("Available capacity: %d bytes\n", availableBytes)
	fmt.Printf("Used capacity: %d bytes\n", usedBytes)
	fmt.Printf("Available capacity: %.2f%%\n", availablePercent)
	fmt.Printf("Total inodes: %d\n", stats.InodesTotal)
	fmt.Printf("Free inodes: %d\n", stats.InodesFree)
	fmt.Printf("Used inodes: %d\n", usedInodes)
	fmt.Printf("Available inodes: %.2f%%\n", availableInodePercent)
}

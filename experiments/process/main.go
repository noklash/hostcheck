package main

import (
	"fmt"
	"log"

	"github.com/noklash/hostcheck/internal/process"
)

func main() {
	pids, err := process.ListPIDs()
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("processes=%d\n", len(pids))

	for i, pid := range pids {
		if i >= 20 {
			break
		}

		fmt.Println(pid)
	}
}

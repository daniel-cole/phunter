package main

import (
	"flag"
	"fmt"
	"github.com/daniel-cole/phunter/process"
	"log"
)

func main() {
	var pid = flag.Int("pid", -1, "process id")
	flag.Parse()
	
	if *pid == -1 || *pid < 0 {
		log.Fatal("PID must be specified and be >= 0")
	}

	p := process.Process{ID: *pid}
	containerName, err := p.FindContainerName()
	if err != nil {
		log.Fatalf("Failed to find container name for %d", *pid)
	}
	fmt.Println(containerName)
}

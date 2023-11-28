package main

import (
	"fmt"
	"os"
	"testing"
	"uk.ac.bris.cs/gameoflife/gol"
)

const benchLength = 1000

func BenchmarkThreads(b *testing.B) {
	//name,time/op (ns/op),± - ADD TO START OF RESULTS.CSV
	// Run the benchmark for different numbers of workers -
	// change this in stubs, and running it with 1 worker,2 workers,3workers,4 workers
	os.Stdout = nil
	p := gol.Params{
		ImageWidth:  512,
		ImageHeight: 512,
		Threads:     1,
		Turns:       benchLength}

	benchmarkThreads := fmt.Sprintf("%dx%dx%d-%d", p.ImageWidth, p.ImageHeight, p.Turns, p.Threads)
	b.Run(benchmarkThreads, func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			events := make(chan gol.Event)
			// Call your distributor function with the appropriate parameters
			go gol.Run(p, events, nil)
		}
	})
}

//Sets up a distributed benchmark for different numbers of workers (threads).
//The BenchmarkDistributed function runs the benchmark for the specified parameters,
//and the inner b.Run function executes the distributed code for each iteration.

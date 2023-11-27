package main

import (
	"flag"
	"fmt"
	"net"
	"net/rpc"
	"time"
	"uk.ac.bris.cs/gameoflife/stubs"
	"uk.ac.bris.cs/gameoflife/util"
)

type WorkerOperations struct {
	kill bool
}

// makeWorld is a way to create empty worlds (or parts of worlds)
func makeWorld(height, width int) []util.BitArray {
	world := make([]util.BitArray, height) //grid [i][j], [i] represents the row index, [j] represents the column index
	for i := range world {
		world[i] = util.NewBitArray(width)
	}
	return world
}

// transformY deals with the wrap around of Y, i.e. negative values or values over the height
func transformY(value, height int) int {
	if value == -1 {
		return height - 1
	}
	return (value + height) % height
}

// threadScale Creates an array of length threads with the scale for each thread
func threadScale(height, threads int) []int {
	baseNumber, remainder := height/threads, height%threads

	scale := make([]int, threads)
	for i := 0; i < threads; i++ {
		if remainder > 0 {
			scale[i] = baseNumber + 1
			remainder--
		} else {
			scale[i] = baseNumber
		}
	}
	return scale
}

// countLiveNeighbors calculates the number of live neighbors around a given cell.
func countLiveNeighbors(x, y, w int, part []util.BitArray) int {
	liveNeighbors := 0
	directions := []struct{ dx, dy int }{
		{-1, -1}, {0, -1}, {1, -1},
		{-1, 0}, {1, 0},
		{-1, 1}, {0, 1}, {1, 1},
	}

	for _, dir := range directions {
		nx := transformY(x+dir.dx, w)
		ny := y + dir.dy
		if part[ny].GetBit(nx) == stubs.Alive {
			liveNeighbors++
		}
	}
	return liveNeighbors
}

// subDistributor is a routine to deal with smaller parts of the world, takes part []util.BitArray, which is part of the world with height + 2
func worker(scale, worldWidth int, part []util.BitArray, outChannel chan []util.BitArray) {
	outPart := makeWorld(scale, worldWidth)
	for y := 1; y < len(part)-1; y++ { // row by row, skipping the overlaps
		for x := 0; x < worldWidth; x++ { // each cell in row
			liveNeighbors := countLiveNeighbors(x, y, worldWidth, part)
			if part[y].GetBit(x) == stubs.Alive { //apply GoL rules
				if liveNeighbors < 2 || liveNeighbors > 3 { //less than 2 live neighbours or more than 3
					outPart[(y-1)].SetBit(x, stubs.Dead)
				} else {
					outPart[(y-1)].SetBit(x, stubs.Alive)
				}
			} else { //dead
				if liveNeighbors == 3 { //any dead cell with exactly three live neighbours becomes alive
					outPart[(y-1)].SetBit(x, stubs.Alive)
				}
			}
		}
	}
	outChannel <- outPart
}

// subDistributor is a routine to deal with smaller parts of the world, takes part []util.BitArray, which is part of the world with height + 2
func subDistributor(scale, worldWidth int, part []util.BitArray) []util.BitArray {
	outPart := make([]util.BitArray, 0)
	subScale := threadScale(scale, stubs.Threads)
	workerChannels := make([]chan []util.BitArray, stubs.Threads) // rows
	for i := 0; i < stubs.Threads; i++ {
		workerChannels[i] = make(chan []util.BitArray) //2d slice  //columns
	}

	//initiates go routines
	startY := 0
	endY := 0
	for i := range workerChannels {
		endY = startY + subScale[i] + 1
		// cuts up world into parts needed for each thread
		inPart := part[startY : endY+1]
		go worker(subScale[i], worldWidth, inPart, workerChannels[i])
		startY += subScale[i]
	}

	//receives response
	for _, ch := range workerChannels {
		workerPart := <-ch
		outPart = append(outPart, workerPart...)
	}
	return outPart
}

// Worker is an RPC call that takes performs the GOL logic for part of the world
func (w *WorkerOperations) Worker(request stubs.WorkerRequest, response *stubs.WorkerResponse) (err error) {
	response.OutPart = subDistributor(request.Scale, request.WorldWidth, request.InPart)
	return
}

// KillWorker is an RPC that stops the subDistributor running, it will only be called when Worker is not due to the nature of broker
func (w *WorkerOperations) KillWorker(_ struct{}, _ *struct{}) error {
	w.kill = true
	return nil
}

// main initialises the server & creates a way of killing through w.kill
func main() {
	pAddr := flag.String("port", "8030", "Port to listen on")
	flag.Parse()
	w := new(WorkerOperations)
	if err := rpc.Register(w); err != nil {
		fmt.Println(err)
	}
	listener, err := net.Listen("tcp", ":"+*pAddr)
	if err != nil {
		fmt.Println(err)
	}
	go func() {
		for {
			if w.kill {
				if err := listener.Close(); err != nil {
					fmt.Println()
				}
				break
			}
			time.Sleep(500 * time.Millisecond)
		}
	}()
	rpc.Accept(listener)
}

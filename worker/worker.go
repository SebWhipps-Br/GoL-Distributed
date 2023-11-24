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

var (
	done = false
)

type WorkerOperations struct {
	Up bool
}

/*
makeWorld is a way to create empty worlds (or parts of worlds)
*/
func makeWorld(height, width int) []util.BitArray {
	world := make([]util.BitArray, height) //grid [i][j], [i] represents the row index, [j] represents the column index
	for i := range world {
		world[i] = util.NewBitArray(width)
	}
	return world
}

/*
transformY deals with the wrap around of Y, i.e. negative values or values over the height
*/
func transformY(value, height int) int {
	if value == -1 {
		return height - 1
	}
	return (value + height) % height
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

/*
worker is a routine to deal with smaller parts of the world
takes part, which is part of the world with height + 2
*/
func worker(scale, worldWidth int, part []util.BitArray) []util.BitArray {
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
	return outPart
}

func (w *WorkerOperations) Worker(request stubs.WorkerRequest, response *stubs.WorkerResponse) (err error) {
	response.OutPart = worker(request.Scale, request.WorldWidth, request.InPart)
	return
}

func (w *WorkerOperations) KillWorker(_ struct{}, _ *stubs.StandardServerResponse) error {
	done = true
	return nil
}

func main() {
	// initialise server
	pAddr := flag.String("port", "8030", "Port to listen on")
	flag.Parse()
	//rand.Seed(time.Now().UnixNano())
	w := new(WorkerOperations)
	w.Up = true
	err := rpc.Register(w)
	if err != nil {
		fmt.Println(err)
	}
	listener, _ := net.Listen("tcp", ":"+*pAddr)

	go func() {
		for {
			if done {
				err := listener.Close()
				if err != nil {
					fmt.Println()
				}
				break
			}
			time.Sleep(500 * time.Millisecond)
		}
	}()
	rpc.Accept(listener)
}

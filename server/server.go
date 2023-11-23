package main

import (
	"flag"
	"fmt"
	"math/rand"
	"net"
	"net/rpc"
	"sync"
	"time"
	"uk.ac.bris.cs/gameoflife/stubs"
	"uk.ac.bris.cs/gameoflife/util"
)

//server is the worker

var (
	mutex sync.Mutex // Mutex for safe access to the global channel
)

// Result represents the result of the executeTurns function
type Result struct {
	World      []util.BitArray
	AliveCells int
}

type GameOfLifeOperations struct {
	World          []util.BitArray
	ResultChannel  chan Result
	CompletedTurns int
	halt           bool
	pause          bool
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

// AliveCount counts the number of alive cells in the world, and returns this as an int
func AliveCount(world []util.BitArray) int {
	count := 0
	for _, row := range world {
		for x := 0; x < row.Len(); x++ {
			if row.GetBit(x) == stubs.Alive {
				count++
			}
		}
	}
	return count
}

// countLiveNeighbors calculates the number of live neighbors around a given cell.
func countLiveNeighbors(x, y, w int, h int, world []util.BitArray) int {
	liveNeighbors := 0
	dx := []int{-1, 0, 1, -1, 1, -1, 0, 1}
	dy := []int{-1, -1, -1, 0, 0, 1, 1, 1}

	for i := 0; i < 8; i++ {
		ny := (y + dy[i] + h) % h
		nx := (x + dx[i] + w) % w
		if world[ny].GetBit(nx) == stubs.Alive {
			liveNeighbors++
		}
	}
	return liveNeighbors
}

// 1 worker to start with
func executeTurns(Turns int, Width int, Height int, g *GameOfLifeOperations) {
	nextWorld := makeWorld(Height, Width)
	//defer mutex.Unlock()
	//Execute all turns of the Game of Life.
	for g.CompletedTurns < Turns && !g.halt {
		for g.pause {
			time.Sleep(500 * time.Millisecond) // A short pause to avoid spinning
		}
		mutex.Lock()
		//iterate through each cell in the current world
		for y := 0; y < Height; y++ {
			for x := 0; x < Width; x++ {
				liveNeighbors := countLiveNeighbors(x, y, Width, Height, g.World)
				if g.World[y].GetBit(x) == stubs.Alive { //apply GoL rules
					//less than 2 live neighbours
					if liveNeighbors < 2 || liveNeighbors > 3 {
						nextWorld[y].SetBit(x, stubs.Dead)
					} else {
						nextWorld[y].SetBit(x, stubs.Alive)
					}
				} else { //any dead cell with exactly three live neighbours becomes alive
					if liveNeighbors == 3 {
						nextWorld[y].SetBit(x, stubs.Alive)
					} else {
						nextWorld[y].SetBit(x, stubs.Dead)
					}
				}
			}
		}
		for row := range g.World { // copy the inner slices of the world
			copy(g.World[row], nextWorld[row])
		}
		g.CompletedTurns++
		mutex.Unlock()
	}
	result := Result{World: g.World, AliveCells: AliveCount(g.World)}
	g.ResultChannel <- result
}

// RunGameOfLife is called to Run game of life, it must assume that it has already been called
func (g *GameOfLifeOperations) RunGameOfLife(req stubs.Request, res *stubs.Response) (err error) {
	g.CompletedTurns = 0
	g.World = req.World
	g.halt = false
	g.pause = false
	go executeTurns(req.Turns, req.ImageWidth, req.ImageHeight, g)
	// Wait for the result from the executeTurns
	result := <-g.ResultChannel
	res.NextWorld = result.World
	res.CompletedTurns = g.CompletedTurns
	return

}

// GetAliveCount is called when the 2-second timer calls it from the client
func (g *GameOfLifeOperations) GetAliveCount(_ struct{}, res *stubs.AliveCellsResponse) (err error) {
	mutex.Lock()
	res.AliveCellsCount = AliveCount(g.World)
	res.CompletedTurns = g.CompletedTurns
	mutex.Unlock()
	return
}

func (g *GameOfLifeOperations) GetCurrentWorld(_ struct{}, res *stubs.CurrentWorldResponse) (err error) {
	mutex.Lock()
	res.World = g.World
	res.CompletedTurns = g.CompletedTurns
	mutex.Unlock()
	return
}

func (g *GameOfLifeOperations) HaltServer(_ struct{}, res *stubs.HaltServerResponse) (err error) {
	mutex.Lock()
	defer mutex.Unlock() //when function finished you unlock
	g.halt = true
	res.Success = true
	return
}

func (g *GameOfLifeOperations) PauseServer(req stubs.PauseServerRequest, res *stubs.PauseServerResponse) (err error) {
	mutex.Lock()
	g.pause = req.Pause
	res.CompletedTurns = g.CompletedTurns
	mutex.Unlock()
	return
}

func main() {
	// initialise server
	pAddr := flag.String("port", "8030", "Port to listen on")
	flag.Parse()
	rand.Seed(time.Now().UnixNano())
	g := new(GameOfLifeOperations)
	g.ResultChannel = make(chan Result)
	g.halt = false
	err := rpc.Register(g)
	if err != nil {
		fmt.Println(err)
	}
	listener, _ := net.Listen("tcp", ":"+*pAddr)

	go func() {
		for {
			if g.halt {
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

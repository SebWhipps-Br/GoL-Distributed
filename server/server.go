package main

import (
	"flag"
	"math/rand"
	"net"
	"net/rpc"
	"sync"
	"time"
	"uk.ac.bris.cs/gameoflife/stubs"
	"uk.ac.bris.cs/gameoflife/util"
)

//server is the worker

const (
	Alive = true
	Dead  = false
)

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
			if row.GetBit(x) == Alive {
				count++
			}
		}
	}
	//fmt.Println(count)
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
		if world[ny].GetBit(nx) == Alive {
			liveNeighbors++
		}
	}
	return liveNeighbors
}

// 1 worker to start with
func executeTurns(Turns int, Width int, Height int, g *GameOfLifeOperations) {
	nextWorld := makeWorld(Height, Width)
	defer mutex.Unlock()
	//Execute all turns of the Game of Life.
	for g.CompletedTurns < Turns {
		mutex.Lock()
		//iterate through each cell in the current world
		for y := 0; y < Height; y++ {
			for x := 0; x < Width; x++ {
				liveNeighbors := countLiveNeighbors(x, y, Width, Height, g.World)
				if g.World[y].GetBit(x) == Alive { //apply GoL rules
					//less than 2 live neighbours
					if liveNeighbors < 2 || liveNeighbors > 3 {
						nextWorld[y].SetBit(x, Dead)
					} else {
						nextWorld[y].SetBit(x, Alive)
					}
				} else { //any dead cell with exactly three live neighbours becomes alive
					if liveNeighbors == 3 {
						nextWorld[y].SetBit(x, Alive)
					} else {
						nextWorld[y].SetBit(x, Dead)
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

func (g *GameOfLifeOperations) UpdateWorld(req stubs.Request, res *stubs.Response) (err error) {
	g.CompletedTurns = 0
	g.World = req.World
	go executeTurns(req.Turns, req.ImageWidth, req.ImageHeight, g)
	// Wait for the result from the executeTurns
	result := <-g.ResultChannel
	res.NextWorld = result.World
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

func main() {
	pAddr := flag.String("port", "8030", "Port to listen on")
	flag.Parse()
	rand.Seed(time.Now().UnixNano())
	g := new(GameOfLifeOperations)
	g.ResultChannel = make(chan Result)
	rpc.Register(g)
	listener, _ := net.Listen("tcp", ":"+*pAddr)
	defer listener.Close()
	rpc.Accept(listener)
}

package main

import (
	"flag"
	"math/rand"
	"net"
	"net/rpc"
	"time"
	"uk.ac.bris.cs/gameoflife/stubs"
	"uk.ac.bris.cs/gameoflife/util"
)

//server is the worker

const (
	Alive = true
	Dead  = false
)

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
func distributor(Turns int, World []util.BitArray, Width int, Height int) []util.BitArray {

	nextWorld := makeWorld(Height, Width)

	turn := 0
	//Execute all turns of the Game of Life.
	for turn < Turns {
		//iterate through each cell in the current world
		for y := 0; y < Height; y++ {
			for x := 0; x < Width; x++ {

				liveNeighbors := countLiveNeighbors(x, y, Width, Height, World)
				if World[y].GetBit(x) == Alive { //apply GoL rules
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
		for row := range World { // copy the inner slices of the world
			copy(World[row], nextWorld[row])
		}

		turn++
	}
	return World
}

type GameOfLifeOperations struct {
	World []util.BitArray
}

// still working on
func (g *GameOfLifeOperations) UpdateWorld(req stubs.Request, res *stubs.Response) (err error) {
	res.NextWorld = distributor(req.Turns, req.World, req.ImageWidth, req.ImageHeight)
	return
}

func main() {
	pAddr := flag.String("port", "8030", "Port to listen on")
	flag.Parse()
	rand.Seed(time.Now().UnixNano())
	rpc.Register(&GameOfLifeOperations{})
	listener, _ := net.Listen("tcp", ":"+*pAddr)
	defer listener.Close()
	rpc.Accept(listener)
}

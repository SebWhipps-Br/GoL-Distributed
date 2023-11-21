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
	interuptChannel chan rune  // Global channel for communication
	globalChannelM  sync.Mutex // Mutex for safe access to the global channel

	responseChannel chan int
)

// Result represents the result of the distributor function
type Result struct {
	World         []util.BitArray
	AliveCells    int
	InterruptData interface{}
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
func AliveCount(world []util.BitArray, turn int) int {
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
func distributor(Turns int, World []util.BitArray, Width int, Height int, resultChannel chan<- Result, g *GameOfLifeOperations) {

	nextWorld := makeWorld(Height, Width)

	halt := false
	//Execute all turns of the Game of Life.
	for g.CompletedTurns < Turns && !halt {
		select {
		/*
			//case key := <-keyPresses:
			case interruptData := <-interuptChannel:
				println('x')
				//globalChannelM.Lock()
				fmt.Printf("Received interrupt data: %v\n", interruptData)
				//globalChannelM.Unlock()

				if interruptData == 't' {
					aliveCells := AliveCount(World, Turns)
					result := Result{World: World, AliveCells: aliveCells}
					resultChannel <- result
				}

		*/
		//keypressed
		default:
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
				copy(g.World[row], nextWorld[row])
			}

			//cellCount := AliveCount(World, Turns)
			//fmt.Println(cellCount)

			g.CompletedTurns++
		}
	}
	aliveCells := AliveCount(World, Turns)
	result := Result{World: World, AliveCells: aliveCells}
	resultChannel <- result
}

// still working on
func (g *GameOfLifeOperations) UpdateWorld(req stubs.Request, res *stubs.Response) (err error) {
	g.CompletedTurns = 0
	go distributor(req.Turns, req.World, req.ImageWidth, req.ImageHeight, g.ResultChannel, g)
	// Wait for the result from the distributor
	result := <-g.ResultChannel
	res.NextWorld = result.World
	return

}

func (g *GameOfLifeOperations) Interrupt(req stubs.Interrupt, res *stubs.InterruptResponse) (err error) {
	/*
		mutex lock
		response := alivecells(g.world)
		mutex unlock
	*/

	globalChannelM.Lock()
	res.AliveCellsCount = AliveCount(g.World, g.CompletedTurns)

	globalChannelM.Unlock()
	/*
		fmt.Println("INTERRUPT")
		fmt.Println("req.Key:", req.Key)
		if req.Key == 't' {
			interuptChannel <- 't'
			fmt.Println('t')
			res.AliveCellsCount = <-responseChannel
		}
		//want to get number of alive cells

	*/
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

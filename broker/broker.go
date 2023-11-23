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

const threads = 4

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

func transformY(value, height int) int {
	if value == -1 {
		return height - 1
	}
	return (value + height) % height
}

func makeRpcCall(scale, worldWidth int, inPart []util.BitArray, clients []*rpc.Client, resultChannel chan<- stubs.WorkerResponse, wg *sync.WaitGroup) {
	defer wg.Done()

	// Establish RPC connection
	client, err := rpc.Dial("tcp", serverAddr)
	if err != nil {
		fmt.Println("Error connecting to server:", err)
		return
	}
	defer client.Close()

	// Perform the RPC call
	var serverResponse stubs.WorkerResponse
	request := stubs.WorkerRequest{
		Scale:      scale,
		WorldWidth: worldWidth,
		InPart:     inPart,
	}
	err = client.Call("Server.Method", request, &serverResponse)
	if err != nil {
		fmt.Println("RPC call error:", err)
		return
	}

	// Send the response through the channel
	resultChannel <- serverResponse
}

func executeTurns(Turns int, Width int, Height int, g *GameOfLifeOperations) {
	scale := threadScale(Height, threads)
	//defer mutex.Unlock()
	//Execute all turns of the Game of Life.
	for g.CompletedTurns < Turns && !g.halt {
		for g.pause {
			time.Sleep(500 * time.Millisecond) // A short pause to avoid spinning
		}
		mutex.Lock()
		nextWorld := make([]util.BitArray, 0)
		//iterate through each cell in the current world

		workerResponses := make([]stubs.WorkerResponse, threads) // rows
		clients := make
		//initiates go routines
		startY := 0 //inclusive
		endY := 0   //exclusive
		for i := range workerResponses {
			endY = startY + scale[i] //endY is exclusive
			// cuts up world into parts needed for each thread

			inPart := make([]util.BitArray, 0)
			for j := startY - 1; j < endY+1; j++ {
				inPart = append(inPart, g.World[transformY(j, Height)])
			}

			//TODO put RPC call here

			go func() {
				err := clients[i].Call(stubs.RunGameOfLife, request, response)
				dones[i] <- err
			}()

			//go worker(startY, scale[i], p.ImageWidth, inPart, workerResponses[i], turn, c)
			startY = endY
		}

		//receives response
		for _, ch := range workerChannels {
			nextWorld = append(nextWorld, <-ch...)
		}
		//copy nextWorld to world
		for row := range world {
			copy(world[row], nextWorld[row])
		}
		turn++
		cellsCount := AliveCount(world, turn, c)

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

func (g *GameOfLifeOperations) HaltServer(_ struct{}, res *stubs.StandardServerResponse) (err error) {
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

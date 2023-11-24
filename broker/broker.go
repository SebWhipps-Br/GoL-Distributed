package main

import (
	"flag"
	"fmt"
	"net"
	"net/rpc"
	"os"
	"sync"
	"time"
	"uk.ac.bris.cs/gameoflife/stubs"
	"uk.ac.bris.cs/gameoflife/util"
)

var mutex sync.Mutex // Mutex for safe access to the global channel

// Result represents the result of the executeTurns function
type Result struct {
	World      []util.BitArray
	AliveCells int
}

type GameOfLifeOperations struct {
	World          []util.BitArray //store the current complete world, only to be accessed with the mutex
	ResultChannel  chan Result
	CompletedTurns int
	haltTurns      bool
	pause          bool
	clients        []*rpc.Client
	killClients    bool
	killBroker     bool
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

// transformY ensures that negative values wrap around to avoid index out of range, and for the game to work properly when counting neighbours
func transformY(value, height int) int {
	if value == -1 {
		return height - 1
	}
	return (value + height) % height
}

// makeWorkerCall performs a call to a worker client and returns the processed part of the world
func makeWorkerCall(scale, worldWidth int, inPart []util.BitArray, client *rpc.Client, resultChannel chan []util.BitArray) {
	var workerResponse stubs.WorkerResponse
	request := stubs.WorkerRequest{
		Scale:      scale,
		WorldWidth: worldWidth,
		InPart:     inPart,
	}
	if err := client.Call(stubs.Worker, request, &workerResponse); err != nil {
		fmt.Println("RPC call error:", err)
	}
	resultChannel <- workerResponse.OutPart
}

// killWorkersCall kills all worker clients that it is given
func killWorkersCall(clients []*rpc.Client) {
	var workerResponse struct{}
	for i := range clients {
		if err := clients[i].Call(stubs.KillWorker, struct{}{}, &workerResponse); err != nil {
			fmt.Println("RPC call error:", err)
		}
	}
}

// connectToWorkers returns []rpc.Client, a slice of clients that are ready for RPC calls
func connectToWorkers(serverAddresses []string) []*rpc.Client {
	clients := make([]*rpc.Client, stubs.Threads)
	for i := range clients {
		dial, err := rpc.Dial("tcp", serverAddresses[i])
		clients[i] = dial
		if err != nil {
			fmt.Println(err)
		}
	}
	return clients
}

// executeTurns Carries out the turns of the game of life by calling the workers
func executeTurns(Turns int, Width int, Height int, g *GameOfLifeOperations) {
	scale := threadScale(Height, stubs.Threads)
	for g.CompletedTurns < Turns && !g.haltTurns {
		for g.pause {
			time.Sleep(500 * time.Millisecond) // A short pause to avoid spinning
		}
		mutex.Lock()
		nextWorld := make([]util.BitArray, 0)
		//iterate through each cell in the current world

		workerResponses := make([]chan []util.BitArray, stubs.Threads) // rows
		for i := 0; i < stubs.Threads; i++ {
			workerResponses[i] = make(chan []util.BitArray) //2d slice  //columns
		}
		//initiates go routines
		startY, endY := 0, 0 //inclusive, exclusive
		for i := range workerResponses {
			endY = startY + scale[i] //endY is exclusive

			// cuts up world into parts needed for each thread
			inPart := make([]util.BitArray, 0)
			for j := startY - 1; j < endY+1; j++ {
				inPart = append(inPart, g.World[transformY(j, Height)])
			}

			go makeWorkerCall(scale[i], Width, inPart, g.clients[i], workerResponses[i])

			startY = endY
		}
		//receives response
		for _, ch := range workerResponses {
			part := <-ch
			nextWorld = append(nextWorld, part...)
		}

		//copy nextWorld to world
		for row := range g.World {
			copy(g.World[row], nextWorld[row])
		}
		g.CompletedTurns++
		mutex.Unlock()
	}
	if g.killClients {
		killWorkersCall(g.clients)
		g.killBroker = true
	}
	result := Result{World: g.World, AliveCells: AliveCount(g.World)}
	g.ResultChannel <- result
}

// RunGameOfLife is called to Run game of life, it must assume that it has already been called
func (g *GameOfLifeOperations) RunGameOfLife(req stubs.Request, res *stubs.Response) (err error) {
	g.CompletedTurns = 0
	g.World = req.World
	g.haltTurns = false
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

// GetCurrentWorld is an RPC method, takes no request arguments and returns the world and number of turns completed
func (g *GameOfLifeOperations) GetCurrentWorld(_ struct{}, res *stubs.CurrentWorldResponse) (err error) {
	mutex.Lock()
	defer mutex.Unlock()
	res.World = g.World
	res.CompletedTurns = g.CompletedTurns
	return
}

// HaltTurns is an RPC method, it stops the execution of turns, the broker remains active
func (g *GameOfLifeOperations) HaltTurns(_ struct{}, _ *struct{}) (err error) {
	mutex.Lock()
	defer mutex.Unlock() //when function is finished, you unlock
	g.haltTurns = true
	return
}

// KillClients is an RPC method, it kills all the clients :)
func (g *GameOfLifeOperations) KillClients(_ struct{}, _ *struct{}) (err error) {
	mutex.Lock()
	defer mutex.Unlock() //when function finished you unlock
	g.killClients = true
	return
}

// PauseServer toggles pausing of the execution of turns
func (g *GameOfLifeOperations) PauseServer(_ struct{}, res *stubs.PauseServerResponse) (err error) {
	mutex.Lock()
	defer mutex.Unlock()
	res.CompletedTurns = g.CompletedTurns
	g.pause = !g.pause
	return
}

// main initialises the server and adds a way to kill it
func main() {
	pAddr := flag.String("port", "8030", "Port to listen on")
	flag.Parse()
	g := new(GameOfLifeOperations)
	g.ResultChannel = make(chan Result)
	g.haltTurns = false
	g.killBroker = false
	serverAddresses := make([]string, 4)

	if len(os.Args) == 5 {
		copy(serverAddresses, os.Args[1:5])
		fmt.Println("#USING ARGUMENT ADDRESSES")
	} else {
		serverAddresses = []string{
			"127.0.0.1:8031",
			"127.0.0.1:8032",
			"127.0.0.1:8033",
			"127.0.0.1:8034",
		}
		fmt.Println("#USING DEFAULT ADDRESSES")
	}

	g.clients = connectToWorkers(serverAddresses)
	if err := rpc.Register(g); err != nil {
		fmt.Println(err)
	}
	listener, err2 := net.Listen("tcp", ":"+*pAddr)
	if err2 != nil {
		fmt.Println(err2)
	}

	go func() {
		for {
			if g.killBroker {
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

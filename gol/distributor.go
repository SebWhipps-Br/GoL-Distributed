package gol

import (
	"fmt"
	"net/rpc"
	"os"
	"strconv"
	"time"
	"uk.ac.bris.cs/gameoflife/stubs"
	"uk.ac.bris.cs/gameoflife/util"
)

type distributorChannels struct {
	events     chan<- Event
	ioCommand  chan<- ioCommand
	ioIdle     <-chan bool
	ioFilename chan<- string
	ioOutput   chan<- uint8
	ioInput    <-chan uint8
}

/*
outputWorld sends the image out byte by byte via the appropriate channels
*/
func outputWorld(height, width, turn int, world []util.BitArray, filename string, c distributorChannels) {
	c.ioCommand <- ioOutput
	c.ioFilename <- fmt.Sprintf("%sx%d", filename, turn)
	for i := 0; i < height; i++ { // each row
		for j := 0; j < width; j++ { //each column
			c.ioOutput <- world[i].GetBitToUint8(j)
		}
	}
	c.events <- ImageOutputComplete{CompletedTurns: turn, Filename: filename}
}

// finalAliveCount gives a slice of util.Cell containing the coordinates of all the alive cells
func finalAliveCount(world []util.BitArray) []util.Cell {
	var aliveCells []util.Cell
	for y, row := range world {
		for x := 0; x < row.Len(); x++ {
			if row.GetBit(x) == stubs.Alive {
				aliveCells = append(aliveCells, util.Cell{X: x, Y: y})
			}
		}
	}
	return aliveCells
}

// makeWorld is a way to create empty worlds (or parts of worlds)
func makeWorld(height, width int) []util.BitArray {
	world := make([]util.BitArray, height) //grid [i][j], [i] represents the row index, [j] represents the column index
	for i := range world {
		world[i] = util.NewBitArray(width)
	}
	return world
}

// getCurrentWorld makes an RPC call to get the last fully updated world, with the turn number of that world
func getCurrentWorld(client *rpc.Client) *stubs.CurrentWorldResponse {
	worldResponse := new(stubs.CurrentWorldResponse)
	if err := client.Call(stubs.GetCurrentWorld, struct{}{}, worldResponse); err != nil {
		fmt.Println(err)
	}
	return worldResponse
}

// regularAliveCount makes an RPC call to the server to retrieve the alive cell count and the turn number and passes this to events
func regularAliveCount(client *rpc.Client, c distributorChannels) {
	response := new(stubs.AliveCellsResponse)
	if err := client.Call(stubs.GetAliveCount, struct{}{}, response); err != nil {
		fmt.Println(err)
	}
	c.events <- AliveCellsCount{CellsCount: response.AliveCellsCount, CompletedTurns: response.CompletedTurns}
}

// haltTurns stops the broker running the game of life until runGameOfLife is called again
func haltTurns(client *rpc.Client) {
	haltServerResponse := new(struct{})
	if err := client.Call(stubs.HaltTurns, struct{}{}, haltServerResponse); err != nil {
		fmt.Println(err)
	}
}

// handlePause blocks other key presses until it p is pressed and pauses the broker and workers
func handlePause(client *rpc.Client, keyPresses <-chan rune) {
	pause := true
	var empty struct{}
	turnResponse := new(stubs.PauseServerResponse)

	if err := client.Call(stubs.PauseServer, empty, turnResponse); err != nil {
		fmt.Println(err)
	}
	fmt.Println("#PAUSED\nCompleted Turns", turnResponse.CompletedTurns)
	for pause {
		select {
		case k := <-keyPresses:
			if k == 'p' {
				if err := client.Call(stubs.PauseServer, empty, &empty); err != nil {
					fmt.Println(err)
				} else {
					pause = false
					fmt.Println("#CONTINUING")
				}
			}
		}
	}
}

// handleKeyPresses takes a keypress and acts accordingly, it returns a boolean value indicting whether the program should halt
func handleKeyPresses(key rune, keyPresses <-chan rune, p Params, c distributorChannels, client *rpc.Client, filename string) bool {
	switch key {
	case 's': // save: outputs current world
		worldResponse := getCurrentWorld(client)
		outputWorld(p.ImageHeight, p.ImageWidth, worldResponse.CompletedTurns, worldResponse.World, filename, c)
	case 'q': // quit: ends the client program
		worldResponse := getCurrentWorld(client)
		haltTurns(client)
		exit(p, c, worldResponse.CompletedTurns, worldResponse.World, filename)
		return true
	case 'k': //kill: shuts down the workers, then broker, then client
		haltClientResponse := new(struct{})
		if err := client.Call(stubs.KillClients, struct{}{}, haltClientResponse); err != nil {
			fmt.Println(err)
		}
		haltTurns(client)
	case 'p': //pause
		handlePause(client, keyPresses)
	}
	return false
}

// exit saves the world in its current state and ensures that the program stops gracefully
func exit(p Params, c distributorChannels, turnsCompleted int, world []util.BitArray, filename string) {
	// Report the final state using FinalTurnCompleteEvent.
	c.events <- FinalTurnComplete{CompletedTurns: turnsCompleted, Alive: finalAliveCount(world)}
	outputWorld(p.ImageHeight, p.ImageWidth, turnsCompleted, world, filename, c)

	// Make sure that the Io has finished any output before exiting.
	c.ioCommand <- ioCheckIdle
	<-c.ioIdle
	c.events <- StateChange{turnsCompleted, Quitting}
	// Close the channel to stop the SDL goroutine gracefully. Removing may cause deadlock.
	close(c.events)
}

// runGameOfLife starts running the GoL through the broker
func runGameOfLife(client *rpc.Client, p Params, c distributorChannels, keyPresses <-chan rune) {
	timer := time.NewTimer(2 * time.Second)
	done := make(chan error)
	resume := p.Turns >= 1000000 //10000000000 - if it is `run .` this is the case. perhaps there is a more exact way of doing this

	filename := strconv.Itoa(p.ImageWidth) + "x" + strconv.Itoa(p.ImageHeight)

	// Create the world and nextWorld as 2D slices
	world := makeWorld(p.ImageWidth, p.ImageHeight)

	// Loads world from input
	c.ioCommand <- ioInput               // Triggers ReadPgmImage()
	c.ioFilename <- filename             // ReadPgmImage waits for this filename
	for i := 0; i < p.ImageHeight; i++ { // each row
		for j := 0; j < p.ImageWidth; j++ { // each value in row
			world[i].SetBitFromUint8(j, <-c.ioInput) //byte by byte pixels of image
		}
	}
	turns := p.Turns
	width := p.ImageWidth
	height := p.ImageHeight

	request := stubs.Request{Turns: turns, ImageWidth: width, ImageHeight: height, World: world, Resume: resume}
	response := new(stubs.Response)
	go func() {
		err := client.Call(stubs.RunGameOfLife, request, response)
		done <- err
	}()

	halt := false
	for !halt {
		select {
		case err := <-done:
			if err != nil {
				fmt.Println(err)
			}
			exit(p, c, response.CompletedTurns, response.NextWorld, filename)
			halt = true
		case k := <-keyPresses:
			halt = handleKeyPresses(k, keyPresses, p, c, client, filename)
		case <-timer.C:
			regularAliveCount(client, c)
			timer.Reset(2 * time.Second)
		}
	}
}

// distributor divides the work between workers and interacts with other goroutines.
func distributor(p Params, c distributorChannels, keyPresses <-chan rune) {
	var serverAddress string
	if len(os.Args) == 2 {
		serverAddress = os.Args[1]
		fmt.Println("#USING ARGUMENT ADDRESS")
	} else {
		serverAddress = "127.0.0.1:8030"
		fmt.Println("#USING DEFAULT ADDRESS")
	}
	client, err := rpc.Dial("tcp", serverAddress)
	if err != nil {
		fmt.Println(err)
		return
	}
	defer func(client *rpc.Client) {
		if err := client.Close(); err != nil {
			fmt.Println(err)
			return
		}
	}(client)
	runGameOfLife(client, p, c, keyPresses)
}

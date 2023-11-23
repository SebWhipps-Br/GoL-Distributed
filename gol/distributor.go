package gol

import (
	"fmt"
	"net/rpc"
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

func haltServer(client *rpc.Client) {
	haltServerResponse := new(stubs.HaltServerResponse)
	err2 := client.Call(stubs.HaltServer, struct{}{}, haltServerResponse)
	if err2 != nil {
		fmt.Println(err2)
	}
}

func getCurrentWorld(client *rpc.Client) *stubs.CurrentWorldResponse {
	worldResponse := new(stubs.CurrentWorldResponse)
	err := client.Call(stubs.GetCurrentWorld, struct{}{}, worldResponse)
	if err != nil {
		fmt.Println(err)
	}
	return worldResponse
}

func regularAliveCount(client *rpc.Client, c distributorChannels) {
	response := new(stubs.AliveCellsResponse)
	err := client.Call(stubs.GetAliveCount, struct{}{}, response)
	if err != nil {
		fmt.Println(err)
	}
	c.events <- AliveCellsCount{CellsCount: response.AliveCellsCount, CompletedTurns: response.CompletedTurns}
}

// handleKeyPresses takes a keypress and acts accordingly, it returns a boolean value indicting whether the program should halt
func handleKeyPresses(key rune, keyPresses <-chan rune, p Params, c distributorChannels, client *rpc.Client, filename string) bool {
	switch key {
	case 's':
		worldResponse := getCurrentWorld(client)
		outputWorld(p.ImageHeight, p.ImageWidth, worldResponse.CompletedTurns, worldResponse.World, filename, c)
	case 'q': // ends the client program without stopping the server, must be able to be called again without failure
		worldResponse := getCurrentWorld(client)
		exit(p, c, worldResponse.CompletedTurns, worldResponse.World, filename)
		return true
	case 'k':
		haltServer(client)
	case 'p':
		pause := true
		//must pass server
		request := stubs.PauseServerRequest{Pause: true}
		turnResponse := new(stubs.PauseServerResponse)
		err := client.Call(stubs.PauseServer, request, turnResponse)
		fmt.Println("Completed Turns when paused: ", turnResponse.CompletedTurns)
		if err != nil {
			fmt.Println("err")
		}
		for pause {
			select {
			case k := <-keyPresses:
				pause = k != 'p'
				request := stubs.PauseServerRequest{Pause: false}
				response := new(stubs.Response)
				err2 := client.Call(stubs.PauseServer, request, response)
				if err2 != nil {
					fmt.Println(err2)
				}
				fmt.Println("Continuing")
			}
		}
	}
	return false
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

func makeCall(client *rpc.Client, p Params, c distributorChannels, keyPresses <-chan rune) {
	timer := time.NewTimer(2 * time.Second)
	done := make(chan error)

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

	request := stubs.Request{Turns: turns, ImageWidth: width, ImageHeight: height, World: world}
	response := new(stubs.Response)
	go func() {
		err := client.Call(stubs.Handler, request, response)
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
	serverAddress := "127.0.0.1:8030"
	client, err := rpc.Dial("tcp", serverAddress)
	if err != nil {
		fmt.Println(err)
	}
	defer func(client *rpc.Client) {
		err := client.Close()
		if err != nil {
			fmt.Println(err)
		}
	}(client)
	makeCall(client, p, c, keyPresses)
}

package gol

import (
	"flag"
	"fmt"
	"net/rpc"
	"strconv"
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

const (
	Alive = true
	Dead  = false
)

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
			if row.GetBit(x) == Alive {
				aliveCells = append(aliveCells, util.Cell{X: x, Y: y})
			}
		}
	}
	return aliveCells
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
func makeCall(client *rpc.Client, p Params, c distributorChannels) {
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
	client.Call(stubs.Handler, request, response)

	// Report the final state using FinalTurnCompleteEvent.
	c.events <- FinalTurnComplete{CompletedTurns: p.Turns, Alive: finalAliveCount(response.NextWorld)}

	outputWorld(p.ImageHeight, p.ImageWidth, p.Turns, response.NextWorld, filename, c)

	// Make sure that the Io has finished any output before exiting.
	c.ioCommand <- ioCheckIdle
	<-c.ioIdle
	c.events <- StateChange{p.Turns, Quitting}
	// Close the channel to stop the SDL goroutine gracefully. Removing may cause deadlock.
	close(c.events)

}

// distributor divides the work between workers and interacts with other goroutines.
func distributor(p Params, c distributorChannels) {
	server := flag.String("server", "127.0.0.1:8030", "IP:port string to connect to as server")
	flag.Parse()
	client, _ := rpc.Dial("tcp", *server)
	defer client.Close()
	makeCall(client, p, c)

}

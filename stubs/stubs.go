package stubs

import "uk.ac.bris.cs/gameoflife/util"

// capital letters for exported types
// game of life operations and processed turns
var Handler = "GameOfLifeOperations.UpdateWorld"
var InterruptHandler = "GameOfLifeOperations.Interrupt"

// final world returned
type Response struct {
	NextWorld []util.BitArray
}

// contains num of turns, 2d slice (initial state), size of image
type Request struct {
	Turns       int
	ImageWidth  int
	ImageHeight int
	World       []util.BitArray
}

type Interrupt struct {
	Key rune
}

type InterruptResponse struct {
	AliveCellsCount int
	CompletedTurns  int
}

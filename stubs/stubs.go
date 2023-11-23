package stubs

import "uk.ac.bris.cs/gameoflife/util"

// capital letters for exported types
// game of life operations and processed turns
var Handler = "GameOfLifeOperations.UpdateWorld"
var GetAliveCount = "GameOfLifeOperations.GetAliveCount"
var GetCurrentWorld = "GameOfLifeOperations.GetCurrentWorld"
var HaltServer = "GameOfLifeOperations.HaltServer"
var PauseServer = "GameOfLifeOperations.PauseServer"

// final world returned
type Response struct {
	NextWorld      []util.BitArray
	CompletedTurns int
}

// contains num of turns, 2d slice (initial state), size of image
type Request struct {
	Turns       int
	ImageWidth  int
	ImageHeight int
	World       []util.BitArray
}

type AliveCellsResponse struct {
	AliveCellsCount int
	CompletedTurns  int
}

type CurrentWorldResponse struct {
	World          []util.BitArray
	CompletedTurns int
}

type HaltServerResponse struct {
	Success bool
}

type PauseServerRequest struct {
	Pause bool
}

type PauseServerResponse struct {
	CompletedTurns int
}

package stubs

import (
	"uk.ac.bris.cs/gameoflife/util"
)

// capital letters for exported types
// game of life operations and processed turns

var RunGameOfLife = "GameOfLifeOperations.RunGameOfLife"
var GetAliveCount = "GameOfLifeOperations.GetAliveCount"
var GetCurrentWorld = "GameOfLifeOperations.GetCurrentWorld"
var HaltServer = "GameOfLifeOperations.HaltServer"
var PauseServer = "GameOfLifeOperations.PauseServer"
var KillServer = "GameOfLifeOperations.KillServer"

type Response struct {
	NextWorld      []util.BitArray
	CompletedTurns int
}

// Request contains num of turns, 2d slice (initial state), size of image
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

type StandardServerResponse struct {
	Success bool
}

type PauseServerRequest struct {
	Pause bool
}

type PauseServerResponse struct {
	CompletedTurns int
}

const (
	Alive = true
	Dead  = false
)

//////

var Worker = "WorkerOperations.Worker"

var KillWorker = "WorkerOperations.KillWorker"

type WorkerRequest struct {
	Scale      int
	WorldWidth int
	InPart     []util.BitArray
}

type WorkerResponse struct {
	OutPart []util.BitArray
}

const Threads = 4

package stubs

import (
	"uk.ac.bris.cs/gameoflife/util"
)

const (
	Alive = true
	Dead  = false
)

// distributor to broker

var RunGameOfLife = "GameOfLifeOperations.RunGameOfLife"
var GetAliveCount = "GameOfLifeOperations.GetAliveCount"
var GetCurrentWorld = "GameOfLifeOperations.GetCurrentWorld"
var HaltTurns = "GameOfLifeOperations.HaltTurns"
var PauseServer = "GameOfLifeOperations.PauseServer"
var HaltClient = "GameOfLifeOperations.HaltClient"

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

// broker to worker

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

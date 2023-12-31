package stubs

import (
	"uk.ac.bris.cs/gameoflife/util"
)

const (
	Alive   = true
	Threads = 4
)

// distributor to broker

var RunGameOfLife = "GameOfLifeOperations.RunGameOfLife"
var GetAliveCount = "GameOfLifeOperations.GetAliveCount"
var GetCurrentWorld = "GameOfLifeOperations.GetCurrentWorld"
var HaltTurns = "GameOfLifeOperations.HaltTurns"
var PauseServer = "GameOfLifeOperations.PauseServer"
var KillClients = "GameOfLifeOperations.KillClients"

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
	Resume      bool
}

type AliveCellsResponse struct {
	AliveCellsCount int
	CompletedTurns  int
}

type CurrentWorldResponse struct {
	World          []util.BitArray
	CompletedTurns int
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

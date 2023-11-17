package stubs

// capital letters for exported types
// game of life operations and processed turns
var Handler = "GameOfLifeOperations.UpdateWorld"

// final world returned
type Response struct {
	NextWorld [][]byte
}

// contains num of turns, 2d slice (initial state), size of image
type Request struct {
	Turns       int
	ImageWidth  int
	ImageHeight int
	World       [][]byte
}

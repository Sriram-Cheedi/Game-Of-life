package gol

import (
	"fmt"
	"log"
	"net/rpc"
	"sync"
	"time"
	"uk.ac.bris.cs/gameoflife/gol/stubs"
	"uk.ac.bris.cs/gameoflife/util"
)

type distributorChannels struct {
	events         chan<- Event
	ioCommand      chan<- ioCommand
	ioIdle         <-chan bool
	ioFilename     chan<- string
	ioOutput       chan<- uint8
	ioInput        <-chan uint8
	KeyPressesChan <-chan rune
}

// distributor divides the work between workers and interacts with other goroutines.
func distributor(p Params, c distributorChannels) {
	//setting command to input
	c.ioCommand <- ioInput
	//send the filename to the channel
	filename := fmt.Sprintf("%vx%v", p.ImageHeight, p.ImageWidth)
	c.ioFilename <- filename
	// TODO: Create a 2D slice to store the world.
	world := createBoard(p)
	//receive the data from the ioInput
	receiveInput(c.ioInput, world)
	initialAliveCells := calculateAliveCells(p, world)
	c.events <- CellsFlipped{CompletedTurns: 0, Cells: initialAliveCells}
	turn := 0

	// TODO: Execute all turns of the Game of Life.
	workerAddr := []string{"44.210.137.48:8030", "52.86.77.81:8040", "3.92.240.101:8050", "54.209.80.86:8060"}
	broker := "54.209.35.220:8020"

	client, err := rpc.Dial("tcp", broker)
	if err != nil {
		log.Fatalf("Failed to connect to server: %v", err)
	}
	defer client.Close()
	request := &stubs.Request{Matrix: world, Height: p.ImageHeight, Width: p.ImageWidth, NumberOfTurns: p.Turns, WorkerAddress: workerAddr, EndX: p.ImageWidth, Threads: p.Threads}
	response := new(stubs.Response)
	continueResponse := &stubs.Response{}
	err = client.Call(stubs.ContinueWorldHandler, request, continueResponse)

	if continueResponse.Continue {
		world = continueResponse.Matrix
		request.Matrix = world
		request.CurrTurns = continueResponse.CurrTurns
		fmt.Printf("Continuing From Turn %d\n", continueResponse.CurrTurns)
		c.events <- StateChange{continueResponse.CurrTurns, Executing}
	} else {
		c.events <- StateChange{turn, Executing}
	}
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()
	paused := false
	quit := false
	quitting := false
	quitChan := make(chan bool)
	var mutex sync.Mutex

	go func() {
		for {
			select {
			case quitting1 := <-quitChan:
				if quitting1 {
					break
				}
			case <-ticker.C:
				if !paused && !quit {
					newResponse := new(stubs.NewResponse)
					client.Call(stubs.ReportNumber0fAliveCells, request, newResponse)
					c.events <- AliveCellsCount{CompletedTurns: newResponse.CurrTurns, CellsCount: newResponse.NumberOfCellsAlive}
				}
			case keyPress := <-c.KeyPressesChan:
				switch keyPress {
				case 's':
					request1 := &stubs.KeyPressRequest{KeyPress: keyPress}
					newResponse := new(stubs.NewResponse)
					client.Call(stubs.KeyPressHandle, request1, newResponse)
					saveBoardAsPgm(p, c, newResponse.CurrWorld, newResponse.CurrTurns)
				case 'q':
					request1 := &stubs.KeyPressRequest{KeyPress: keyPress}
					newResponse := new(stubs.NewResponse)
					client.Call(stubs.KeyPressHandle, request1, newResponse)
					saveBoardAsPgm(p, c, newResponse.CurrWorld, newResponse.CurrTurns)
					c.events <- StateChange{newResponse.CurrTurns, Quitting}
					quit = true
				case 'k':
					request3 := &stubs.KeyPressRequest{KeyPress: keyPress}
					newResponse := new(stubs.NewResponse)
					client.Call(stubs.KeyPressHandle, request3, newResponse)
					saveBoardAsPgm(p, c, newResponse.CurrWorld, newResponse.CurrTurns)
					c.events <- StateChange{newResponse.CurrTurns, Quitting}
				case 'p':
					paused = !paused
					request4 := &stubs.KeyPressRequest{KeyPress: 'p', Paused: paused}
					newResponse := new(stubs.NewResponse)
					client.Call(stubs.KeyPressHandle, request4, newResponse)
					if paused {
						fmt.Println("Current turn: ", newResponse.CurrTurns)
						c.events <- StateChange{newResponse.CurrTurns, Paused}
					} else {
						fmt.Println(newResponse.Message)
						c.events <- StateChange{newResponse.CurrTurns, Executing}
					}

				}
			default:
				mutex.Lock()
				quit = quitting
				mutex.Unlock()

			}
		}
	}()

	done := make(chan bool)
	go func() {
		client1, err1 := rpc.Dial("tcp", broker)
		for i := 1; i <= p.Turns; i++ {
			var cellsToFlip []util.Cell
			reportRequest := &stubs.ReportRequest{}
			reportResponse := &stubs.ReportResponse{}

			if err1 != nil {
				log.Fatalf("Failed to connect to server: %v", err1)
			}
			client1.Call(stubs.ReportWorldAndTurn, reportRequest, reportResponse)

			previousWorld := reportResponse.PreviousWorld
			newWorld := reportResponse.Matrix
			for y := 0; y < p.ImageHeight; y++ {
				for x := 0; x < p.ImageWidth; x++ {
					if previousWorld[y][x] != newWorld[y][x] {
						cellsToFlip = append(cellsToFlip, util.Cell{X: x, Y: y})
					}
				}
			}

			completedTurns := reportResponse.CurrTurns
			if reportResponse.Kill {
				c.events <- CellsFlipped{CompletedTurns: completedTurns, Cells: cellsToFlip}
				c.events <- TurnComplete{CompletedTurns: completedTurns}
				done <- true
				return
			}
			c.events <- CellsFlipped{CompletedTurns: completedTurns, Cells: cellsToFlip}
			c.events <- TurnComplete{CompletedTurns: completedTurns}

		}
	}()

	err = client.Call(stubs.CallGameOfLife, request, response)

	if err != nil {
		log.Printf("error getting saved data: %v", err)
	}

	go func() {
		if p.Turns == 0 {
			done <- true
		}
	}()

	world = response.Matrix
	turn = response.CurrTurns
	mutex.Lock()
	quitting = true
	mutex.Unlock()
	quitChan <- true

	// TODO: Report the final state using FinalTurnCompleteEvent.
	c.events <- FinalTurnComplete{CompletedTurns: response.CurrTurns, Alive: calculateAliveCells(p, world)}
	c.ioCommand <- ioOutput
	filename1 := fmt.Sprintf("%vx%vx%v", p.ImageWidth, p.ImageHeight, p.Turns)
	c.ioFilename <- filename1
	sendOutput(c.ioOutput, world)
	<-done
	c.ioCommand <- ioCheckIdle
	<-c.ioIdle
	c.events <- StateChange{turn, Quitting}
	// Close the channel to stop the SDL goroutine gracefully. Removing may cause deadlock.

	close(c.events)
}

func calculateAliveCells(p Params, world [][]byte) []util.Cell {
	var aliveCells []util.Cell
	for i := 0; i < p.ImageHeight; i++ {
		for j := 0; j < p.ImageWidth; j++ {
			if world[i][j] == 255 {
				aliveCells = append(aliveCells, util.Cell{X: j, Y: i})
			}
		}
	}
	return aliveCells
}

func AliveCells(world [][]byte) int {
	var aliveCells int
	for i := 0; i < len(world); i++ {
		for j := 0; j < len(world[i]); j++ {
			if world[i][j] == 255 {
				aliveCells++
			}
		}
	}
	return aliveCells
}

func createBoard(p Params) [][]byte {
	world := make([][]byte, p.ImageHeight)
	for i := range world {
		world[i] = make([]byte, p.ImageWidth)
	}
	return world
}

func receiveInput(inputChan <-chan uint8, world [][]byte) {
	for i := range world {
		for j := range world[i] {
			world[i][j] = <-inputChan
		}

	}

}

func sendOutput(outputChan chan<- uint8, world [][]byte) {
	for i := range world {
		for j := range world[i] {
			outputChan <- world[i][j]
		}
	}
}

func saveBoardAsPgm(p Params, c distributorChannels, world [][]byte, turn int) {
	c.ioCommand <- ioOutput
	filename := fmt.Sprintf("%vx%vx%v", p.ImageWidth, p.ImageHeight, turn)
	c.ioFilename <- filename
	sendOutput(c.ioOutput, world)
	c.ioCommand <- ioCheckIdle
	<-c.ioIdle
	c.events <- ImageOutputComplete{CompletedTurns: turn, Filename: filename}
}

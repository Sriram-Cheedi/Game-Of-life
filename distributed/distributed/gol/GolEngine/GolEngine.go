package main

import (
	"flag"
	"log"
	"math/rand"
	"net"
	"net/rpc"
	"os"
	"sync"
	"time"
	"uk.ac.bris.cs/gameoflife/gol/stubs"
)

type GolOperation struct {
	end chan bool
}

func (g *GolOperation) CalculateNextState(req *stubs.Request, res *stubs.Response) (err error) {
	res.Matrix = parallelCalculateNextState(req.StartY, req.EndY, req.StartX, req.EndX, req.Matrix, req.Height, req.Width, req.Threads)
	return
}

func (g *GolOperation) KillServer(req *stubs.KillRequest, res *stubs.KillResponse) (err error) {
	go func() {
		time.Sleep(3 * time.Second)
		os.Exit(0)
	}()
	return
}

func parallelCalculateNextState(startY, endY, startX, endX int, world [][]byte, ImageHeight, ImageWidth, numThreads int) [][]byte {

	neighbors := [8][2]int{
		{-1, -1}, {0, -1}, {1, -1},
		{-1, 0}, {1, 0},
		{-1, 1}, {0, 1}, {1, 1},
	}
	height := endY - startY
	width := endX - startX
	newWorld := make([][]byte, height)
	for i := range newWorld {
		newWorld[i] = make([]byte, width)
	}
	var wg sync.WaitGroup
	rowsPerThread := height / numThreads
	for t := 0; t < numThreads; t++ {
		startRow := t * rowsPerThread
		endRow := (t + 1) * rowsPerThread
		if t == numThreads-1 {
			endRow = height
		}
		wg.Add(1)
		go func(startRow, endRow int) {
			defer wg.Done()
			for i := startRow; i < endRow; i++ {
				for j := 0; j < width; j++ {
					numberOfNeighborsAlive := 0

					for _, neighbor := range neighbors {
						neighborY := (i + startY + neighbor[0] + ImageHeight) % ImageHeight
						neighborX := (j + startX + neighbor[1] + ImageWidth) % ImageWidth
						if world[neighborY][neighborX] == 255 {
							numberOfNeighborsAlive++
						}
					}

					if world[startY+i][startX+j] == 255 {
						if numberOfNeighborsAlive < 2 || numberOfNeighborsAlive > 3 {
							newWorld[i][j] = 0
						} else {
							newWorld[i][j] = 255 // Stays alive
						}
					} else { // Cell is dead
						if numberOfNeighborsAlive == 3 {
							newWorld[i][j] = 255 // Becomes alive
						}
					}
				}
			}
		}(startRow, endRow)
	}
	wg.Wait()
	return newWorld
}

func main() {
	pAddr := flag.String("port", "8050", "Port to listen on")
	flag.Parse()
	rand.Seed(time.Now().UnixNano())
	end := make(chan bool)
	rpc.Register(&GolOperation{end})
	listener, err := net.Listen("tcp", ":"+*pAddr)
	if err != nil {
		log.Fatal("Listener Error:", err)
	}
	defer listener.Close()
	rpc.Accept(listener)
}

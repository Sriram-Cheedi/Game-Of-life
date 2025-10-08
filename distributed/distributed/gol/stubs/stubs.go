package stubs

import "uk.ac.bris.cs/gameoflife/util"

var CalculateGameOfLife = "GolOperation.CalculateNextState"
var CallGameOfLife = "GolOperation.CalculateGameOfLife"

var ReportNumber0fAliveCells = "GolOperation.ReportNumberOfAliveCells"
var KeyPressHandle = "GolOperation.KeyPressHandle"
var KillSever = "GolOperation.KillServer"
var ContinueWorldHandler = "GolOperation.GetContinue"
var ReportWorldAndTurn = "GolOperation.ReportTurnsAndWorld"

type ReportRequest struct {
}

type ReportResponse struct {
	Matrix        [][]byte
	CurrTurns     int
	CellsFlipped  []util.Cell
	PreviousWorld [][]byte
	Kill          bool
}
type Response struct {
	Matrix     [][]byte
	CurrTurns  int
	MatrixChan chan [][]byte
	Continue   bool
}

type Request struct {
	Matrix        [][]byte
	Height        int
	Width         int
	NumberOfTurns int
	StartX        int
	StartY        int
	EndX          int
	EndY          int
	WorkerAddress []string
	Threads       int
	CurrTurns     int
}

type KeyPressRequest struct {
	KeyPress      rune
	Paused        bool
	WorkerAddress string
}

type NewResponse struct {
	NumberOfCellsAlive int
	CurrTurns          int
	CurrWorld          [][]byte
	Message            string
}

type KillRequest struct {
	WorkerAddress string
}

type KillResponse struct {
	Message string
}

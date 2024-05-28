package main

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"image/color"
	g "qlp_server/game-controllers"
	"strconv"
)

var (
	GREY  color.RGBA
	RED   color.RGBA
	GREEN color.RGBA
)

type simulation struct {
	cellSize      fyne.Size
	width, height int
	grid          *fyne.Container
	gridObjects   [][]fyne.CanvasObject
	collisions    []g.Coordinate
	players       []*g.Player
	paths         [][]g.Cell
}

func NewSimulation(width, height int, cellWidth, cellHeight float32, collisions []g.Coordinate, playersCoordinates []g.Coordinate) *simulation {
	var players []*g.Player
	for _, coor := range playersCoordinates {
		players = append(players, g.NewPlayer(coor.X, coor.Y))
	}

	return &simulation{
		cellSize:   fyne.NewSize(cellWidth, cellHeight),
		width:      width,
		height:     height,
		collisions: collisions,
		players:    players,
	}
}

func (s *simulation) initColors() {
	RED = color.RGBA{R: 255, A: 100}
	GREY = color.RGBA{R: 210, G: 215, B: 211, A: 30}
	GREEN = color.RGBA{R: 60, G: 179, B: 113, A: 255}
}

func (s *simulation) startSimulation() {
	a := app.New()
	w := a.NewWindow("AI Visualization")
	w.Resize(fyne.NewSize(float32(s.width)*s.cellSize.Width, float32(s.height)*s.cellSize.Height))

	s.paths = g.GetPaths(s.width, s.height, s.collisions, s.players)
	s.initColors()
	s.drawGrid()

	w.SetContent(s.grid)
	w.ShowAndRun()
}

func (s *simulation) drawResults() {
}

func (s *simulation) createRectangle(rectText string, textColor color.Color, rectColor color.RGBA) *fyne.Container {
	text := canvas.NewText(rectText, textColor)
	text.Alignment = fyne.TextAlignCenter
	rect := canvas.NewRectangle(rectColor)
	rect.SetMinSize(s.cellSize)
	cell := container.NewMax(rect, text)
	return cell
}

func (s *simulation) addCollisions() {
	for ind := range s.collisions {
		collision := s.collisions[ind]
		s.gridObjects[collision.X][collision.Y] = s.createRectangle("C", color.White, RED)
	}
}

func (s *simulation) addPlayers() {
	for ind := range s.players {
		p := s.players[ind].GetPosition()
		s.gridObjects[p.X][p.Y] = s.createRectangle("P", color.White, GREEN)
	}
}

func (s *simulation) drawGrid() {
	var objects [][]fyne.CanvasObject

	for row := 0; row < s.height; row++ {
		var rowObjects []fyne.CanvasObject
		for column := 0; column < s.width; column++ {
			text := strconv.Itoa(s.paths[row][column].GetCellValue())
			cell := s.createRectangle(text, color.White, GREY)
			rowObjects = append(rowObjects, cell)
		}
		objects = append(objects, rowObjects)
	}

	s.gridObjects = objects
	s.addCollisions()
	s.addPlayers()
	var flatObjects []fyne.CanvasObject
	for _, row := range objects {
		flatObjects = append(flatObjects, row...)
	}

	grid := container.New(layout.NewGridLayoutWithColumns(s.width), flatObjects...)
	s.grid = grid
}

func main() {
	collisions := []g.Coordinate{{1, 2}, {1, 3}, {2, 3}}
	players := []g.Coordinate{{2, 2}, {5, 5}, {9, 2}}
	sm := NewSimulation(10, 10, 50.0, 50.0, collisions, players)
	sm.startSimulation()
}

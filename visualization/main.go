package main

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"github.com/ungerik/go3d/vec2"
	"image/color"
	g "qlp_server/game-controllers"
	"strconv"
)

var (
	BLACK       color.RGBA
	GREY        color.RGBA
	RED         color.RGBA
	GREEN       color.RGBA
	BLUE        color.RGBA
	PINK        color.RGBA
	ORANGE      color.RGBA
	PURPLE      color.RGBA
	CELL_SIZE   fyne.Size
	WINDOW_SIZE float32

	PLAYERS_COLORS []color.RGBA
)

type Simulation struct {
	width, height int
	grid          *fyne.Container
	collisions    []g.Coordinate
	players       []*g.Player
	graph         *[][]g.Cell
}

func NewSimulation(width, height int, collisions []g.Coordinate, playersCoordinates []g.Coordinate) *Simulation {
	var players []*g.Player

	WINDOW_SIZE = 800.0
	cellWidth := WINDOW_SIZE / float32(width)
	CELL_SIZE = fyne.NewSize(cellWidth, cellWidth)
	initColors()

	for ind, coor := range playersCoordinates {
		players = append(players, g.NewPlayer(coor.X, coor.Y, PLAYERS_COLORS[ind]))
	}

	return &Simulation{
		width:      width,
		height:     height,
		collisions: collisions,
		players:    players,
	}
}

func initColors() {
	RED = color.RGBA{R: 255, A: 100}
	BLACK = color.RGBA{R: 60, G: 60, B: 60, A: 5}
	GREY = color.RGBA{R: 210, G: 215, B: 211, A: 30}
	GREEN = color.RGBA{R: 60, G: 179, B: 113, A: 255}
	BLUE = color.RGBA{B: 255, A: 100}
	PINK = color.RGBA{R: 238, G: 130, B: 238, A: 255}
	ORANGE = color.RGBA{R: 255, G: 99, B: 71, A: 100}
	PURPLE = color.RGBA{R: 177, B: 252, A: 100}

	PLAYERS_COLORS = append(PLAYERS_COLORS, GREEN, BLUE, PINK, ORANGE, PURPLE)
}

func (s *Simulation) startSimulation() {
	a := app.New()
	w := a.NewWindow("AI Visualization")
	w.Resize(fyne.NewSize(WINDOW_SIZE, WINDOW_SIZE))

	s.graph = g.GetPaths(s.width, s.height, s.collisions, s.players)
	s.createGrid()

	w.SetContent(s.grid)
	w.ShowAndRun()
}

func (s *Simulation) createRectangle(rectText string, textColor color.Color, rectColor color.RGBA) *fyne.Container {
	text := canvas.NewText(rectText, textColor)
	text.Alignment = fyne.TextAlignCenter
	rect := canvas.NewRectangle(rectColor)
	rect.SetMinSize(CELL_SIZE)
	cell := container.NewMax(rect, text)
	return cell
}

func (s *Simulation) addCollisions(objects *[][]fyne.CanvasObject) {
	for collision := range s.collisions {
		coordinate := s.collisions[collision]
		(*objects)[coordinate.Y][coordinate.X] = s.createRectangle("C", color.White, BLACK)
	}
}

func (s *Simulation) addPlayers(objects *[][]fyne.CanvasObject) {
	for player := range s.players {
		coordinate := s.players[player].GetPosition()
		rgba := s.players[player].GetColor()
		(*objects)[coordinate.Y][coordinate.X] = s.createRectangle("P", color.White, rgba)
	}
}

func (s *Simulation) addLine(vector vec2.T, cell *fyne.Container) {
	line := canvas.NewLine(color.White)
	x := vector[0]
	y := vector[1]
	line.Position2 = fyne.NewPos(x, y)
	cell.Add(line)
}

func (s *Simulation) createGrid() {
	var objects [][]fyne.CanvasObject

	for row := 0; row < s.height; row++ {
		var rowObjects []fyne.CanvasObject
		for column := 0; column < s.width; column++ {
			gridCell := (*s.graph)[row][column]
			text := strconv.Itoa(gridCell.GetCellValue())
			cell := s.createRectangle(text, color.White, GREY)
			rowObjects = append(rowObjects, cell)
		}
		objects = append(objects, rowObjects)
	}

	s.addCollisions(&objects)
	s.addPlayers(&objects)
	var flatObjects []fyne.CanvasObject
	for _, row := range objects {
		flatObjects = append(flatObjects, row...)
	}

	grid := container.NewGridWithColumns(s.width, flatObjects...)
	s.grid = grid
}

func main() {
	collisions := []g.Coordinate{{4, 0}}
	players := []g.Coordinate{{3, 3}}
	sm := NewSimulation(5, 5, collisions, players)
	sm.startSimulation()
}

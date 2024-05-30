package main

import (
	"fmt"
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
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
	enemies       []*g.Enemy
	paths         [][]g.Cell
}

func NewSimulation(width, height int, collisions []g.Coordinate, playersCoordinates []g.Coordinate, enemiesCoordinates []g.Coordinate) *Simulation {
	var players []*g.Player
	var enemies []*g.Enemy

	WINDOW_SIZE = 800.0
	cellWidth := WINDOW_SIZE / float32(width)
	CELL_SIZE = fyne.NewSize(cellWidth, cellWidth)
	initColors()

	for ind, coor := range playersCoordinates {
		players = append(players, g.NewPlayer(coor.X, coor.Y, PLAYERS_COLORS[ind]))
	}

	for _, coor := range enemiesCoordinates {
		enemies = append(enemies, g.NewEnemy(coor.X, coor.Y, RED))
	}

	return &Simulation{
		width:      width,
		height:     height,
		collisions: collisions,
		players:    players,
		enemies:    enemies,
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

	s.paths = g.GetPaths(s.width, s.height, s.collisions, s.players)
	s.createGrid()
	s.createPathsForEnemies()

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

func (s *Simulation) addEnemies(objects *[][]fyne.CanvasObject) {
	for enemy := range s.enemies {
		coordinate := s.enemies[enemy].GetPosition()
		rgba := s.enemies[enemy].GetColor()
		(*objects)[coordinate.Y][coordinate.X] = s.createRectangle("E", color.White, rgba)
	}
}

func (s *Simulation) createGrid() {
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

	s.addCollisions(&objects)
	s.addPlayers(&objects)
	s.addEnemies(&objects)
	var flatObjects []fyne.CanvasObject
	for _, row := range objects {
		flatObjects = append(flatObjects, row...)
	}

	grid := container.NewGridWithColumns(s.width, flatObjects...)
	s.grid = grid
}

func (s *Simulation) createPathsForEnemies() {
	for enemy := range s.enemies {
		s.findPlayer(s.enemies[enemy])
		fmt.Println(s.enemies[enemy].GetMoves())
	}
	s.updateGrid()
}

func (s *Simulation) updateGrid() {
	for _, enemy := range s.enemies {
		for move := 0; move < len(enemy.GetMoves())-1; move++ {
			coordinate := s.parseMove(enemy.GetMoves()[move])
			currPosition := enemy.GetPosition()
			newX := currPosition.X + coordinate.X
			newY := currPosition.Y + coordinate.Y
			enemy.SetPosition(newX, newY)
			text := strconv.Itoa(s.paths[newY][newX].GetCellValue())
			ind := newY*s.width + newX
			s.grid.Objects[ind] = s.createRectangle(text, color.White, RED)
		}
	}
}

func (s *Simulation) findPlayer(enemy *g.Enemy) {
	start := enemy.GetPosition()
	path := []int{s.paths[start.Y][start.X].GetDirection()}
	moveCoordinate := start
	move := g.IDLE
	finish := false
	for {
		move, moveCoordinate = s.nextMove(moveCoordinate)

		for _, player := range s.players {
			pos := player.GetPosition()
			if pos.X == moveCoordinate.X && pos.Y == moveCoordinate.Y {
				finish = true
				break
			}
		}

		if finish {
			break
		}

		path = append(path, move)
	}

	enemy.SetMoves(path)
}

func (s *Simulation) parseMove(move int) g.Coordinate {
	switch move {
	case g.UP:
		return g.Coordinate{X: 0, Y: -1}
	case g.DOWN:
		return g.Coordinate{X: 0, Y: 1}
	case g.LEFT:
		return g.Coordinate{X: -1, Y: 0}
	case g.RIGHT:
		return g.Coordinate{X: 1, Y: 0}
	case g.IDLE:
		return g.Coordinate{X: 0, Y: 0}
	default:
		panic("unhandled default case")
	}
}

func (s *Simulation) nextMove(curr g.Coordinate) (int, g.Coordinate) {
	coordinates := []g.Coordinate{
		{X: max(0, curr.X-1), Y: curr.Y},
		{X: curr.X, Y: max(0, curr.Y-1)},
		{X: curr.X, Y: min(s.height-1, curr.Y+1)},
		{X: min(s.width-1, curr.X+1), Y: curr.Y},
	}

	bestCoordinate := g.Coordinate{}
	move := g.IDLE
	for _, coor := range coordinates {
		if coor.X != curr.X || coor.Y != curr.Y {
			coorValue := s.paths[coor.Y][coor.X].GetCellValue()
			bestCoordinateValue := s.paths[bestCoordinate.Y][bestCoordinate.X].GetCellValue()
			if bestCoordinateValue < coorValue {
				bestCoordinate = coor
				move = s.paths[coor.Y][coor.X].GetDirection()
			}
		}
	}

	return move, bestCoordinate
}

func main() {
	//collisions := []g.Coordinate{{10, 12}, {10, 13}, {12, 13}}
	//players := []g.Coordinate{{18, 2}, {15, 15}, {7, 11}}
	//enemies := []g.Coordinate{{5, 2}, {15, 1}}
	//collisions := []g.Coordinate{{1, 2}, {0, 1}, {2, 2}, {3, 2}}
	collisions := []g.Coordinate{{4, 0}}
	players := []g.Coordinate{{1, 4}, {4, 4}}
	enemies := []g.Coordinate{{0, 0}, {3, 0}}
	sm := NewSimulation(5, 5, collisions, players, enemies)
	sm.startSimulation()
}

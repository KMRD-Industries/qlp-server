package game_controllers

import (
	"fmt"
	"github.com/ungerik/go3d/vec2"
	"math"
	"server/errors"
)

var (
	UP        int
	DOWN      int
	LEFT      int
	RIGHT     int
	IDLE      int
	COLLISION int
	MIN       int
)

type AIAlgorithm struct {
	width, height, offsetWidth, offsetHeight       int
	collisions                                     []Coordinate // pierwsza tablica jest dla współrzędnych, każda tablica reprezentuje jeden blok kolizyjny
	players                                        map[uint32]Coordinate
	enemies                                        map[uint32]*Enemy
	graph                                          *[][]Cell
	minBorderX, minBorderY, maxBorderX, maxBorderY int
}

type Cell struct {
	direction *vec2.T
	value     int
}

type Coordinate struct {
	X, Y, Height, Width float32
}

func (c *Cell) GetDirection() vec2.T {
	return *(c.direction)
}

func (c *Cell) GetCellValue() int {
	return c.value
}

func (a *AIAlgorithm) GetEnemiesUpdate() {
	a.createDistancesMap()
}

func NewAIAlgorithm() *AIAlgorithm {
	return &AIAlgorithm{}
}

func (a *AIAlgorithm) initDirections() {
	MIN = 0
	UP = a.width + a.height + 1
	DOWN = UP + 1
	LEFT = DOWN + 1
	RIGHT = LEFT + 1
	IDLE = RIGHT + 1
	COLLISION = IDLE + 1
}

// TODO 1. zrób czyszczenie mapy w takich samych granicach jak wypełnianie vector field
//TODO 2. sprawdź czy czasem nie wypierdoli outOfRange jak dasz spawnery na samych krańcach mapy
// na symulacji się nie da, przydałoby się ją zaaktualizować

func (a *AIAlgorithm) createDistancesMap() {
	a.initDirections()

	a.addPlayers()
	a.addCollisions()
	a.findBorders()
	// generating heat map
	err := a.bfs()
	if err != nil {
		fmt.Println(err)
	}

	// generating vector field
	a.fillDirections()

	// debbuging
	//for _, row := range *(a.graph) {
	//	for _, el := range row {
	//		if el.direction != nil {
	//			fmt.Printf("%9f, %9f, %2d||", el.direction.Get(1, 0), el.direction.Get(0, 1), el.value)
	//		} else {
	//			fmt.Printf("%9f, %9f, %2d||", 0.0, 0.0, 0)
	//		}
	//	}
	//	fmt.Print("\n")
	//}
}

func (a *AIAlgorithm) InitGraph() {
	graph := make([][]Cell, a.height)
	for i := range graph {
		graph[i] = make([]Cell, a.width)
	}

	a.graph = &graph
}

func (a *AIAlgorithm) ClearGraph() {
	for i := range *a.graph {
		row := (*a.graph)[i]
		for j := range row {
			row[j].direction = nil
			row[j].value = 0
		}
	}
	//for i := a.minBorderY - 1; i <= a.maxBorderY; i++ {
	//	for j := a.minBorderX - 1; j <= a.maxBorderX; j++ {
	//		(*a.graph)[i][j].direction = nil
	//		(*a.graph)[i][j].value = 0
	//	}
	//}
	//fmt.Println("Cleared graph")

	//for _, row := range *(a.graph) {
	//	for _, el := range row {
	//		if el.direction != nil {
	//			fmt.Printf("%9f, %9f, %2d||", el.direction.Get(1, 0), el.direction.Get(0, 1), el.value)
	//		} else {
	//			fmt.Printf("%9f, %9f, %2d||", 0.0, 0.0, 0)
	//		}
	//	}
	//	fmt.Print("\n")
	//}
}

func (a *AIAlgorithm) addPlayers() {
	for _, player := range a.players {
		(*a.graph)[int(player.Y)-a.offsetHeight][int(player.X)-a.offsetWidth] = Cell{&vec2.T{0, 0}, MIN}
	}
}

func (a *AIAlgorithm) addCollisions() {
	for _, coll := range a.collisions {
		(*a.graph)[int(coll.Y)-a.offsetHeight][int(coll.X)-a.offsetHeight] = Cell{&vec2.T{0, 0}, COLLISION}
	}
}

func (a *AIAlgorithm) findBorders() {
	minBorderX := math.MaxInt
	maxBorderX := 0
	minBorderY := math.MaxInt
	maxBorderY := 0

	for _, enemy := range a.enemies {
		position := enemy.GetPosition()
		minBorderX = min(minBorderX, int(position.X))
		minBorderY = min(minBorderY, int(position.Y))
		maxBorderX = max(maxBorderX, int(position.X))
		maxBorderY = max(maxBorderY, int(position.Y))
	}

	for _, player := range a.players {
		minBorderX = min(minBorderX, int(player.X))
		minBorderY = min(minBorderY, int(player.Y))
		maxBorderX = max(maxBorderX, int(player.X))
		maxBorderY = max(maxBorderY, int(player.Y))
	}

	a.maxBorderX = maxBorderX - a.offsetWidth
	a.maxBorderY = maxBorderY - a.offsetHeight
	a.minBorderX = minBorderX - a.offsetWidth
	a.minBorderY = minBorderY - a.offsetHeight
}

// TODO ogranicz wyszukiwanie sąsiadów do najdalej oddalonego wroga i gracza
// maksymalna/minimalna wartość to właśnie pozycja takiego granicznego wroga/gracza
func (a *AIAlgorithm) bfs() error {
	queue := Queue{}
	for _, player := range a.players {
		playerWithOffset := Coordinate{
			X:      player.X - float32(a.offsetWidth),
			Y:      player.Y - float32(a.offsetHeight),
			Height: player.Height,
			Width:  player.Width}
		queue.put(playerWithOffset)
	}

	for {
		if queue.isEmpty() {
			break
		}

		current, ok := queue.get()
		if !ok {
			return errors.EmptyQueue
		}

		if int(current.X) >= a.minBorderX && int(current.X) <= a.maxBorderX && int(current.Y) >= a.minBorderY && int(current.Y) <= a.maxBorderY {
			neighbors := a.getNeighbors(current)
			for _, next := range neighbors {
				found := (*a.graph)[int(next.Y)][int(next.X)]
				if found.direction == nil {
					queue.put(next)
					distance := (*a.graph)[int(current.Y)][int(current.X)].value + 1
					if distance < (*a.graph)[int(next.Y)][int(next.X)].value {
						distance = (*a.graph)[int(next.Y)][int(next.X)].value
					}
					(*a.graph)[int(next.Y)][int(next.X)] = Cell{&vec2.T{0, 0}, distance}
				}
			}
		}
	}
	return nil
}

func (a *AIAlgorithm) fillDirections() {
	for i := a.minBorderY; i < a.maxBorderY+1; i++ {
		for j := a.minBorderX; j < a.maxBorderX+1; j++ {
			value := (*a.graph)[i][j].value
			if value != MIN && value != COLLISION {
				(*a.graph)[i][j].direction = a.parseToMove(Coordinate{X: float32(j), Y: float32(i)})
			}
		}
	}

	for _, enemy := range a.enemies {
		position := enemy.GetPosition()
		vector := (*a.graph)[int(position.Y)-a.offsetHeight][int(position.X)-a.offsetWidth].direction
		//log.Printf("ENEMY %d POSITION from fillDirections: x %d, y %d, vector: x %f, y %f\n", enemy.GetId(), position.X, position.Y, vector.Get(1, 0), vector.Get(0, 1))
		enemy.SetDirection(*vector)
	}
}

func (a *AIAlgorithm) parseToMove(position Coordinate) *vec2.T {
	neighbors := a.getNeighbors(position)
	x := (*a.graph)[int(neighbors[LEFT].Y)][int(neighbors[LEFT].X)].value - (*a.graph)[int(neighbors[RIGHT].Y)][int(neighbors[RIGHT].X)].value
	y := (*a.graph)[int(neighbors[DOWN].Y)][int(neighbors[DOWN].X)].value - (*a.graph)[int(neighbors[UP].Y)][int(neighbors[UP].X)].value

	move := vec2.T{float32(x), float32(y)}
	return move.Normalize()
}

// TODO coś tu nie gra - naprw, konkretnie na samych brzegach, policz czy to na pewno dobrze tworzy te wektory
func (a *AIAlgorithm) getNeighbors(vertex Coordinate) map[int]Coordinate {
	return map[int]Coordinate{
		UP:    {X: vertex.X, Y: max(0, vertex.Y-1)},
		LEFT:  {X: max(0, vertex.X-1), Y: vertex.Y},
		DOWN:  {X: vertex.X, Y: min(float32(a.height-1), vertex.Y+1)},
		RIGHT: {X: min(float32(a.width-1), vertex.X+1), Y: vertex.Y},
	}
}

func (a *AIAlgorithm) SetWidth(width int) {
	a.width = width
}

func (a *AIAlgorithm) SetHeight(height int) {
	a.height = height
}

func (a *AIAlgorithm) SetOffset(offsetWidth, offsetHeight int) {
	a.offsetWidth = offsetWidth
	a.offsetHeight = offsetHeight
}

func (a *AIAlgorithm) SetPlayers(players map[uint32]Coordinate) {
	a.players = players
}

func (a *AIAlgorithm) SetEnemies(enemies map[uint32]*Enemy) {
	a.enemies = enemies
}

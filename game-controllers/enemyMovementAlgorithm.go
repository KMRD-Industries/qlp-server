package game_controllers

import (
	"fmt"
	"github.com/ungerik/go3d/vec2"
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
	width      int
	height     int
	collisions []Coordinate // pierwsza tablica jest dla współrzędnych, każda tablica reprezentuje jeden blok kolizyjny
	players    []Coordinate
	enemies    []*Enemy
	graph      *[][]Cell
}

type Cell struct {
	direction *vec2.T
	value     int
}

type Coordinate struct {
	X, Y, Height, Width int
}

func (c *Cell) GetDirection() vec2.T {
	return *(c.direction)
}

func (c *Cell) GetCellValue() int {
	return c.value
}

func (a *AIAlgorithm) GetEnemiesUpdate(width, height int, collisions, players []Coordinate, enemies []*Enemy) {
	a.createDistancesMap(width, height, collisions, players, enemies)
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

func (a *AIAlgorithm) initAlgorithm(width, height int, collisions, players []Coordinate, enemies []*Enemy) {
	a.width = width
	a.height = height
	a.collisions = collisions
	a.players = players
	a.enemies = enemies
}

func (a *AIAlgorithm) createDistancesMap(width, height int, collisions, players []Coordinate, enemies []*Enemy) {
	a.initAlgorithm(width, height, collisions, players, enemies)
	a.initDirections()
	a.initGraph()
	err := a.bfs()
	if err != nil {
		fmt.Println(err)
	}

	a.fillDirections()
	//for _, row := range *(a.graph) {
	//	for _, el := range row {
	//		fmt.Printf("%10f, %10f, %2d ||", el.direction.Get(1, 0), el.direction.Get(0, 1), el.value)
	//	}
	//	fmt.Print("\n")
	//}
}

func (a *AIAlgorithm) initGraph() {
	graph := make([][]Cell, a.height)
	for i := range graph {
		graph[i] = make([]Cell, a.width)
	}

	a.graph = &graph
	a.addPlayers()
	a.addCollisions()
}

func (a *AIAlgorithm) addPlayers() {
	for _, player := range a.players {
		(*a.graph)[player.Y][player.X] = Cell{&vec2.T{0, 0}, MIN}
	}
}

func (a *AIAlgorithm) addCollisions() {
	for _, coll := range a.collisions {
		(*a.graph)[coll.Y][coll.X] = Cell{&vec2.T{0, 0}, COLLISION}
	}
}

func (a *AIAlgorithm) bfs() error {
	queue := Queue{}
	for _, player := range a.players {
		queue.put(player)
	}

	for {
		if queue.isEmpty() {
			break
		}

		current, ok := queue.get()
		if !ok {
			return errors.EmptyQueue
		}

		neighbors := a.getNeighbors(current)
		for _, next := range neighbors {
			found := (*a.graph)[next.Y][next.X]
			if found.direction == nil {
				queue.put(next)
				distance := (*a.graph)[current.Y][current.X].value + 1
				if distance < (*a.graph)[next.Y][next.X].value {
					distance = (*a.graph)[next.Y][next.X].value
				}
				(*a.graph)[next.Y][next.X] = Cell{&vec2.T{0, 0}, distance}
			}
		}
	}
	return nil
}

func (a *AIAlgorithm) fillDirections() {
	for _, enemy := range a.enemies {
		position := enemy.GetPosition()
		parsedPosition := a.parseToMove(position)
		enemy.SetDirection(parsedPosition)
		(*a.graph)[position.X][position.Y].direction = &parsedPosition
	}

}

func (a *AIAlgorithm) parseToMove(position Coordinate) vec2.T {
	neighbors := a.getNeighbors(position)
	x := (*a.graph)[neighbors[LEFT].Y][neighbors[LEFT].X].value - (*a.graph)[neighbors[RIGHT].Y][neighbors[RIGHT].X].value
	y := (*a.graph)[neighbors[DOWN].Y][neighbors[DOWN].X].value - (*a.graph)[neighbors[UP].Y][neighbors[UP].X].value

	move := vec2.T{float32(x), float32(y)}
	return *move.Normalize()
}

// TODO coś tu nie gra - naprwa
func (a *AIAlgorithm) getNeighbors(vertex Coordinate) map[int]Coordinate {
	tmpResult := map[int]Coordinate{
		UP:    {X: vertex.X, Y: max(0, vertex.Y-1)},
		LEFT:  {X: max(0, vertex.X-1), Y: vertex.Y},
		DOWN:  {X: vertex.X, Y: min(a.height-1, vertex.Y+1)},
		RIGHT: {X: min(a.width-1, vertex.X+1), Y: vertex.Y},
	}
	return tmpResult
}

func (a *AIAlgorithm) printGrid(arr [][]int) {
	for _, row := range arr {
		fmt.Println(row)
	}
	fmt.Print("\n")
}

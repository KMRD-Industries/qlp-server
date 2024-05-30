package game_controllers

import (
	"fmt"
	"qlp_server/errors"
)

const (
	COLLISION = iota - 1
	UP        = iota
	DOWN      = iota
	LEFT      = iota
	RIGHT     = iota
	IDLE      = iota
	MAX       = 100
	MIN       = -999
)

type AIAlgorithm struct {
	width      int
	height     int
	collisions []Coordinate // pierwsza tablica jest dla współrzędnych, każda tablica reprezentuje jeden blok kolizyjny
	players    []*Player
	graph      *[][]int
}

type Cell struct {
	direction int
	value     int
}

func (c *Cell) GetDirection() int {
	return c.direction
}

func (c *Cell) GetCellValue() int {
	return c.value
}

type Coordinate struct {
	X, Y int
}

// TODO porpaw to bo wygląda jak gówno
func GetPaths(width, height int, collisions []Coordinate, players []*Player) [][]Cell {
	algorithm := AIAlgorithm{}
	return algorithm.createDistancesMap(width, height, collisions, players)
}

// TODO dodaj do każdej komórki id gracza
func (a *AIAlgorithm) createDistancesMap(width, height int, collisions []Coordinate, players []*Player) [][]Cell {
	a.width = width
	a.height = height
	a.collisions = collisions
	a.players = players

	a.initGraph()
	paths, err := a.bfs()
	if err != nil {
		fmt.Println(err)
	}

	for _, row := range paths {
		fmt.Println(row)
	}

	return paths
}

func (a *AIAlgorithm) initGraph() {
	graph := make([][]int, a.height)
	for i := range graph {
		graph[i] = make([]int, a.width)
	}

	a.graph = &graph
	a.addPlayers()
	a.addCollisions()

	//for _, row := range *a.graph {
	//	fmt.Println(row)
	//}
}

func (a *AIAlgorithm) addPlayers() {
	for _, player := range a.players {
		position := player.GetPosition()
		(*a.graph)[position.Y][position.X] = MAX
	}
}

func (a *AIAlgorithm) addCollisions() {
	for _, coll := range a.collisions {
		(*a.graph)[coll.Y][coll.X] = MIN
	}
}

func (a *AIAlgorithm) bfs() ([][]Cell, error) {
	queue := Queue{}
	parent := make([][]Cell, a.height)
	for i := range parent {
		parent[i] = make([]Cell, a.width)
	}

	for _, p := range a.players {
		player := p.GetPosition()
		queue.put(player)
		parent[player.Y][player.X] = Cell{IDLE, MAX}
	}

	for {
		if queue.isEmpty() {
			break
		}

		current, ok := queue.get()
		if !ok {
			return nil, errors.EmptyQueue
		}

		for _, next := range a.getNeighbors(current) {
			found := parent[next.Y][next.X]
			val := (*a.graph)[next.Y][next.X]
			if found.direction == 0 && val != IDLE && val != COLLISION {
				queue.put(next)
				distance := parent[current.Y][current.X].value - 1
				if distance < parent[next.Y][next.X].value {
					distance = parent[next.Y][next.X].value
				}
				parent[next.Y][next.X] = Cell{a.parseToMove(current, next), distance}
			}
		}
	}

	return parent, nil
}

func (a *AIAlgorithm) parseToMove(current, next Coordinate) int {
	move := IDLE
	if current.X-next.X == 0 {
		if current.Y > next.Y {
			move = DOWN
		} else {
			move = UP
		}
	} else {
		if current.X > next.X {
			move = RIGHT
		} else {
			move = LEFT
		}
	}
	return move
}

func (a *AIAlgorithm) getNeighbors(vertex Coordinate) []Coordinate {
	tmpResult := []Coordinate{
		{X: max(0, vertex.X-1), Y: vertex.Y},
		{X: vertex.X, Y: max(0, vertex.Y-1)},
		{X: min(a.width-1, vertex.X+1), Y: vertex.Y},
		{X: vertex.X, Y: min(a.height-1, vertex.Y+1)},
	}

	var result []Coordinate
	for ver := range tmpResult {
		x := tmpResult[ver].X
		y := tmpResult[ver].Y
		if (x != vertex.X || y != vertex.Y) && (*a.graph)[y][x] != MIN {
			result = append(result, tmpResult[ver])
		}
	}

	return result
}

func (a *AIAlgorithm) printGrid(arr [][]int) {
	for _, row := range arr {
		fmt.Println(row)
	}
	fmt.Println("\n")
}

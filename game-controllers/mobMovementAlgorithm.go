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
	for _, val := range a.collisions {
		graph[val.X][val.Y] = -1
	}
	a.graph = &graph
	a.addPlayers()
	a.addCollisions()
}

func (a *AIAlgorithm) addPlayers() {
	for _, player := range a.players {
		(*a.graph)[player.position.X][player.position.Y] = MAX
	}
}

func (a *AIAlgorithm) addCollisions() {
	for _, coll := range a.collisions {
		(*a.graph)[coll.X][coll.Y] = MIN
	}
}

func (a *AIAlgorithm) bfs() ([][]Cell, error) {
	queue := Queue{}
	parent := make([][]Cell, a.height)
	for i := range parent {
		parent[i] = make([]Cell, a.width)
	}

	for _, p := range a.players {
		queue.put(p.position)
		parent[p.position.X][p.position.Y] = Cell{IDLE, MAX}
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
			found := parent[next.X][next.Y]
			val := (*a.graph)[next.X][next.Y]
			if found.direction == 0 && val != IDLE && val != COLLISION {
				queue.put(next)
				distance := parent[current.X][current.Y].value - 1
				if distance < parent[next.X][next.Y].value {
					distance = parent[next.X][next.Y].value
				}
				parent[next.X][next.Y] = Cell{a.parseToMove(current, next), distance}
			}
		}
	}

	return parent, nil
}

func (a *AIAlgorithm) parseToMove(current, next Coordinate) int {
	move := IDLE
	if current.X-next.X == 0 {
		if current.Y > next.Y {
			move = RIGHT
		} else {
			move = LEFT
		}
	} else {
		if current.X > next.X {
			move = DOWN
		} else {
			move = UP
		}
	}
	return move
}

func (a *AIAlgorithm) getNeighbors(vertex Coordinate) []Coordinate {
	tmpResult := []Coordinate{
		{X: max(0, vertex.X-1), Y: vertex.Y},
		{X: vertex.X, Y: max(0, vertex.Y-1)},
		{X: min(a.height-1, vertex.X+1), Y: vertex.Y},
		{X: vertex.X, Y: min(a.width-1, vertex.Y+1)},
	}

	var result []Coordinate
	for ver := range tmpResult {
		if tmpResult[ver].X != vertex.X && tmpResult[ver].Y != vertex.Y {
			result = append(result, tmpResult[ver])
		}
	}

	for ind, val := range result {
		if (*a.graph)[val.X][val.Y] == -1 {
			result = append(result[:ind], result[ind:]...)
		}
	}

	return tmpResult
}

func (a *AIAlgorithm) printGrid(arr [][]int) {
	for _, row := range arr {
		fmt.Println(row)
	}
	fmt.Println("\n")
}

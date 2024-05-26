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
)

type cell struct {
	direction int
	value     int
}

func NewAI(width, height int, collisions []Coordinate, player Coordinate) *AIAlgorithm {
	graph := make([][]int, height)
	for i := range graph {
		graph[i] = make([]int, width)
	}
	for _, val := range collisions {
		graph[val.X][val.Y] = -1
	}
	graph[player.X][player.Y] = MAX
	return &AIAlgorithm{width: width, height: height, collisions: collisions, player: player, graph: &graph}
}

func (a *AIAlgorithm) CreateDistancesMap() {
	paths, err := a.bfs()
	if err != nil {
		fmt.Println(err)
	}

	for _, row := range paths {
		fmt.Println(row)
	}
}

func (a *AIAlgorithm) bfs() ([][]cell, error) {
	queue := Queue{}
	queue.put(a.player)
	parent := make([][]cell, a.height)
	for i := range parent {
		parent[i] = make([]cell, a.width)
	}
	parent[a.player.X][a.player.Y] = cell{IDLE, MAX}
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
				parent[next.X][next.Y] = cell{a.parseToMove(current, next), parent[current.X][current.Y].value - 1}
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

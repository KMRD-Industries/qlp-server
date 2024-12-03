package game_controllers

import (
	"errors"
	"fmt"
	"github.com/ungerik/go3d/vec2"
	"log"
	"math"
)

var (
	UP         int
	DOWN       int
	LEFT       int
	RIGHT      int
	UP_LEFT    int
	UP_RIGHT   int
	DOWN_LEFT  int
	DOWN_RIGHT int
	IDLE       int
	COLLISION  int
	MIN        int
)

type AIAlgorithm struct {
	width, height, offsetWidth, offsetHeight       int
	collisions                                     []Coordinate
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
	UP_LEFT = RIGHT + 1
	UP_RIGHT = UP_LEFT + 1
	DOWN_LEFT = UP_RIGHT + 1
	DOWN_RIGHT = DOWN_LEFT + 1
	IDLE = DOWN_RIGHT + 1
	COLLISION = IDLE + 1
}

// TODO 1. zrób czyszczenie mapy w takich samych granicach jak wypełnianie vector field
// TODO 2. potwory się buggują i nie widzą drugiego playera i idą tylko do jednego
// TODO 3. źle się kolizję wypełniają i potwory wchodzą mi w ścianę

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
	log.Printf("Created graph, width: %d, height: %d\n", a.width, a.height)
}

func (a *AIAlgorithm) ClearGraph() {
	for i := range *a.graph {
		row := (*a.graph)[i]
		for j := range row {
			row[j].direction = nil
			row[j].value = 0
		}
	}
}

func (a *AIAlgorithm) addPlayers() {
	for _, player := range a.players {
		x := int(player.X) - a.offsetWidth
		y := int(player.Y) - a.offsetHeight
		if x < a.width && x >= 0 && y < a.height && y >= 0 {
			(*a.graph)[y][x] = Cell{&vec2.T{0, 0}, MIN}
		}
	}
}

func (a *AIAlgorithm) addCollisions() {
	for _, coll := range a.collisions {
		x := int(coll.X) - a.offsetWidth
		y := int(coll.Y) - a.offsetHeight
		if x < a.width && x >= 0 && y < a.height && y >= 0 {
			(*a.graph)[y][x] = Cell{&vec2.T{0, 0}, COLLISION}
		}
	}
}

func (a *AIAlgorithm) findBorders() {
	minBorderX := math.MaxInt
	maxBorderX := 0
	minBorderY := math.MaxInt
	maxBorderY := 0

	for _, enemy := range a.enemies {
		position := enemy.GetPosition()
		//log.Printf("Enemy's position %f, %f\noffset: %d, %d\nmap dimensions: %d, %d\n", position.X, position.Y, a.offsetWidth, a.offsetHeight, a.width, a.height)
		minBorderX = min(minBorderX, int(position.X)-a.offsetWidth)
		minBorderY = min(minBorderY, int(position.Y)-a.offsetHeight)
		maxBorderX = max(maxBorderX, int(position.X)-a.offsetWidth)
		maxBorderY = max(maxBorderY, int(position.Y)-a.offsetHeight)
	}
	//log.Printf("Boarders after enemies update:\nminBorderX: %d\nminBorderY: %d\nmaxBorderX: %d\nmaxBorderY: %d\n", minBorderX, minBorderY, maxBorderX, maxBorderY)

	for _, player := range a.players {
		//log.Printf("Player's position %f, %f\noffset: %d, %d\nmap dimensions: %d, %d\n", player.X, player.Y, a.offsetWidth, a.offsetHeight, a.width, a.height)
		minBorderX = min(minBorderX, int(player.X)-a.offsetWidth)
		minBorderY = min(minBorderY, int(player.Y)-a.offsetHeight)
		maxBorderX = max(maxBorderX, int(player.X)-a.offsetWidth)
		maxBorderY = max(maxBorderY, int(player.Y)-a.offsetHeight)
	}

	//log.Printf("Boarders after player update:\nminBorderX: %d\nminBorderY: %d\nmaxBorderX: %d\nmaxBorderY: %d\n", minBorderX, minBorderY, maxBorderX, maxBorderY)

	a.maxBorderX = min(a.width-1, maxBorderX)
	a.maxBorderY = min(a.height-1, maxBorderY)
	a.minBorderX = max(0, min(minBorderX, a.width-1))
	a.minBorderY = max(0, min(minBorderY, a.height-1))
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
			return errors.New("queue is empty")
		}

		if int(current.X) >= 0 && int(current.X) < a.width && int(current.Y) >= 0 && int(current.Y) < a.height {
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
		if vector != nil {
			enemy.SetDirection(*vector)
		}
	}
}

func (a *AIAlgorithm) parseToMove(position Coordinate) *vec2.T {
	neighbors := a.getNeighbors(position)
	//x := 0.0
	//y := 0.0

	// poziome
	//x += float64((*a.graph)[int(neighbors[LEFT].Y)][int(neighbors[LEFT].X)].value) -
	//	float64((*a.graph)[int(neighbors[RIGHT].Y)][int(neighbors[RIGHT].X)].value)
	//
	//// pionowe
	//y += float64((*a.graph)[int(neighbors[DOWN].Y)][int(neighbors[DOWN].X)].value) -
	//	float64((*a.graph)[int(neighbors[UP].Y)][int(neighbors[UP].X)].value)
	//
	//// ukośne
	//x += float64((*a.graph)[int(neighbors[UP_LEFT].Y)][int(neighbors[UP_LEFT].X)].value) -
	//	float64((*a.graph)[int(neighbors[DOWN_RIGHT].Y)][int(neighbors[DOWN_RIGHT].X)].value)
	//
	//y += float64((*a.graph)[int(neighbors[DOWN_RIGHT].Y)][int(neighbors[DOWN_RIGHT].X)].value) -
	//	float64((*a.graph)[int(neighbors[UP_LEFT].Y)][int(neighbors[UP_LEFT].X)].value)
	//
	//x += float64((*a.graph)[int(neighbors[UP_RIGHT].Y)][int(neighbors[UP_RIGHT].X)].value) -
	//	float64((*a.graph)[int(neighbors[DOWN_LEFT].Y)][int(neighbors[DOWN_LEFT].X)].value)
	//
	//y += float64((*a.graph)[int(neighbors[DOWN_LEFT].Y)][int(neighbors[DOWN_LEFT].X)].value) -
	//	float64((*a.graph)[int(neighbors[UP_RIGHT].Y)][int(neighbors[UP_RIGHT].X)].value)
	x := (*a.graph)[int(neighbors[LEFT].Y)][int(neighbors[LEFT].X)].value - (*a.graph)[int(neighbors[RIGHT].Y)][int(neighbors[RIGHT].X)].value
	y := (*a.graph)[int(neighbors[DOWN].Y)][int(neighbors[DOWN].X)].value - (*a.graph)[int(neighbors[UP].Y)][int(neighbors[UP].X)].value

	move := vec2.T{float32(x), float32(y)}
	return move.Normalize()
}

func (a *AIAlgorithm) getNeighbors(vertex Coordinate) map[int]Coordinate {
	return map[int]Coordinate{
		UP:    {X: vertex.X, Y: max(0, vertex.Y-1)},
		LEFT:  {X: max(0, vertex.X-1), Y: vertex.Y},
		DOWN:  {X: vertex.X, Y: min(float32(a.height-1), vertex.Y+1)},
		RIGHT: {X: min(float32(a.width-1), vertex.X+1), Y: vertex.Y},
	}
}

func (a *AIAlgorithm) getNeighborsExtended(vertex Coordinate) map[int]Coordinate {
	neighbors := make(map[int]Coordinate)

	neighbors[UP] = Coordinate{X: vertex.X, Y: max(0, vertex.Y-1)}
	neighbors[LEFT] = Coordinate{X: max(0, vertex.X-1), Y: vertex.Y}
	neighbors[DOWN] = Coordinate{X: vertex.X, Y: min(float32(a.height-1), vertex.Y+1)}
	neighbors[RIGHT] = Coordinate{X: min(float32(a.width-1), vertex.X+1), Y: vertex.Y}

	neighbors[UP_LEFT] = Coordinate{X: max(0, vertex.X-1), Y: max(0, vertex.Y-1)}
	neighbors[UP_RIGHT] = Coordinate{X: min(float32(a.width-1), vertex.X+1), Y: max(0, vertex.Y-1)}
	neighbors[DOWN_LEFT] = Coordinate{X: max(0, vertex.X-1), Y: min(float32(a.height-1), vertex.Y+1)}
	neighbors[DOWN_RIGHT] = Coordinate{X: min(float32(a.width-1), vertex.X+1), Y: min(float32(a.height-1), vertex.Y+1)}

	return neighbors
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

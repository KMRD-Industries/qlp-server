package game_controllers

import (
	"fmt"
	"github.com/ungerik/go3d/vec2"
	"log"
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
	X, Y, Height, Width int
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

// TODO 1. zrób czyszczenie pozycji graczy i potworów z grafu
// TODO 2, kiilkanaście razy ten sam potwór/gracz jest dodawany do mapy
// TODO 3. ogranicz rysowanie vector field (nie tworzenie grafu) do najdalszego potwora/gracza
// TODO 4. tablica kolizji ciągle rośnie - ona nie jest czyszczona
// TODO 5. sprawdź czy tablica graczy dodaje tylko jednego gracza per id - jak nie zrób mapę: klucz idGracza, wartość pozycja gracza
// TODO 6.

//func (a *AIAlgorithm) initAlgorithm(width, height, offsetWidth, offsetHeight int, collisions, players []Coordinate, enemies map[uint32]*Enemy) {
//	a.width = width
//	a.height = height
//	a.offsetWidth = offsetWidth
//	a.offsetHeight = offsetHeight
//	a.collisions = collisions
//	a.players = players
//	a.enemies = enemies
//}

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

	fmt.Printf("Value near player: %d\n", (*a.graph)[a.players[1].Y-a.offsetHeight+1][a.players[1].X-a.offsetWidth+1].value)

	// generating vector field
	a.fillDirections()
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
}

func (a *AIAlgorithm) addPlayers() {
	for _, player := range a.players {
		(*a.graph)[player.Y-a.offsetHeight][player.X-a.offsetWidth] = Cell{&vec2.T{0, 0}, MIN}
	}
}

func (a *AIAlgorithm) addCollisions() {
	for _, coll := range a.collisions {
		(*a.graph)[coll.Y-a.offsetHeight][coll.X-a.offsetHeight] = Cell{&vec2.T{0, 0}, COLLISION}
	}
}

func (a *AIAlgorithm) findBorders() {
	minBorderX := math.MaxInt
	maxBorderX := 0
	minBorderY := math.MaxInt
	maxBorderY := 0

	for _, enemy := range a.enemies {
		position := enemy.GetPosition()
		minBorderX = min(minBorderX, position.X)
		minBorderY = min(minBorderY, position.Y)
		maxBorderX = max(maxBorderX, position.X)
		maxBorderY = max(maxBorderY, position.Y)
	}

	for _, player := range a.players {
		minBorderX = min(minBorderX, player.X)
		minBorderY = min(minBorderY, player.Y)
		maxBorderX = max(maxBorderX, player.X)
		maxBorderY = max(maxBorderY, player.Y)
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
			X:      player.X - a.offsetWidth,
			Y:      player.Y - a.offsetHeight,
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

		//if current.X >= a.minBorderX && current.X <= a.maxBorderX && current.Y >= a.minBorderY && current.Y <= a.maxBorderY {
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
	//}
	return nil
}

func (a *AIAlgorithm) fillDirections() {

	for i := a.minBorderY; i < a.maxBorderY+1; i++ {
		for j := a.minBorderX; j < a.maxBorderX+1; j++ {
			value := (*a.graph)[i][j].value
			if value != MIN && value != COLLISION {
				(*a.graph)[i][j].direction = a.parseToMove(Coordinate{X: j, Y: i})
			}
		}
	}

	for _, enemy := range a.enemies {
		position := enemy.GetPosition()
		vector := (*a.graph)[position.Y-a.offsetHeight][position.X-a.offsetWidth].direction
		log.Printf("ENEMY %d POSITION from fillDirections: x %d, y %d, vector: x %f, y %f\n", enemy.GetId(), position.X, position.Y, vector.Get(1, 0), vector.Get(0, 1))
		enemy.SetDirection(*vector)
	}
}

func (a *AIAlgorithm) parseToMove(position Coordinate) *vec2.T {
	neighbors := a.getNeighbors(position)
	x := (*a.graph)[neighbors[LEFT].Y][neighbors[LEFT].X].value - (*a.graph)[neighbors[RIGHT].Y][neighbors[RIGHT].X].value
	y := (*a.graph)[neighbors[DOWN].Y][neighbors[DOWN].X].value - (*a.graph)[neighbors[UP].Y][neighbors[UP].X].value

	move := vec2.T{float32(x), float32(y)}
	return move.Normalize()
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

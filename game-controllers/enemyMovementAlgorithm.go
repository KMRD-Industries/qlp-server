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
	debug                                          bool
}

type Cell struct {
	direction *vec2.T
	value     int
}

type Coordinate struct {
	X, Y int
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
	//a.printGraphWithAxes2()
	a.findBorders()
	err := a.bfs()
	if err != nil {
		fmt.Println(err)
	}
	a.fillDirections()
	if a.debug {
		a.debug = false
		a.printGraphWithAxes()
	}
}

func (a *AIAlgorithm) printGraphWithAxes2() {
	fmt.Print("    ")
	for i := 0; i < len((*a.graph)[0]); i++ {
		fmt.Printf("%2d |", i)
	}
	fmt.Print("\n")

	for i, row := range *(a.graph) {
		fmt.Printf("%2d ", i)
		for _, el := range row {
			if el.value != COLLISION {
				fmt.Printf("%s|", getDirectionArrow(0, 0))
			} else {
				fmt.Printf(" C |")
				//fmt.Printf("%s||", getDirectionArrow(0.0, 0.0))
			}

		}
		fmt.Print("\n")
	}
}

func (a *AIAlgorithm) printGraphWithAxes() {
	fmt.Print("    ")
	for i := 0; i < len((*a.graph)[0]); i++ {
		fmt.Printf("%2d |", i)
	}
	fmt.Print("\n")

	enemeySpawned := false
	for i, row := range *(a.graph) {
		fmt.Printf("%2d  ", i)
		for j, el := range row {
			for _, enemy := range a.enemies {
				position := enemy.position
				if position.X-a.offsetWidth == j && position.Y-a.offsetHeight == i {
					fmt.Printf(" ● |")
					enemeySpawned = true
				}
			}
			if !enemeySpawned {
				if el.value != COLLISION && el.direction != nil {
					fmt.Printf("%s|", getDirectionArrow(float64(el.direction.Get(1, 0)), float64(el.direction.Get(0, 1))))
				} else if el.value == COLLISION {
					fmt.Printf(" C |")
					//fmt.Printf("%s||", getDirectionArrow(0.0, 0.0))
				} else {
					fmt.Printf("%s|", getDirectionArrow(0, 0))
				}
			}
			enemeySpawned = false
		}
		fmt.Print("\n")
	}
}

func getDirectionArrow(x, y float64) string {
	//if x == 0 && y == 0 {
	//	return " ● "
	//}
	if x == 0 && y == 0 {
		return "   "
	}

	angle := math.Atan2(y, x)
	directions := []string{" → ", " ↗ ", " ↑ ", " ↖ ", " ← ", " ↙ ", " ↓ ", " ↘ "}
	index := int(math.Round(angle/(math.Pi/4)+8)) % 8
	return directions[index]
}

func (a *AIAlgorithm) InitGraph() {
	graph := make([][]Cell, a.height)
	for i := range graph {
		graph[i] = make([]Cell, a.width)
	}
	a.graph = &graph
	log.Printf("Created graph, width: %d, height: %d\n", a.width, a.height)
	a.expandCollisions()
	a.addCollisions()
	a.printGraphWithAxes2()
	a.debug = true
}

func (a *AIAlgorithm) ClearGraph() {
	for i := range *a.graph {
		row := (*a.graph)[i]
		for j := range row {
			row[j].direction = nil
			row[j].value = 0
		}
	}
	//a.printGraphWithAxes2()
}

func (a *AIAlgorithm) addPlayers() {
	for _, player := range a.players {
		x := player.X - a.offsetWidth
		y := player.Y - a.offsetHeight
		if x < a.width && x >= 0 && y < a.height && y >= 0 {
			(*a.graph)[y][x] = Cell{&vec2.T{0, 0}, MIN}
		}
	}
}

func (a *AIAlgorithm) addCollisions() {
	//fmt.Printf("Collision length in add collisions: %d\n", len(a.collisions))
	for _, coll := range a.collisions {
		x := coll.X - a.offsetWidth
		y := coll.Y - a.offsetHeight
		if x < a.width && x >= 0 && y < a.height && y >= 0 {
			//fmt.Printf("adding collision: %d\n", cnt)
			(*a.graph)[y][x] = Cell{&vec2.T{0, 0}, COLLISION}
		}
	}
}

func (a *AIAlgorithm) expandCollisions() {
	padding := 2

	expandedCollisions := make([]Coordinate, 0)

	for _, coll := range a.collisions {
		x := coll.X
		y := coll.Y

		for dy := -padding; dy <= padding; dy++ {
			for dx := -padding; dx <= padding; dx++ {
				newX := x + dx
				newY := y + dy

				//if newX >= 0 && newX < a.width && newY >= 0 && newY < a.height {
				expandedCollisions = append(expandedCollisions, Coordinate{
					X: newX,
					Y: newY,
				})
				//}
			}
		}
	}

	fmt.Printf("ExpandedCollisions: %d\n", len(expandedCollisions))
	a.collisions = expandedCollisions
}

func (a *AIAlgorithm) findBorders() {
	minBorderX := math.MaxInt
	maxBorderX := 0
	minBorderY := math.MaxInt
	maxBorderY := 0

	for _, enemy := range a.enemies {
		position := enemy.GetPosition()
		//log.Printf("Enemy's position %f, %f\noffset: %d, %d\nmap dimensions: %d, %d\n", position.X, position.Y, a.offsetWidth, a.offsetHeight, a.width, a.height)
		minBorderX = min(minBorderX, position.X-a.offsetWidth)
		minBorderY = min(minBorderY, position.Y-a.offsetHeight)
		maxBorderX = max(maxBorderX, position.X-a.offsetWidth)
		maxBorderY = max(maxBorderY, position.Y-a.offsetHeight)
	}
	//log.Printf("Boarders after enemies update:\nminBorderX: %d\nminBorderY: %d\nmaxBorderX: %d\nmaxBorderY: %d\n", minBorderX, minBorderY, maxBorderX, maxBorderY)

	for _, player := range a.players {
		//log.Printf("Player's position %f, %f\noffset: %d, %d\nmap dimensions: %d, %d\n", player.X, player.Y, a.offsetWidth, a.offsetHeight, a.width, a.height)
		minBorderX = min(minBorderX, player.X-a.offsetWidth)
		minBorderY = min(minBorderY, player.Y-a.offsetHeight)
		maxBorderX = max(maxBorderX, player.X-a.offsetWidth)
		maxBorderY = max(maxBorderY, player.Y-a.offsetHeight)
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
			X: player.X - a.offsetWidth,
			Y: player.Y - a.offsetHeight}
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

		if current.X >= 0 && current.X < a.width && current.Y >= 0 && current.Y < a.height {
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
	}
	return nil
}

func (a *AIAlgorithm) fillDirections() {
	//for i := a.minBorderY; i < a.maxBorderY+1; i++ {
	//	for j := a.minBorderX; j < a.maxBorderX+1; j++ {
	//		value := (*a.graph)[i][j].value
	//		if value != MIN && value != COLLISION {
	//			(*a.graph)[i][j].direction = a.parseToMove(Coordinate{X: j, Y: i})
	//		}
	//	}
	//}
	for i := 0; i < a.height; i++ {
		for j := 0; j < a.width; j++ {
			value := (*a.graph)[i][j].value
			if value != MIN && value != COLLISION {
				(*a.graph)[i][j].direction = a.parseToMove(Coordinate{X: j, Y: i})
			}
		}
	}

	for _, enemy := range a.enemies {
		position := enemy.GetPosition()
		//vector := (*a.graph)[position.Y-a.offsetHeight][position.X-a.offsetWidth].direction
		y := position.Y - a.offsetHeight
		x := position.X - a.offsetWidth
		vector := (*a.graph)[y][x].direction

		if vector == nil {
			continue
		}

		vecX := vector[0]
		vecY := vector[1]
		if vecX == -1 && vecY == 0 && (*a.graph)[y+3][x-1].value == COLLISION && y%16 > 0 {
			enemy.SetDirection(vec2.T{0, -1})
			//a.printGraphAroundEnemy(position.X-a.offsetWidth, position.Y-a.offsetHeight)
		} else if vecX == 0 && vecY == -1 && (*a.graph)[y-1][x+3].value == COLLISION && x%16 > 0 {
			enemy.SetDirection(vec2.T{-1, 0})
		} else {
			enemy.SetDirection(*vector)
		}

		//if vecX > 0 {
		//	// prawo góra, prawo, sprawdzamy na prawo dół
		//	if vecY >= 0 && (*a.graph)[x+3][y+2].value == COLLISION && y%16 > 0 {
		//		enemy.SetDirection(vec2.T{0, -1})
		//		// prawo dół, sprawdzamy na prawo góre
		//	} else if vecY < 0 && (*a.graph)[x][y+3].value == COLLISION {
		//		enemy.SetDirection(vec2.T{0, 1})
		//	}
		//	//else if vecY == 0 && (*a.graph)[x+3][y+2].value == COLLISION {
		//	//	enemy.SetDirection(vec2.T{1, 0})
		//	//}
		//} else if vecX < 0 {
		//	// lew góra, lewo, sprawdzamy lewo dół
		//	if vecY <= 0 && (*a.graph)[x-1][y+3].value == COLLISION && y%16 > 0 {
		//		enemy.SetDirection(vec2.T{0, 1})
		//		// lew dół, sprawdzamy dół prawo
		//	} else if vecY > 0 && (*a.graph)[x+2][y+3].value == COLLISION && x%16 > 0 {
		//		enemy.SetDirection(vec2.T{-1, 0})
		//	}
		//} else {
		//	// dół, sprawdzamy dół prawo
		//	if vecY >= 0 && (*a.graph)[x+2][y+3].value == COLLISION && x%16 > 0 {
		//		enemy.SetDirection(vec2.T{-1, 0})
		//		// góra, sprawdzamy góra prawo
		//	} else if vecY < 0 && (*a.graph)[x+3][y-1].value == COLLISION && x%16 > 0 {
		//		enemy.SetDirection(vec2.T{-1, 0})
		//	}
		//}
	}
}

func (a *AIAlgorithm) parseToMove(vertex Coordinate) *vec2.T {
	neighbors := a.getNeighbors(vertex)
	x := 0
	y := 0

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
	if (*a.graph)[neighbors[UP].Y][neighbors[UP].X].value == COLLISION && (*a.graph)[neighbors[DOWN].Y][neighbors[DOWN].X].value == COLLISION {
		y = 0
	} else if (*a.graph)[neighbors[UP].Y][neighbors[UP].X].value == COLLISION {
		y = (*a.graph)[neighbors[DOWN].Y][neighbors[DOWN].X].value - (*a.graph)[vertex.Y][vertex.X].value - 1
	} else if (*a.graph)[neighbors[DOWN].Y][neighbors[DOWN].X].value == COLLISION {
		y = (*a.graph)[vertex.Y][vertex.X].value - (*a.graph)[neighbors[UP].Y][neighbors[UP].X].value + 1
	} else {
		y = (*a.graph)[neighbors[DOWN].Y][neighbors[DOWN].X].value - (*a.graph)[neighbors[UP].Y][neighbors[UP].X].value
	}

	if (*a.graph)[neighbors[LEFT].Y][neighbors[LEFT].X].value == COLLISION && (*a.graph)[neighbors[RIGHT].Y][neighbors[RIGHT].X].value == COLLISION {
		x = 0
	} else if (*a.graph)[neighbors[LEFT].Y][neighbors[LEFT].X].value == COLLISION {
		x = (*a.graph)[vertex.Y][vertex.X].value - (*a.graph)[neighbors[RIGHT].Y][neighbors[RIGHT].X].value + 1
	} else if (*a.graph)[neighbors[RIGHT].Y][neighbors[RIGHT].X].value == COLLISION {
		x = (*a.graph)[neighbors[LEFT].Y][neighbors[LEFT].X].value - (*a.graph)[vertex.Y][vertex.X].value - 1
	} else {
		x = (*a.graph)[neighbors[LEFT].Y][neighbors[LEFT].X].value - (*a.graph)[neighbors[RIGHT].Y][neighbors[RIGHT].X].value
	}

	move := vec2.T{float32(x), float32(y)}
	return move.Normalize()
}

func (a *AIAlgorithm) getNeighbors(vertex Coordinate) map[int]Coordinate {
	neighbors := map[int]Coordinate{
		UP:    {X: vertex.X, Y: max(0, vertex.Y-1)},
		LEFT:  {X: max(0, vertex.X-1), Y: vertex.Y},
		DOWN:  {X: vertex.X, Y: min(a.height-1, vertex.Y+1)},
		RIGHT: {X: min(a.width-1, vertex.X+1), Y: vertex.Y},
	}
	return neighbors
}

func (a *AIAlgorithm) getNeighborsExtended(vertex Coordinate) map[int]Coordinate {
	neighbors := make(map[int]Coordinate)

	neighbors[UP] = Coordinate{X: vertex.X, Y: max(0, vertex.Y-1)}
	neighbors[LEFT] = Coordinate{X: max(0, vertex.X-1), Y: vertex.Y}
	neighbors[DOWN] = Coordinate{X: vertex.X, Y: min(a.height-1, vertex.Y+1)}
	neighbors[RIGHT] = Coordinate{X: min(a.width-1, vertex.X+1), Y: vertex.Y}

	neighbors[UP_LEFT] = Coordinate{X: max(0, vertex.X-1), Y: max(0, vertex.Y-1)}
	neighbors[UP_RIGHT] = Coordinate{X: min(a.width-1, vertex.X+1), Y: max(0, vertex.Y-1)}
	neighbors[DOWN_LEFT] = Coordinate{X: max(0, vertex.X-1), Y: min(a.height-1, vertex.Y+1)}
	neighbors[DOWN_RIGHT] = Coordinate{X: min(a.width-1, vertex.X+1), Y: min(a.height-1, vertex.Y+1)}

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

func (a *AIAlgorithm) SetCollision(collisions []Coordinate) {
	a.collisions = collisions
}

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
	width, height, offsetWidth, offsetHeight int
	collisions, players                      []Coordinate // pierwsza tablica jest dla współrzędnych, każda tablica reprezentuje jeden blok kolizyjny
	enemies                                  map[uint32]*Enemy
	graph                                    *[][]Cell
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

func (a *AIAlgorithm) GetEnemiesUpdate(width, height, offsetWidth, offsetHeight int, collisions, players []Coordinate, enemies map[uint32]*Enemy) {
	a.createDistancesMap(width, height, offsetWidth, offsetHeight, collisions, players, enemies)
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

// TODO 1. niech inicjalizuje graf tylko raz na pokój - ściany są rysowane raz na pokój, potem zmienia się tylko ilość potworów, pozycja potworów i pozycja gracza
// TODO 2. niech graf ma granice tam, gdzie najdalszy potwór/gracz
// TODO 3. nieaktualizuje się pozycja gracza
// TODO 4. niech tworzy się tylko jedna instancja algorytmu per serwer
// TODO 5. tablica kolizji ciągle rośnie - ona nie jest czyszczona
// TODO 6. sprawdź czy tablica graczy dodaje tylko jednego gracza per id - jak nie zrób mapę: klucz idGracza, wartość pozycja gracza
// !!! TODO 7. x z y nie jest pomylony, potwory ciągle idą w lewy górny/dolny, nie idą nigdy w prawy górny/dolny
// TODO 8. idą w jedna stronę, bo zawsze wysyłam tylko pierwszy ruch - może powinienem przesyłąć całą sekwencję ruchów
// TODO 8.cd chodzi o to, że fixują się na pierwszym ruchu, bo nie przesyłam dalszej im ścieżki - zmiana ścieżki powinna następować po zmianie pozycji gracza (chyba???)
func (a *AIAlgorithm) initAlgorithm(width, height, offsetWidth, offsetHeight int, collisions, players []Coordinate, enemies map[uint32]*Enemy) {
	a.width = width
	a.height = height
	a.offsetWidth = offsetWidth
	a.offsetHeight = offsetHeight
	a.collisions = collisions
	a.players = players
	a.enemies = enemies
}

func (a *AIAlgorithm) createDistancesMap(width, height, offsetWidth, offsetHeight int, collisions, players []Coordinate, enemies map[uint32]*Enemy) {
	a.initAlgorithm(width, height, offsetWidth, offsetHeight, collisions, players, enemies)
	a.initDirections()
	a.initGraph()
	err := a.bfs()
	if err != nil {
		fmt.Println(err)
	}

	a.fillDirections()
	//for _, row := range *(a.graph) {
	//	for _, el := range row {
	//		fmt.Printf(" %2d |", el.value)
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
		(*a.graph)[player.Y-a.offsetHeight][player.X-a.offsetWidth] = Cell{&vec2.T{0, 0}, MIN}
	}
}

func (a *AIAlgorithm) addCollisions() {
	for _, coll := range a.collisions {
		(*a.graph)[coll.Y-a.offsetHeight][coll.X-a.offsetHeight] = Cell{&vec2.T{0, 0}, COLLISION}
	}
}

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
	//for _, enemy := range a.enemies {
	//	position := enemy.GetPosition()
	//	parsedPosition := a.parseToMove(position)
	//	enemy.SetDirection(parsedPosition)
	//	(*a.graph)[position.Y-a.offsetHeight][position.X-a.offsetWidth].direction = &parsedPosition
	//}
	for i := 0; i < a.height; i++ {
		for j := 0; j < a.width; j++ {
			value := (*a.graph)[i][j].value
			if value != MIN && value != COLLISION {
				(*a.graph)[i][j].direction = a.parseToMove(Coordinate{X: j, Y: i})
			}
		}
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

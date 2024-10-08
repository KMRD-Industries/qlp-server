package main

import (
	"fmt"
	"math"
	g "server/game-controllers"
)

type Simulation struct {
	width, height, offsetWidth, offsetHeight int
	collisions                               []g.Coordinate
	players                                  map[uint32]g.Coordinate
	enemies                                  map[uint32]*g.Enemy
	graph                                    *[][]g.Cell
	algorithm                                g.AIAlgorithm
}

func NewSimulation(width, height, offsetWidth, offsetHeight int, collisions []g.Coordinate, players map[uint32]g.Coordinate, enemies map[uint32]*g.Enemy) *Simulation {
	return &Simulation{
		width:        width,
		height:       height,
		offsetWidth:  offsetWidth,
		offsetHeight: offsetHeight,
		collisions:   collisions,
		players:      players,
		enemies:      enemies,
	}
}

func (s *Simulation) startSimulation() {
	s.algorithm.SetWidth(s.width)
	s.algorithm.SetHeight(s.height)
	s.algorithm.SetOffset(s.offsetWidth, s.offsetHeight)
	s.algorithm.InitGraph()

	s.algorithm.SetEnemies(s.enemies)
	s.algorithm.SetPlayers(s.players)

	s.algorithm.GetEnemiesUpdate()
	for _, enemy := range s.enemies {
		position := enemy.GetPosition()
		fmt.Printf("Enemy's new direction id: %d, position: xpos: %d, ypos: %d, x: %f, y: %f\n", enemy.GetId(), position.X, position.Y, enemy.GetDirectionX(), enemy.GetDirectionY())
	}
}

func initSimulation(collisions []g.Coordinate, players map[uint32]g.Coordinate, enemies map[uint32]*g.Enemy) *Simulation {
	var maxHeight int32 = 0
	var maxWidth int32 = 0
	var minHeight int32 = math.MaxInt32
	var minWidth int32 = math.MaxInt32
	fmt.Println("Obstacles: ")
	for _, collision := range collisions {
		//fmt.Printf("Obstacle: top %d, left: %d, height: %d, width: %d\n", collision.Top, collision.Left, collision.Height, collision.Width)
		maxHeight = max(maxHeight, int32(collision.Y))
		maxWidth = max(maxWidth, int32(collision.X))
		minHeight = min(minHeight, int32(collision.Y))
		minWidth = min(minWidth, int32(collision.X))
	}
	offsetWidth := minWidth
	offsetHeight := minHeight

	return NewSimulation(int(maxWidth-minWidth)+1, int(maxHeight-minHeight)+1, int(offsetWidth), int(offsetHeight), collisions, players, enemies)
}

func main() {
	collisions := []g.Coordinate{{7, 7, 0, 0}, {0, 0, 0, 0}}
	players := map[uint32]g.Coordinate{}
	players[1] = g.Coordinate{X: 2, Y: 4}
	enemies := map[uint32]*g.Enemy{}
	enemy1 := g.NewEnemy(1, 5, 1)
	enemy2 := g.NewEnemy(2, 7, 3)
	enemy3 := g.NewEnemy(3, 6, 6)
	enemy4 := g.NewEnemy(4, 4, 7)
	enemies[enemy1.GetId()] = enemy1
	enemies[enemy2.GetId()] = enemy2
	enemies[enemy3.GetId()] = enemy3
	enemies[enemy4.GetId()] = enemy4

	//players := []g.Coordinate{{6, 4, 0, 0}}
	//enemy1 := g.NewEnemy(1, 4, 1)
	//enemy2 := g.NewEnemy(2, 2, 3)
	//enemy3 := g.NewEnemy(3, 3, 5)
	//enemy4 := g.NewEnemy(4, 5, 7)
	//enemies[enemy1.GetId()] = enemy1
	//enemies[enemy2.GetId()] = enemy2
	//enemies[enemy3.GetId()] = enemy3
	//enemies[enemy4.GetId()] = enemy4
	//TODO działa dobrze na 1000x1000 - zrobić optymalizacjie żeby tworzyło mi graf na podstawie gdzie jest najbardziej
	// oddalony przeciwnik
	sm := initSimulation(collisions, players, enemies)
	sm.startSimulation()
}

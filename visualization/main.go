package main

import (
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
	s.algorithm.SetCollision(s.collisions)

	s.algorithm.GetEnemiesUpdate()
	//for _, enemy := range s.enemies {
	//	position := enemy.GetPosition()
	//	fmt.Printf("Enemy's new direction id: %d, position: xpos: %d, ypos: %d, x: %f, y: %f\n", enemy.GetId(), position.X, position.Y, enemy.GetDirectionX(), enemy.GetDirectionY())
	//}
}

func initSimulation(collisions []g.Coordinate, players map[uint32]g.Coordinate, enemies map[uint32]*g.Enemy) *Simulation {
	var maxHeight int32 = 0
	var maxWidth int32 = 0
	var minHeight int32 = math.MaxInt32
	var minWidth int32 = math.MaxInt32
	for _, collision := range collisions {
		//fmt.Printf("Obstacle: top %d, left: %d, height: %d, width: %d\n", collision.Top, collision.Left, collision.Height, collision.Width)
		maxHeight = max(maxHeight, int32(collision.Y))
		maxWidth = max(maxWidth, int32(collision.X))
		minHeight = min(minHeight, int32(collision.Y))
		minWidth = min(minWidth, int32(collision.X))
	}
	offsetWidth := minWidth
	offsetHeight := minHeight

	return NewSimulation(12, 12, int(offsetWidth), int(offsetHeight), collisions, players, enemies)
}

func main() {
	//collisions := []g.Coordinate{{7, 7}, {0, 0}, {3, 4}, {3, 3}, {3, 5}, {3, 2}, {3, 1}, {3, 6}, {3, 0}}
	//players := map[uint32]g.Coordinate{}
	//players[1] = g.Coordinate{X: 1, Y: 4}
	//enemies := map[uint32]*g.Enemy{}
	//enemy1 := g.NewTestEnemy(1, 5, 1)
	//enemy2 := g.NewTestEnemy(2, 7, 3)
	//enemy3 := g.NewTestEnemy(3, 6, 6)
	//enemy4 := g.NewTestEnemy(4, 4, 7)
	//enemies[enemy1.GetId()] = enemy1
	//enemies[enemy2.GetId()] = enemy2
	//enemies[enemy3.GetId()] = enemy3
	//enemies[enemy4.GetId()] = enemy4

	//collisions := []g.Coordinate{{0, 4}, {1, 4}, {2, 4}, {3, 4}, {4, 4}, {5, 4}, {5, 3}, {5, 2}, {5, 1}, {5, 0}, {7, 7}}
	//players := map[uint32]g.Coordinate{}
	//players[1] = g.Coordinate{X: 7, Y: 1}
	//enemies := map[uint32]*g.Enemy{}
	//enemy1 := g.NewTestEnemy(1, 1, 6)
	//enemies[enemy1.GetId()] = enemy1

	collisions := []g.Coordinate{{0, 0}, {8, 8}, {0, 5}, {1, 5}, {2, 5}, {3, 5}, {4, 5}, {5, 5},
		{0, 9}, {1, 9}, {2, 9}, {3, 9}, {4, 9}, {5, 9}}
	//collisions := []g.Coordinate{{2, 7}}
	players := map[uint32]g.Coordinate{}
	players[1] = g.Coordinate{X: 0, Y: 7}
	enemies := map[uint32]*g.Enemy{}
	enemy1 := g.NewTestEnemy(1, 10, 2)
	enemies[enemy1.GetId()] = enemy1
	enemy2 := g.NewTestEnemy(2, 11, 11)
	enemies[enemy2.GetId()] = enemy2

	sm := initSimulation(collisions, players, enemies)
	sm.startSimulation()
}

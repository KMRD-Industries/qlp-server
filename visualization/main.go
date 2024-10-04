package main

import (
	"fmt"
	"math"
	g "server/game-controllers"
)

type Simulation struct {
	width, height       int
	collisions, players []g.Coordinate
	enemies             []*g.Enemy
	graph               *[][]g.Cell
	algorithm           g.AIAlgorithm
}

func NewSimulation(width, height int, collisions, players []g.Coordinate, enemies []*g.Enemy) *Simulation {
	return &Simulation{
		width:      width,
		height:     height,
		collisions: collisions,
		players:    players,
		enemies:    enemies,
	}
}

func (s *Simulation) startSimulation() {
	s.algorithm.GetEnemiesUpdate(s.width, s.height, s.collisions, s.players, s.enemies)
	for _, enemy := range s.enemies {
		fmt.Printf("Enemy's new direction x: %f, y: %f\n", enemy.GetX(), enemy.GetY())
	}
}

func initSimulation(collisions, players []g.Coordinate, enemies []*g.Enemy) *Simulation {
	maxHeight := 0
	maxWidth := 0
	minHeight := math.MaxInt32
	minWidth := math.MaxInt32
	for _, collision := range collisions {
		maxHeight = max(maxHeight, collision.Y)
		maxWidth = max(maxWidth, collision.X)
		minHeight = min(minHeight, collision.Y)
		minWidth = min(minWidth, collision.X)
	}
	return NewSimulation(maxWidth-minWidth+1, maxHeight-minHeight+1, collisions, players, enemies)
}

func main() {
	collisions := []g.Coordinate{{9, 9, 0, 0}, {2, 2, 0, 0}}
	players := []g.Coordinate{{3, 3, 0, 0}}
	enemies := []*g.Enemy{g.NewEnemy(1, 0, 0)}
	//TODO działa dobrze na 1000x1000 - zrobić optymalizacjie żeby tworzyło mi graf na podstawie gdzie jest najbardziej
	// oddalony przeciwnik
	sm := initSimulation(collisions, players, enemies)
	sm.startSimulation()
}

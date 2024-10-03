package main

import (
	"fmt"
	"fyne.io/fyne/v2"
	g "server/game-controllers"
)

type Simulation struct {
	width, height       int
	grid                *fyne.Container
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

func main() {
	collisions := []g.Coordinate{{4, 0, 0, 0}}
	players := []g.Coordinate{{3, 3, 0, 0}}
	enemies := []*g.Enemy{g.NewEnemy(1, 0, 0)}
	//TODO działa dobrze na 1000x1000 - zrobić optymalizacjie żeby tworzyło mi graf na podstawie gdzie jest najbardziej
	// oddalony przeciwnik
	sm := NewSimulation(10, 10, collisions, players, enemies)
	sm.startSimulation()
}

package main

import (
	g "qlp_server/game-controllers"
)

func main() {
	width := 5
	height := 10
	collisions := []g.Coordinate{{1, 2}, {1, 3}, {2, 3}}
	player := g.Coordinate{2, 2}
	algorithm := g.NewAI(width, height, collisions, player)
	algorithm.CreateDistancesMap()
}

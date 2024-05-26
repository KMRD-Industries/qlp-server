package game_controllers

type AIAlgorithm struct {
	width      int
	height     int
	collisions []Coordinate // pierwsza tablica jest dla współrzędnych, każda tablica reprezentuje jeden blok kolizyjny
	player     Coordinate
	graph      *[][]int
}

type Coordinate struct {
	X, Y int
}

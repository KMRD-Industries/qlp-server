package game_controllers

func NewPlayer(x, y int) *Player {
	return &Player{
		position: Coordinate{X: x, Y: y},
		paths:    nil,
	}
}

type Player struct {
	position Coordinate
	paths    [][]Cell
}

func (p *Player) GetPosition() Coordinate {
	return p.position
}

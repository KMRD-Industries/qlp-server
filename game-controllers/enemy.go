package game_controllers

import "image/color"

type Enemy struct {
	position Coordinate
	moves    []int
	color    color.RGBA
}

func NewEnemy(x, y int, rgba color.RGBA) *Enemy {
	return &Enemy{
		position: Coordinate{X: x, Y: y},
		color:    rgba,
	}
}

func (e *Enemy) GetPosition() Coordinate {
	return e.position
}

func (e *Enemy) SetMoves(move []int) {
	e.moves = move
}

func (e *Enemy) GetMoves() []int {
	return e.moves
}

func (e *Enemy) GetColor() color.RGBA {
	return e.color
}

func (e *Enemy) SetPosition(newX, newY int) {
	e.position = Coordinate{newX, newY}
}

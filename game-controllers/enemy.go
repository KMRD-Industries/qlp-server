package game_controllers

import "github.com/ungerik/go3d/vec2"

type Enemy struct {
	id        uint32
	position  Coordinate
	direction vec2.T
}

func NewEnemy(id uint32, x, y int) *Enemy {
	return &Enemy{
		id:        id,
		position:  Coordinate{X: x, Y: y},
		direction: vec2.T{0, 0},
	}
}

func (e *Enemy) GetPosition() Coordinate {
	return e.position
}

func (e *Enemy) SetPosition(newX, newY int) {
	e.position = Coordinate{newX, newY, 0, 0}
}

func (e *Enemy) GetId() uint32 {
	return e.id
}

func (e *Enemy) GetDirection() vec2.T {
	return e.direction
}

func (e *Enemy) GetX() float32 {
	return e.direction.Get(1, 0)
}

func (e *Enemy) GetY() float32 {
	return e.direction.Get(0, 1)
}

func (e *Enemy) SetDirection(direction vec2.T) {
	e.direction = direction
}

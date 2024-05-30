package game_controllers

import (
	"image/color"
)

type Player struct {
	position Coordinate
	color    color.RGBA
}

func NewPlayer(x, y int, rgba color.RGBA) *Player {
	return &Player{
		position: Coordinate{X: x, Y: y},
		color:    rgba,
	}
}

func (p *Player) GetPosition() Coordinate {
	return p.position
}

func (p *Player) GetColor() color.RGBA {
	return p.color
}

func (p *Player) SetPosition(newX, newY int) {
	p.position = Coordinate{newX, newY}
}

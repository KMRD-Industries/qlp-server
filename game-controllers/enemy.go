package game_controllers

import (
	"github.com/ungerik/go3d/vec2"
	u "server/utils"
)

type Enemy struct {
	id            uint32
	position      Coordinate
	direction     vec2.T
	typ, name     string
	hp, damage    float64
	textureData   u.TextureData
	collisionData u.CollisionData
}

func NewEnemy(id uint32, x, y float32, typ, name string, hp, damage float64, textureData u.TextureData, collisionData u.CollisionData) *Enemy {
	return &Enemy{
		id:            id,
		position:      Coordinate{X: x, Y: y},
		direction:     vec2.T{0, 0},
		typ:           typ,
		name:          name,
		hp:            hp,
		damage:        damage,
		textureData:   textureData,
		collisionData: collisionData,
	}
}

func (e *Enemy) GetPosition() Coordinate {
	return e.position
}

func (e *Enemy) SetPosition(newX, newY float32) {
	e.position = Coordinate{newX, newY, 0, 0}
}

func (e *Enemy) GetId() uint32 {
	return e.id
}

func (e *Enemy) GetDirection() vec2.T {
	return e.direction
}

func (e *Enemy) GetDirectionX() float32 {
	return e.direction.Get(1, 0)
}

func (e *Enemy) GetDirectionY() float32 {
	return e.direction.Get(0, 1)
}

func (e *Enemy) SetDirection(direction vec2.T) {
	e.direction = direction
}

func (e *Enemy) GetType() string {
	return e.typ
}

func (e *Enemy) GetName() string {
	return e.name
}

func (e *Enemy) GetHp() float64 {
	return e.hp
}

func (e *Enemy) GetDamage() float64 {
	return e.damage
}

func (e *Enemy) GetTextureData() u.TextureData {
	return e.textureData
}

func (e *Enemy) GetCollisionData() u.CollisionData {
	return e.collisionData
}
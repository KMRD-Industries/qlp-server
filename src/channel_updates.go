package main

type GameUpdateVariant uint32

const (
	PLAYER_CONNECTED = iota
	PLAYER_DISCONNECTED
	ITEM_GENERATOR_REQUESTED
	GAME_RESTARTED
)

type GameUpdate struct {
	variant GameUpdateVariant
	player  Player
	u       uint32
	f       float32
	i       int64
}

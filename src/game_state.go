package main

import (
	"time"

	pb "github.com/kmrd-industries/qlp-proto-bindings/gen/go"
)

type Weapon struct {
	id uint32
}

type Player struct {
	id         uint32
	registered bool
	weapon     *Weapon
}

func (p *Player) toProtoPlayer() *pb.Player {
	return &pb.Player{
		Id:     p.id,
		Weapon: &pb.Weapon{Id: p.weapon.id},
	}
}

type Game struct {
	players   []Player
	weapons   map[uint32]*Weapon
	generator *ItemGenerator
	seed      int64
	playerIDs *idPool
	weaponIDs *idPool
}

func newGame() *Game {
	players := make([]Player, MAX_PLAYERS+1)
	for i := range players {
		players[i].registered = false
	}

	return &Game{
		players:   players,
		weapons:   make(map[uint32]*Weapon),
		generator: newGenerator(MAX_PLAYERS + 1),
		seed:      time.Now().Unix(),
		playerIDs: newIDPool(1),
		weaponIDs: newIDPool(100),
	}
}

func (g *Game) createInitialInfo() *pb.InitialInfo {
	playerID := g.playerIDs.getID()
	weaponID := g.weaponIDs.getID()

	player := &g.players[playerID]
	weapon := &Weapon{id: weaponID}

	player.id = playerID
	player.registered = true
	player.weapon = weapon

	g.weapons[weaponID] = weapon
	weapon.id = weaponID

	connectedPlayers := make([]*pb.Player, 0, MAX_PLAYERS)

	for _, p := range g.players {
		if p.registered && p.id != playerID {
			connectedPlayers = append(connectedPlayers, p.toProtoPlayer())
		}
	}

	return &pb.InitialInfo{
		Player:           player.toProtoPlayer(),
		Seed:             g.seed,
		ConnectedPlayers: connectedPlayers,
	}
}

func (g *Game) removePlayer(playerID uint32) {
	player := &g.players[playerID]
	weaponID := player.weapon.id

	player.registered = false
	player.weapon = nil

	delete(g.weapons, weaponID)
}

func (g *Game) requestItemGenerator(playerID uint32) *Item {
	return g.generator.requestItemGenerator(playerID)
}

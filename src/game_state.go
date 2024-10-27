package main

import (
	"math/rand/v2"
	"time"

	pb "github.com/kmrd-industries/qlp-proto-bindings/gen/go"
)

type Player struct {
	id         uint32
	registered bool
	items      []Item
}

func (p *Player) toProtoPlayer() *pb.Player {
	items := make([]*pb.Item, len(p.items))
	for i := range items {
		base := p.items[i]
		items[i] = &pb.Item{Gen: base.r, Type: base.variant}
	}

	return &pb.Player{
		Id:    p.id,
		Items: items,
	}
}

type Game struct {
	players   []Player
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
		generator: newGenerator(MAX_PLAYERS + 1),
		seed:      time.Now().Unix(),
		playerIDs: newIDPool(1),
		weaponIDs: newIDPool(100),
	}
}

func (g *Game) createInitialInfo() *pb.InitialInfo {
	playerID := g.playerIDs.getID()

	player := &g.players[playerID]
	items := make([]Item, 2)

	items[0].r = rand.Uint32()
	items[0].variant = pb.ItemType_WEAPON

	items[1].r = rand.Uint32()
	items[1].variant = pb.ItemType_HELMET

	player.id = playerID
	player.registered = true
	player.items = items

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

	player.registered = false
	player.items = nil

	g.playerIDs.returnID(playerID)
}

func (g *Game) getProtoPlayer(playerID uint32) *pb.Player {
	return g.players[playerID].toProtoPlayer()
}

func (g *Game) requestItemGenerator(playerID uint32) *Item {
	return g.generator.requestItemGenerator(playerID)
}

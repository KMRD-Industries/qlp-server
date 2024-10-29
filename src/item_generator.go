package main

import (
	"math/rand/v2"

	pb "github.com/kmrd-industries/qlp-proto-bindings/gen/go"
)

type Item struct {
	id      uint32
	r       uint32
	variant pb.ItemType
}

func (item *Item) intoProtoItem() *pb.Item {
	return &pb.Item{Id: item.id, Gen: item.r, Type: item.variant}
}

type ItemGenerator struct {
	currentGeneration  uint32
	randintGenerations map[uint32]uint32
	idGenerations      map[uint32]uint32
	nextRandint        []uint32
	nextID             []uint32
	nextGeneration     []uint32
	itemIDs            *idPool
}

func newGenerator(players int) *ItemGenerator {
	r := rand.Uint32()
	nextRandint := make([]uint32, players)
	nextGeneration := make([]uint32, players)
	nextID := make([]uint32, players)

	randintGenerations := make(map[uint32]uint32)
	randintGenerations[0] = r

	idPool := newIDPool(100)
	initialID := idPool.getID()
	idGenerations := make(map[uint32]uint32)
	idGenerations[0] = initialID

	for i := range nextRandint {
		nextRandint[i] = r
		nextID[i] = initialID
		nextGeneration[i] = 0
	}

	return &ItemGenerator{
		currentGeneration:  0,
		randintGenerations: randintGenerations,
		idGenerations:      idGenerations,
		nextRandint:        nextRandint,
		nextID:             nextID,
		nextGeneration:     nextGeneration,
		itemIDs:            newIDPool(100),
	}
}

func (ig *ItemGenerator) requestItemID() uint32 {
	return ig.itemIDs.getID()
}

func (ig *ItemGenerator) returnItemID(id uint32) {
	ig.itemIDs.returnID(id)
}

func (ig *ItemGenerator) requestItemGenerator(playerID uint32) *Item {
	r := ig.nextRandint[playerID]
	itemID := ig.nextID[playerID]

	gen := ig.nextGeneration[playerID]
	ig.nextGeneration[playerID]++

	firstToProcess := true
	generationExpired := true

	for id, otherGen := range ig.nextGeneration {
		if id == int(playerID) {
			continue
		}

		if otherGen > gen {
			firstToProcess = false
		} else {
			generationExpired = false
		}
	}

	if firstToProcess {
		ig.currentGeneration++
		ig.idGenerations[ig.currentGeneration] = ig.itemIDs.getID()
		ig.randintGenerations[ig.currentGeneration] = rand.Uint32()
	}

	if generationExpired {
		delete(ig.randintGenerations, gen)
		delete(ig.idGenerations, gen)
	}

	ig.nextRandint[playerID] = ig.randintGenerations[gen+1]
	ig.nextID[playerID] = ig.idGenerations[gen+1]

	return &Item{id: itemID, r: r, variant: pb.ItemType_WEAPON}
}

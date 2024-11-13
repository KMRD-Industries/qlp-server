package main

import (
	"math/rand/v2"

	pb "github.com/kmrd-industries/qlp-proto-bindings/gen/go"
)

var (
	SPAWN_THRESHOLDS = [...]float32{0.3, 0.6, 0.85, 1}
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
	variantGenerations map[uint32]pb.ItemType
	nextRandint        []uint32
	nextID             []uint32
	nextVariant        []pb.ItemType
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

	nextVariant := make([]pb.ItemType, players)
	variantGenerations := make(map[uint32]pb.ItemType)
	variantGenerations[0] = pb.ItemType_POTION

	for i := range nextRandint {
		nextRandint[i] = r
		nextID[i] = initialID
		nextGeneration[i] = 0
		nextVariant[i] = pb.ItemType_POTION
	}

	return &ItemGenerator{
		currentGeneration:  0,
		randintGenerations: randintGenerations,
		idGenerations:      idGenerations,
		variantGenerations: variantGenerations,
		nextRandint:        nextRandint,
		nextID:             nextID,
		nextVariant:        nextVariant,
		nextGeneration:     nextGeneration,
		itemIDs:            idPool,
	}
}

func (ig *ItemGenerator) requestItemID() uint32 {
	return ig.itemIDs.getID()
}

func (ig *ItemGenerator) returnItemID(id uint32) {
	ig.itemIDs.returnID(id)
}

func (ig *ItemGenerator) requestItemGenerator(playerID uint32, nextUint uint32, nextFloat float32) *Item {
	r := ig.nextRandint[playerID]
	itemID := ig.nextID[playerID]
	itemVariant := ig.nextVariant[playerID]

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
		ig.randintGenerations[ig.currentGeneration] = nextUint

		var nextVariant pb.ItemType
		switch r := nextFloat; {
		case r < SPAWN_THRESHOLDS[0]:
			nextVariant = pb.ItemType_POTION
		case r < SPAWN_THRESHOLDS[1]:
			nextVariant = pb.ItemType_WEAPON
		case r < SPAWN_THRESHOLDS[2]:
			nextVariant = pb.ItemType_HELMET
		default:
			nextVariant = pb.ItemType_ARMOUR
		}
		ig.variantGenerations[ig.currentGeneration] = nextVariant
	}

	if generationExpired {
		delete(ig.randintGenerations, gen)
		delete(ig.idGenerations, gen)
		delete(ig.variantGenerations, gen)
	}

	ig.nextRandint[playerID] = ig.randintGenerations[gen+1]
	ig.nextID[playerID] = ig.idGenerations[gen+1]
	ig.nextVariant[playerID] = ig.variantGenerations[gen+1]

	return &Item{id: itemID, r: r, variant: itemVariant}
}

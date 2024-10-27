package main

import (
	"math/rand/v2"

	pb "github.com/kmrd-industries/qlp-proto-bindings/gen/go"
)

type Item struct {
	r       uint32
	variant pb.ItemType
}

func (item *Item) intoProtoItem() *pb.Item {
	return &pb.Item{Gen: item.r, Type: item.variant}
}

type ItemGenerator struct {
	currentGeneration uint32
	generations       map[uint32]uint32
	nextRandint       []uint32
	nextGeneration    []uint32
}

func newGenerator(players int) *ItemGenerator {
	r := rand.Uint32()
	a := make([]uint32, players)
	b := make([]uint32, players)
	m := make(map[uint32]uint32)
	m[0] = r

	for i := range a {
		a[i] = r
		b[i] = 0
	}

	return &ItemGenerator{
		currentGeneration: 0,
		generations:       m,
		nextRandint:       a,
		nextGeneration:    b,
	}
}

func (ig *ItemGenerator) requestItemGenerator(playerID uint32) *Item {
	r := ig.nextRandint[playerID]

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
		ig.generations[ig.currentGeneration] = rand.Uint32()
	}

	if generationExpired {
		delete(ig.generations, gen)
	}

	ig.nextRandint[playerID] = ig.generations[gen+1]

	return &Item{r: r, variant: pb.ItemType_WEAPON}
}

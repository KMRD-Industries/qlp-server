package main

import (
	"container/heap"
	"log"
	"sync"
)

type idPool struct {
	availableIDs *PriorityQueue
	nextID       uint32
	lock         sync.Mutex
	minId, maxId uint32
}

func newIDPool(minId, maxId uint32) *idPool {
	pq := &PriorityQueue{}
	heap.Init(pq)
	return &idPool{
		availableIDs: pq,
		nextID:       minId,
		lock:         sync.Mutex{},
		maxId:        maxId,
		minId:        minId,
	}
}

func (p *idPool) getID() uint32 {
	p.lock.Lock()
	defer p.lock.Unlock()

	if p.availableIDs.Len() > 0 {
		return heap.Pop(p.availableIDs).(uint32)
	}

	if p.nextID > p.maxId {
		log.Printf("ERROR DURING ID ASSIGMENT: Id pool out of ids, current id: %d, maxId:%d\n", p.nextID, p.maxId)
	}

	id := p.nextID
	p.nextID++

	return id
}

func (p *idPool) returnID(id uint32) {
	p.lock.Lock()
	if id <= p.maxId {
		heap.Push(p.availableIDs, id)
	} else {
		log.Printf("ERORR DURING RETURNING ID: Id is not from the pool, id: %d, maxId: %d\n", id, p.maxId)
	}
	p.lock.Unlock()
}

type PriorityQueue []uint32

func (pq PriorityQueue) Len() int           { return len(pq) }
func (pq PriorityQueue) Less(i, j int) bool { return pq[i] < pq[j] }
func (pq PriorityQueue) Swap(i, j int)      { pq[i], pq[j] = pq[j], pq[i] }

func (pq *PriorityQueue) Push(x any) {
	*pq = append(*pq, x.(uint32))
}

func (pq *PriorityQueue) Pop() any {
	old := *pq
	n := len(old)
	item := old[n-1]
	*pq = old[0 : n-1]
	return item
}

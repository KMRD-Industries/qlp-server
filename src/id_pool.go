package main

import (
	"container/heap"
	"sync"
)

type IDPool struct {
	availableIDs *PriorityQueue
	nextID       uint32
	lock         sync.Mutex
}

func NewIDPool() *IDPool {
	pq := &PriorityQueue{}
	heap.Init(pq)
	return &IDPool{
		availableIDs: pq,
		nextID:       1,
		lock:         sync.Mutex{},
	}
}

func (p *IDPool) GetID() uint32 {
	p.lock.Lock()
	defer p.lock.Unlock()

	if p.availableIDs.Len() > 0 {
		return heap.Pop(p.availableIDs).(uint32)
	}
	id := p.nextID
	p.nextID++

	return id
}

func (p *IDPool) ReturnID(id uint32) {
	p.lock.Lock()
	heap.Push(p.availableIDs, id)
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

package main

import (
	"net"
	"sync"
)

type Clients struct {
	Lock     sync.Mutex
	TcpConns map[uint32]*net.TCPConn
}

func NewClientPool(initialCap uint32) *Clients {
	pool := &Clients{Lock: sync.Mutex{}, TcpConns: make(map[uint32]*net.TCPConn, initialCap)}

	return pool
}

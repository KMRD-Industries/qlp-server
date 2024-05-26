package main

import (
	"fmt"
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

func (c *Clients) Store(id uint32, conn *net.TCPConn) {
	c.Lock.Lock()
	c.TcpConns[id] = conn
	fmt.Printf("%+v\n", c.TcpConns)
	c.Lock.Unlock()
}

func (c *Clients) Delete(id uint32) {
	c.Lock.Lock()
	if conn, ok := c.TcpConns[id]; ok {
		conn.Close()
		delete(c.TcpConns, id)
	}
	c.Lock.Unlock()
}

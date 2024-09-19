package main

import (
	"errors"
	"io"
	"log"
	"net"
	"net/netip"
	"sync"
	"time"

	pb "github.com/kmrd-industries/qlp-proto-bindings/gen/go"
	"google.golang.org/protobuf/proto"
)

const (
	MAX_PLAYERS = 8
	SERVER_PORT = 9001
	BUF_SIZE    = 4096
)

var (
	ip        = net.ParseIP("127.0.0.1")
	addrPorts = make(map[uint32]netip.AddrPort, MAX_PLAYERS+1)
	tcpConns  = make(map[uint32]*net.TCPConn, MAX_PLAYERS+1)
	lock      = sync.RWMutex{}
	ids       = newIDPool()

	seed = time.Now().Unix()
)

func listenTCP() {
	addr := net.TCPAddr{
		IP:   ip,
		Port: SERVER_PORT,
	}

	listener, err := net.ListenTCP("tcp", &addr)

	if err != nil {
		log.Printf("Failed to open tcp socket: %v\n", err)
		return
	}
	defer listener.Close()

	for {
		conn, err := listener.AcceptTCP()
		id := ids.getID()

		if err != nil {
			log.Printf("Failed to accept tcp connection: %v\n", err)
		} else {
			stateUpdate := &pb.StateUpdate{
				Id:      id,
				Variant: pb.StateVariant_CONNECTED,
			}

			connectedPlayers := make([]uint32, 0, 8)
			lock.Lock()
			for otherID, c := range tcpConns {
				log.Printf("comm: %d %d\n", id, otherID)
				// collect connected players
				connectedPlayers = append(connectedPlayers, otherID)

				// inform connected players of new one
				encoded, _ := proto.Marshal(stateUpdate)
				c.Write(encoded)
			}

			// these will be monitored (we're assuming that closing conn means losing connection)
			tcpConns[id] = conn
			lock.Unlock()

			// inform player of current game state
			gameState := &pb.GameState{
				PlayerId:         id,
				Seed:             seed,
				ConnectedPlayers: connectedPlayers,
			}

			encoded, _ := proto.Marshal(gameState)
			log.Printf("connected: %d\n", id)

			conn.Write(encoded)
		}
	}
}

func handleTCP(ch chan uint32) {
	b := make([]byte, BUF_SIZE)

	msg := &pb.StateUpdate{
		Variant: pb.StateVariant_CONNECTED,
	}

	for {
		lock.RLock()
		for id, conn := range tcpConns {
			conn.SetReadDeadline(time.Now().Add(20 * time.Millisecond))
			n, err := conn.Read(b)

			if err == nil {
				err = proto.Unmarshal(b[:n], msg)
				if err != nil {
					log.Printf("Failed to deserialize state update: %v\n", err)
					continue
				}
				log.Printf("state update: %v\n", msg)

				for otherID, otherConn := range tcpConns {
					if id != otherID {
						otherConn.Write(b[:n])
					}
				}
				continue
			}

			if errors.Is(err, io.EOF) {
				ch <- id
				msg.Id = id
				msg.Variant = pb.StateVariant_DISCONNECTED

				for otherID, c := range tcpConns {
					if otherID != id {
						encoded, _ := proto.Marshal(msg)
						c.Write(encoded)
					}
				}
				log.Printf("disconnected %d\n", id)

				ids.returnID(id)

				lock.RUnlock()
				lock.Lock()
				if conn, ok := tcpConns[id]; ok {
					conn.Close()
					delete(tcpConns, id)
				}
				lock.Unlock()
				lock.RLock()
			}
		}
		lock.RUnlock()
	}
}

func handleUDP(ch chan uint32) {
	addr := net.UDPAddr{
		Port: SERVER_PORT,
		IP:   ip,
	}
	b := make([]byte, BUF_SIZE)

	conn, err := net.ListenUDP("udp", &addr)

	if err != nil {
		log.Printf("Failed to open udp socket: %v\n", err)
		return
	}
	defer conn.Close()

	for {
		conn.SetReadDeadline(time.Now().Add(200 * time.Millisecond))
		n, sender, err := conn.ReadFromUDP(b)

		select {
		case id := <-ch:
			delete(addrPorts, id)
		default:
		}

		if err == nil {
			received := &pb.PositionUpdate{}

			err = proto.Unmarshal(b[:n], received)

			log.Printf("%v\n", received)

			if err != nil {
				log.Printf("Failed to deserialize: %v\n", err)
				continue
			}

			senderAddrPort := sender.AddrPort()
			id := received.EntityId

			// skip packets from disconnected player
			lock.RLock()
			if _, ok := tcpConns[id]; !ok {
				continue
			}
			lock.RUnlock()

			if val, ok := addrPorts[id]; !ok || val != senderAddrPort {
				addrPorts[id] = senderAddrPort
			}

			// pass update to other players
			for otherID, addrPort := range addrPorts {
				if otherID != id {
					udpAddr := net.UDPAddrFromAddrPort(addrPort)
					conn.WriteToUDP(b[:n], udpAddr)
				}
			}
		}
	}
}

func main() {
	ch := make(chan uint32, 32)
	go handleUDP(ch)
	go listenTCP()
	go handleTCP(ch)

	for {
	}
}

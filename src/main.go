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
	SERVER_PORT = 10823
	BUF_SIZE    = 4096
)

var (
	ip        = net.ParseIP("127.0.0.1")
	addrPorts = make(map[uint32]netip.AddrPort, MAX_PLAYERS+1)
	tcpConns  = make(map[uint32]*net.TCPConn, MAX_PLAYERS+1)
	g         = newGame()
	gameLock  = sync.Mutex{}
	connLock  = sync.RWMutex{}

	seed = time.Now().Unix()
)

func listenTCP() {
	stateUpdate := &pb.StateUpdate{
		Player: &pb.Player{
			Id: 0,
		},
		Variant: pb.StateVariant_CONNECTED,
	}
	prefix := &pb.BytePrefix{}

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

		if err != nil {
			log.Printf("Failed to accept tcp connection: %v\n", err)
		} else {
			gameLock.Lock()
			initialInfo := g.createInitialInfo()
			id := initialInfo.Player.Id
			stateUpdate.Player = g.getProtoPlayer(id)
			gameLock.Unlock()

			// create message with prefix byte length of update
			// and the update itself
			bs, _ := proto.Marshal(stateUpdate)
			prefix.Bytes = uint32(len(bs))
			bp, _ := proto.Marshal(prefix)
			encoded := append(bp, bs...)

			connLock.Lock()
			for otherID, c := range tcpConns {
				log.Printf("comm: %d %d\n", id, otherID)

				// inform connected players of new one
				c.Write(encoded)
			}

			// these will be monitored (we're assuming that closing conn means losing connection)
			tcpConns[id] = conn
			connLock.Unlock()

			// inform player of current game state
			encoded, _ = proto.Marshal(initialInfo)
			log.Printf("connected: %d\n", id)

			conn.Write(encoded)
		}
	}
}

func handleTCP(ch chan uint32) {
	bs := make([]byte, BUF_SIZE)

	stateUpdate := &pb.StateUpdate{}
	prefix := &pb.BytePrefix{}

	for {
		connLock.RLock()
		for id, conn := range tcpConns {
			conn.SetReadDeadline(time.Now().Add(200 * time.Millisecond))
			n, err := conn.Read(bs)

			if err == nil {
				err = proto.Unmarshal(bs[:n], stateUpdate)
				if err != nil {
					log.Printf("Failed to deserialize state update: %v\n", err)
					continue
				}
				log.Printf("state update: %v\n", stateUpdate)

				switch stateUpdate.Variant {
				case pb.StateVariant_REQUEST_ITEM_GENERATOR:
					gameLock.Lock()
					stateUpdate.Item = g.requestItemGenerator(stateUpdate.Player.Id).intoProtoItem()
					gameLock.Unlock()
					bs, _ := proto.Marshal(stateUpdate)
					prefix.Bytes = uint32(len(bs))
					bp, _ := proto.Marshal(prefix)
					encoded := append(bp, bs...)

					conn.Write(encoded)
				default:
					for otherID, otherConn := range tcpConns {
						if id != otherID {
							prefix.Bytes = uint32(n)
							bp, _ := proto.Marshal(prefix)

							encoded := append(bp, bs[:n]...)

							otherConn.Write(encoded)
						}
					}
				}
				continue
			}

			if errors.Is(err, io.EOF) {
				ch <- id

				gameLock.Lock()
				g.removePlayer(id)
				gameLock.Unlock()

				msg := &pb.StateUpdate{
					Player:  &pb.Player{Id: id},
					Variant: pb.StateVariant_DISCONNECTED,
				}

				for otherID, c := range tcpConns {
					if otherID != id {
						bs, _ := proto.Marshal(msg)
						prefix.Bytes = uint32(len(bs))
						bp, _ := proto.Marshal(prefix)

						encoded := append(bp, bs...)

						c.Write(encoded)
					}
				}
				log.Printf("disconnected %d\n", id)

				connLock.RUnlock()
				connLock.Lock()
				if conn, ok := tcpConns[id]; ok {
					conn.Close()
					delete(tcpConns, id)
				}
				connLock.Unlock()
				connLock.RLock()
			}
		}
		connLock.RUnlock()
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
			received := &pb.MovementUpdate{}

			err = proto.Unmarshal(b[:n], received)

			if err != nil {
				log.Printf("Failed to deserialize: %v\n", err)
				continue
			}

			senderAddrPort := sender.AddrPort()
			id := received.EntityId

			// skip packets from disconnected player
			connLock.RLock()
			if _, ok := tcpConns[id]; !ok {
				continue
			}
			connLock.RUnlock()

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

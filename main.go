package main

import (
	"errors"
	"io"
	"log"
	"net"
	"net/netip"
	"time"

	pb "github.com/kmrd-industries/qlp-proto-bindings/gen/go"
	"google.golang.org/protobuf/proto"
)

const (
	SERVER_PORT = 9001
	BUF_SIZE    = 4096
)

var (
	ip          = net.ParseIP("127.0.0.1")
	addrPorts   = make(map[uint32]netip.AddrPort, 32)
	connections = NewClientPool(32)
	ids         = NewIDPool()
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
		id := ids.GetID()

		if err != nil {
			log.Printf("Failed to accept tcp connection: %v\n", err)
		} else {
			// inform player of received id
			msg := &pb.StateUpdate{
				Id:      id,
				Variant: pb.StateVariant_CONNECTED,
			}
			encoded, _ := proto.Marshal(msg)
			log.Printf("connected: %d\n", id)

			conn.Write(encoded)

			connections.Lock.Lock()
			for otherID, c := range connections.TcpConns {
				log.Printf("comm: %d %d\n", id, otherID)
				// inform new player of those already connected
				msg.Id = otherID
				encoded, _ = proto.Marshal(msg)
				conn.Write(encoded)

				// inform connected players of new one
				msg.Id = id
				encoded, _ = proto.Marshal(msg)
				c.Write(encoded)
			}
			connections.Lock.Unlock()

			// these will be monitored (we're assuming that closing conn means losing connection)
			connections.Store(id, conn)
		}
	}
}

// for now only for sending ids
func handleTCP() {
	b := make([]byte, BUF_SIZE)

	msg := &pb.StateUpdate{
		Variant: pb.StateVariant_CONNECTED,
	}

	for {
		connections.Lock.Lock()
		for id, conn := range connections.TcpConns {
			conn.SetReadDeadline(time.Now().Add(20 * time.Millisecond))
			_, err := conn.Read(b)

			if errors.Is(err, io.EOF) {
				msg.Id = id
				msg.Variant = pb.StateVariant_DISCONNECTED

				for otherID, c := range connections.TcpConns {
					if otherID != id {
						encoded, _ := proto.Marshal(msg)
						c.Write(encoded)
					}
				}
				log.Printf("disconnected %d\n", id)

				ids.ReturnID(id)
				connections.Delete(id)
				break
			}
		}
		connections.Lock.Unlock()
	}
}

func handleUDP() {
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

		if err == nil {
			received := &pb.PositionUpdate{}

			err = proto.Unmarshal(b[:n], received)

			if err != nil {
				log.Printf("Failed to deserialize: %v\n", err)
				continue
			}

			senderAddrPort := sender.AddrPort()
			id := received.EntityId
			if val, ok := addrPorts[id]; !ok || val != senderAddrPort {
				addrPorts[id] = senderAddrPort
			}

			// pass update to other players
			for otherID, addrPort := range addrPorts {
				if otherID != id {
					udpAddr := net.UDPAddrFromAddrPort(addrPort)
					conn.WriteToUDP(b[:n], udpAddr)
					// log.Printf("sending %d -> %d\n", id, otherID)
				}
			}
		}
	}
}

func main() {
	go handleUDP()
	go listenTCP()
	go handleTCP()

	for {
	}
}

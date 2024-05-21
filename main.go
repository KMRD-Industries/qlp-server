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
	ip        = net.ParseIP("127.0.0.1")
	addrPorts = make(map[uint32]netip.AddrPort, 32)
	tcp_conns = make(map[uint32]net.Conn, 32)
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

	var id uint32 = 1
	for ; ; id++ {
		conn, err := listener.AcceptTCP()

		if err != nil {
			log.Printf("Failed to accept tcp connection: %v\n", err)
		} else {
			// inform player of received id
			msg := &pb.StateUpdate{
				Id:      id,
				Variant: pb.StateVariant_CONNECTED,
			}
			encoded, _ := proto.Marshal(msg)

			conn.Write(encoded)
			// these will be monitored (we're assuming that closing conn means losing connection)
			tcp_conns[id] = conn

			go handleTCP(id, conn)
		}
	}
}

// for now only for sending ids
func handleTCP(id uint32, conn *net.TCPConn) {
	b := make([]byte, BUF_SIZE)
	defer conn.Close()

	msg := &pb.StateUpdate{
		Variant: pb.StateVariant_CONNECTED,
	}

	for otherID, c := range tcp_conns {
		msg.Id = otherID
		encoded, _ := proto.Marshal(msg)
		c.Write(encoded)
	}

	for {
		conn.SetReadDeadline(time.Now().Add(2000 * time.Millisecond))
		_, err := conn.Read(b)

		if errors.Is(err, io.EOF) {
			msg.Id = id
			msg.Variant = pb.StateVariant_DISCONNECTED

			for _, c := range tcp_conns {
				encoded, _ := proto.Marshal(msg)
				c.Write(encoded)
			}
			log.Printf("disconnected %d\n", id)

			delete(tcp_conns, id)
			break
		}
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
				}
			}
		}
	}
}

func main() {
	go handleUDP()
	go listenTCP()

	for {
	}
}

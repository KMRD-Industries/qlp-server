package main

import (
	"log"
	"net"
	"net/netip"
	"slices"
	"time"

	pb "github.com/kmrd-industries/qlp-proto-bindings/gen/go"
	"google.golang.org/protobuf/proto"
)

const (
	SERVER_PORT = 9001
	BUF_SIZE    = 4096
)

var (
	ip = net.ParseIP("127.0.0.1")
	// addrPorts = make(map[uint32]*net.UDPAddr, 32)
	addrPorts = make([]netip.AddrPort, 32)
	tcp_conns = make(map[uint32]net.Conn, 32)
)

// for now only for sending ids
func handleTCP() {
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

	var id uint32 = 0
	for id = 0; ; id++ {
		conn, err := listener.AcceptTCP()

		if err != nil {
			log.Printf("Failed to accept tcp connection: %v\n", err)
		} else {
			// inform player of received id
			msg := &pb.ConnectionReply{
				Id: id,
			}
			encoded, _ := proto.Marshal(msg)

			conn.Write(encoded)
			// these will be monitored (we're assuming that closing conn means losing connection)
			tcp_conns[id] = conn
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
			// log.Printf("%v\n", received)

			// id := received.EntityId

			// if _, ok := udp_addrs[id]; !ok {
			// 	udp_addrs[id] = sender
			// }

			senderAddrPort := sender.AddrPort()
			if !slices.Contains(addrPorts, senderAddrPort) {
				addrPorts = append(addrPorts, senderAddrPort)
			}

			// pass update to other players
			for _, addrPort := range addrPorts {
				// if id != otherID {
				if senderAddrPort != addrPort {
					udp_addr := net.UDPAddrFromAddrPort(addrPort)
					conn.WriteToUDP(b[:n], udp_addr)
				}
			}
		}
	}
}

func main() {
	go handleTCP()
	go handleUDP()

	for {
	}
}

package main

import (
	"errors"
	"fmt"
	pb "github.com/kmrd-industries/qlp-proto-bindings/gen/go"
	"google.golang.org/protobuf/proto"
	"io"
	"log"
	"math"
	"net"
	"net/netip"
	g "server/game-controllers"
	"time"
)

const (
	CONNECTION_PORT = 9001
	BUF_SIZE        = 4096
)

// TODO sprawdź czy ma sens rano
// na początku pokoju przesyłam wygląd całego pokoju
// potem w update przsyłam tylko to co się zmieniło czyli pewnie tylko pozycje graczy,
// potem odsyłam graczom i tutaj jest problem bo całe mapy czy pozycje gdzie ma się jaki stwór poruszyć - przekmiń to

var (
	connectionUpdateIp = net.ParseIP("127.0.0.1")
	addrPorts          = make(map[uint32]netip.AddrPort, 32)
	connections        = NewClientPool(32)
	ids                = NewIDPool()

	collisions = make([]g.Coordinate, 0)
	enemies    = make([]*g.Enemy, 0)
	players    = make([]g.Coordinate, 0)
	algorithm  = g.NewAIAlgorithm()
)

func listenConnectionUpdates() {
	addr := net.TCPAddr{
		IP:   connectionUpdateIp,
		Port: CONNECTION_PORT,
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

			// these will be monitored (we're assuming that closing conn means losing connection)
			connections.TcpConns[id] = conn
			connections.Lock.Unlock()

		}
	}
}

// for now only for sending ids
func handleTCP(ch chan uint32) {
	b := make([]byte, BUF_SIZE)

	//msg := &pb.StateUpdate{
	//	Variant: pb.StateVariant_CONNECTED,
	//}

	for {
		connections.Lock.Lock()
		for id, conn := range connections.TcpConns {
			conn.SetReadDeadline(time.Now().Add(20 * time.Millisecond))
			n, err := conn.Read(b)

			if errors.Is(err, io.EOF) {
				ch <- id
				handlePlayerDisconnect(id)
			}

			var msg pb.StateUpdate
			if err := proto.Unmarshal(b[:n], &msg); err != nil {
				log.Printf("Failded to unmarshall message from player %d, error: %v\n", id, err)
				continue
			}

			switch msg.Variant {
			case pb.StateVariant_DISCONNECTED:
				handlePlayerDisconnect(msg.Id)
			case pb.StateVariant_MAP_UPDATE:
				log.Printf("Map has been updated by user %d\n", msg.Id)
				handleMapUpdate(msg.MapPositionsUpdate)
				continue
			}
		}
		connections.Lock.Unlock()
	}
}

// sprawdzć czy działa
func handlePlayerDisconnect(id uint32) {
	msg := &pb.StateUpdate{Id: id,
		Variant: pb.StateVariant_DISCONNECTED}

	for otherID, c := range connections.TcpConns {
		if otherID != id {
			encoded, _ := proto.Marshal(msg)
			c.Write(encoded)
		}
	}
	log.Printf("disconnected %d\n", id)

	ids.ReturnID(id)

	if conn, ok := connections.TcpConns[id]; ok {
		conn.Close()
		delete(connections.TcpConns, id)
	}
}

func handleMapUpdate(update *pb.MapPositionsUpdate) {
	var maxHeight uint32 = 0
	var maxWidth uint32 = 0
	var minHeight uint32 = math.MaxUint32
	var minWidth uint32 = math.MaxUint32
	for _, obstacle := range update.Obstacles {
		fmt.Printf("Obstacle: top %d, left: %d, height: %d, width: %d\n", obstacle.Top, obstacle.Left, obstacle.Height, obstacle.Width)
		collisions = append(collisions, convertToCollision(obstacle))
		maxHeight = max(maxHeight, obstacle.Top)
		maxWidth = max(maxWidth, obstacle.Left)
		minHeight = min(minHeight, obstacle.Top)
		minWidth = min(minWidth, obstacle.Left)
	}
	fmt.Printf("length of the collision table: %d\n", len(collisions))

	algorithm.GetEnemiesUpdate(
		int(maxWidth-minWidth),
		int(maxHeight-minHeight),
		collisions,
		players,
		enemies,
	)

	//algorithm.SetHeight(int(maxHeight - minHeight))
	//algorithm.SetWidth(int(maxWidth - minWidth))
	//algorithm.SetCollisions(collisions)
	//algorithm.SetHeightOffset(int(minHeight))
	//algorithm.SetWidthOffset(int(minWidth))
	//players := make([]*game_controllers.Player, 0)

	//algorithm.CreateDistancesMap(int(maxWidth-minWidth), int(maxHeight-minHeight), collisions, players)
}

func convertToCollision(obstacle *pb.Obstacle) g.Coordinate {
	return g.Coordinate{
		X:      int(obstacle.Left),
		Y:      int(obstacle.Top),
		Height: int(obstacle.Height),
		Width:  int(obstacle.Width),
	}
}

func handleUDP(ch chan uint32) {
	addr := net.UDPAddr{
		Port: CONNECTION_PORT,
		IP:   connectionUpdateIp,
	}
	b := make([]byte, BUF_SIZE)

	conn, err := net.ListenUDP("udp", &addr)

	if err != nil {
		log.Printf("Failed to open udp socket: %v\n", err)
		return
	}
	defer conn.Close()

	for {
		// co to jest?
		conn.SetReadDeadline(time.Now().Add(200 * time.Millisecond))
		n, sender, err := conn.ReadFromUDP(b)

		select {
		case id := <-ch:
			delete(addrPorts, id)
		default:
		}

		if err == nil {
			receivedMessage := &pb.StateUpdate{}

			err := proto.Unmarshal(b[:n], receivedMessage)
			if err != nil {
				log.Printf("Failed to deserialize: %v\n", err)
				continue
			}

			switch receivedMessage.Variant {
			case pb.StateVariant_MAP_UPDATE:
				//log.Println("Map update received...")
				continue
			case pb.StateVariant_PLAYER_POSITION_UPDATE:
				positionUpdate := receivedMessage.PositionUpdate
				log.Printf("Position update received from: %d and position: %f, %f\n", positionUpdate.EntityId, positionUpdate.X, positionUpdate.Y)

				if err != nil {
					log.Printf("Failed to deserialize: %v\n", err)
					continue
				}

				senderAddrPort := sender.AddrPort()
				id := positionUpdate.EntityId
				//fmt.Println("Before looking for id in addrPort")
				//for val, _ := range addrPorts {
				//	fmt.Println(val)
				//}
				if val, ok := addrPorts[id]; !ok || val != senderAddrPort {
					addrPorts[id] = senderAddrPort
				}

				// pass update to other players
				for otherID, addrPort := range addrPorts {
					if otherID != id {
						udpAddr := net.UDPAddrFromAddrPort(addrPort)
						log.Printf("Sending message to: %d\n", otherID)
						msg := &pb.StateUpdate{
							Variant:        pb.StateVariant_PLAYER_POSITION_UPDATE,
							PositionUpdate: positionUpdate,
						}
						serializedMsg, err := proto.Marshal(msg)

						if err != nil {
							log.Printf("Failed to deserialize: %v\n", err)
						}
						_, err = conn.WriteToUDP(serializedMsg, udpAddr)
						if err != nil {
							log.Printf("Error while sending a message to %d\n", otherID)
						}
					}
				}
			}
		}
	}
}

func main() {
	ch := make(chan uint32, 32)
	go handleUDP(ch)
	go listenConnectionUpdates()
	go handleTCP(ch)

	for {
	}
}

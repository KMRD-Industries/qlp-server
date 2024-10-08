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
	"sync"
	"time"
)

const (
	MAX_PLAYERS = 8
	SERVER_PORT = 9001
	BUF_SIZE    = 4096
)

// TODO sprawdź czy ma sens rano
// na początku pokoju przesyłam wygląd całego pokoju
// potem w update przsyłam tylko to co się zmieniło czyli pewnie tylko pozycje graczy,
// potem odsyłam graczom i tutaj jest problem bo całe mapy czy pozycje gdzie ma się jaki stwór poruszyć - przekmiń to

var (
	ip        = net.ParseIP("127.0.0.1")
	addrPorts = make(map[uint32]netip.AddrPort, MAX_PLAYERS+1)
	tcpConns  = make(map[uint32]*net.TCPConn, MAX_PLAYERS+1)
	lock      = sync.RWMutex{}
	ids       = newIDPool()

	seed = time.Now().Unix()

	collisions = make([]g.Coordinate, 0)
	enemies    = make(map[uint32]*g.Enemy)
	players    = make(map[uint32]g.Coordinate)
	algorithm  = g.NewAIAlgorithm()
)

func listenConnectionUpdates() {
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
				//log.Printf("comm: %d %d\n", id, otherID)
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

	for {
		lock.RLock()
		for id, conn := range tcpConns {
			conn.SetReadDeadline(time.Now().Add(20 * time.Millisecond))
			n, err := conn.Read(b)

			if err == nil {
				var msg pb.StateUpdate
				err = proto.Unmarshal(b[:n], &msg)
				if err != nil {
					log.Printf("Failed to deserialize state update: %v\n", err)
					continue
				}
				//log.Printf("state update: %v\n", &msg)

				switch msg.Variant {
				case pb.StateVariant_DISCONNECTED:
					for otherID, otherConn := range tcpConns {
						if id != otherID {
							otherConn.Write(b[:n])
						}
					}
				case pb.StateVariant_MAP_UPDATE:

					log.Printf("Map has been updated by user %d\n", msg.Id)
					handleMapUpdate(msg.MapPositionsUpdate)
					// TODO jak tutaj przesyłam tą samą wiadomość na resztę połączeń to mi wybucha gra
					// sprawdź to albo nie rób tak
					enemiesToSend := make([]*pb.Enemy, 0, len(enemies))
					for _, enemy := range enemies {
						enemiesToSend = append(enemiesToSend, convertToProtoEnemy(enemy))
					}

					enemyPositionsUpdate := &pb.EnemyPositionsUpdate{
						EnemyPositions: enemiesToSend,
					}
					responseMsg := &pb.StateUpdate{
						Id:                   msg.Id,
						Variant:              pb.StateVariant_MAP_UPDATE,
						EnemyPositionsUpdate: enemyPositionsUpdate,
					}
					for _, conn := range tcpConns {
						serializedMsg, err := proto.Marshal(responseMsg)
						if err != nil {
							log.Printf("Failed to serialize enemy positions update, err: %s\n", err)
						}
						conn.Write(serializedMsg)
					}
					continue
				case pb.StateVariant_MAP_DIMENSIONS_UPDATE:
					log.Println("MAP DIMENSIONS HAS BEEN SET...")
					handleMapDimensionUpdate(msg.MapDimensionsUpdate)
				case pb.StateVariant_ROOM_CHANGED:
					for otherID, otherConn := range tcpConns {
						if id != otherID {
							otherConn.Write(b[:n])
						}
					}
				}
				continue
			}

			if errors.Is(err, io.EOF) {
				ch <- id
				handlePlayerDisconnect(id)
			}
		}
		lock.RUnlock()
	}
}

func handleMapDimensionUpdate(update *pb.MapDimensionsUpdate) {
	var maxHeight int32 = 0
	var maxWidth int32 = 0
	var minHeight int32 = math.MaxInt32
	var minWidth int32 = math.MaxInt32
	//fmt.Println("Obstacles: ")
	for _, obstacle := range update.Obstacles {
		//fmt.Printf("Obstacle: top %d, left: %d, height: %d, width: %d\n", obstacle.Top, obstacle.Left, obstacle.Height, obstacle.Width)
		collisions = append(collisions, convertToCollision(obstacle))
		maxHeight = max(maxHeight, obstacle.Top)
		maxWidth = max(maxWidth, obstacle.Left)
		minHeight = min(minHeight, obstacle.Top)
		minWidth = min(minWidth, obstacle.Left)
	}

	fmt.Printf("length of the collision table: %d\n", len(collisions))

	fmt.Printf("Height: %d, Real height: %d\nWidth: %d, Real width: %d\n", int(maxHeight-minHeight), maxHeight, int(maxWidth-minWidth), maxWidth)

	algorithm.SetWidth(int(maxWidth-minWidth) + 1)
	algorithm.SetHeight(int(maxHeight-minHeight) + 1)
	algorithm.SetOffset(int(minWidth), int(minHeight))
	algorithm.InitGraph()
}

func convertToProtoEnemy(enemy *g.Enemy) *pb.Enemy {
	return &pb.Enemy{
		Id: enemy.GetId(),
		X:  enemy.GetDirectionX(),
		Y:  enemy.GetDirectionY(),
	}
}

func handlePlayerDisconnect(id uint32) {
	msg := &pb.StateUpdate{Id: id,
		Variant: pb.StateVariant_DISCONNECTED}

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

func handleMapUpdate(update *pb.MapPositionsUpdate) {
	//fmt.Println("Players: ")
	fmt.Println(update.Players)
	for _, player := range update.Players {
		fmt.Printf("Player: x %d, y %d\n", player.X, player.Y)
		players[player.GetId()] = g.Coordinate{
			X:      int(player.X),
			Y:      int(player.Y),
			Height: 0,
			Width:  0,
		}
	}

	for _, enemy := range update.Enemies {
		//fmt.Printf("Enemy: x %f, y %f\n", enemy.GetX(), enemy.GetY())
		enemies[enemy.GetId()] = g.NewEnemy(enemy.GetId(), int(enemy.GetX()), int(enemy.GetY()))
		//fmt.Printf("Enemies length: %d\n", len(enemies))
	}

	algorithm.SetPlayers(players)
	algorithm.SetEnemies(enemies)

	start := time.Now()
	algorithm.GetEnemiesUpdate()
	elapsed := time.Since(start)
	//players = players[:0]
	//algorithm.ClearGraph()
	log.Printf("Finished after: %s\n", elapsed)
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

			//log.Printf("%v\n", received)

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
	go listenConnectionUpdates()
	go handleTCP(ch)

	for {
	}
}

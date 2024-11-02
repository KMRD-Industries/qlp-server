package main

import (
	"errors"
	pb "github.com/kmrd-industries/qlp-proto-bindings/gen/go"
	"google.golang.org/protobuf/proto"
	"io"
	"log"
	"math"
	"net"
	"net/netip"
	g "server/game-controllers"
	u "server/utils"
	"sync"
	"time"
)

const (
	MAX_PLAYERS     = 8
	SERVER_PORT     = 9001
	BUF_SIZE        = 4096
	SCALLING_FACTOR = 10
	PLAYER_MIN_ID   = 1
	PLAYER_MAX_ID   = 10
	ENEMY_MIN_ID    = PLAYER_MAX_ID + 1
	ENEMY_MAX_ID    = ENEMY_MIN_ID + 100
)

var (
	ip        = net.ParseIP("127.0.0.1")
	addrPorts = make(map[uint32]netip.AddrPort, MAX_PLAYERS+1)
	tcpConns  = make(map[uint32]*net.TCPConn, MAX_PLAYERS+1)
	lock      = sync.RWMutex{}
	playerIds = newIDPool(PLAYER_MIN_ID, PLAYER_MAX_ID)
	enemyIds  = newIDPool(ENEMY_MIN_ID, ENEMY_MAX_ID)

	seed = time.Now().Unix()

	collisions        = make([]g.Coordinate, 0)
	enemies           = make(map[uint32]*g.Enemy)
	players           = make(map[uint32]g.Coordinate)
	algorithm         = g.NewAIAlgorithm()
	isSpawned         = false
	spawnedEnemiesIds = make([]uint32, 0)
	config            = u.Config{}
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
		id := playerIds.getID()

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
				//TODO przemyśl co się stanie w przypadku jak gracz się rozłączy i będzie próbował ponownie dołączyć
				// jak mu wrogów wysyłać
				c.Write(encoded)
			}

			// these will be monitored (we're assuming that closing conn means losing connection)
			tcpConns[id] = conn
			lock.Unlock()

			// TODO dodać flagę z do spawnu potworów i w zależności od flagi odsyłać odpowiednie info, ręcznie usuwać spawnery jak zrespie potwory
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
					// TODO dodaj roomchange i tam daj id potworów
				case pb.StateVariant_MAP_UPDATE:
					//log.Printf("Map has been updated by user %d\n", msg.Id)
					handleMapUpdate(msg.MapPositionsUpdate)
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
				case pb.StateVariant_MAP_DIMENSIONS_UPDATE:
					//log.Println("MAP DIMENSIONS HAS BEEN SET...")
					handleMapDimensionUpdate(msg.MapDimensionsUpdate)
				case pb.StateVariant_ROOM_CHANGED:
					for otherID, otherConn := range tcpConns {
						if id != otherID {
							otherConn.Write(b[:n])
						}
					}
				case pb.StateVariant_SPAWN_ENEMY_REQUEST:
					if !isSpawned {
						handleSpawnEnemyRequest(msg.EnemyPositionsUpdate.EnemyPositions)
					}

					spawnedEnemies := make([]*pb.Enemy, 0, len(enemies))
					for _, enemy := range enemies {
						textureData := enemy.GetTextureData()
						collisionData := enemy.GetCollisionData()
						protoEnemy := &pb.Enemy{
							Id:     enemy.GetId(),
							X:      enemy.GetPosition().X * SCALLING_FACTOR,
							Y:      enemy.GetPosition().Y * SCALLING_FACTOR,
							Type:   enemy.GetType(),
							Name:   enemy.GetName(),
							Hp:     enemy.GetHp(),
							Damage: enemy.GetDamage(),
							TextureData: &pb.TextureData{
								TileId:    textureData.TileID,
								TileSet:   textureData.TileSet,
								TileLayer: textureData.TileLayer,
							},
							CollisionData: &pb.CollisionData{
								Type:    collisionData.Type,
								Width:   collisionData.Width,
								Height:  collisionData.Height,
								XOffset: collisionData.XOffset,
								YOffset: collisionData.YOffset,
							},
						}
						spawnedEnemies = append(spawnedEnemies, protoEnemy)
					}

					spawnedEnemiesMsg := &pb.EnemyPositionsUpdate{EnemyPositions: spawnedEnemies}

					responseMsg := &pb.StateUpdate{
						Id:                   id,
						Variant:              pb.StateVariant_SPAWN_ENEMY_REQUEST,
						EnemyPositionsUpdate: spawnedEnemiesMsg,
					}

					serializedMsg, err := proto.Marshal(responseMsg)
					if err != nil {
						log.Printf("Failed to serialize enemy spawn request response, err: %s\n", err)
					}

					tcpConns[id].Write(serializedMsg)
					//case pb.StateVariant_ENEMY_HP_UPDATE:
					//handleEnemyHpUpdate()
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

	//fmt.Printf("length of the collision table: %d\n", len(collisions))

	//fmt.Printf("Height: %d, Real height: %d\nWidth: %d, Real width: %d\n", int(maxHeight-minHeight), maxHeight, int(maxWidth-minWidth), maxWidth)

	algorithm.SetWidth(int((maxWidth-minWidth)/SCALLING_FACTOR) + 1)
	algorithm.SetHeight(int((maxHeight-minHeight)/SCALLING_FACTOR) + 1)
	algorithm.SetOffset(int(minWidth/SCALLING_FACTOR), int(minHeight/SCALLING_FACTOR))
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

	playerIds.returnID(id)

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
	//fmt.Println(update.Players)
	for _, player := range update.Players {
		//fmt.Printf("Player: x %d, y %d\n", player.X, player.Y)
		players[player.GetId()] = g.Coordinate{
			X:      player.X / SCALLING_FACTOR,
			Y:      player.Y / SCALLING_FACTOR,
			Height: 0,
			Width:  0,
		}
	}

	for _, enemy := range update.Enemies {
		//TODO sprawdź czy enemies jest puste
		enemies[enemy.GetId()].SetPosition(enemy.GetX()/SCALLING_FACTOR, enemy.GetY()/SCALLING_FACTOR)
	}

	// TODO nie wiem czy nie da się jakoś sprytniej tego przypisywać - sprawdź to
	algorithm.SetPlayers(players)
	algorithm.SetEnemies(enemies)

	//start := time.Now()
	algorithm.GetEnemiesUpdate()
	algorithm.ClearGraph()
	//elapsed := time.Since(start)
	//log.Printf("Finished after: %s\n", elapsed)
}

func handleSpawnEnemyRequest(enemiesToSpawn []*pb.Enemy) {
	for _, enemyToSpawn := range enemiesToSpawn {
		enemyId, err := spawnEnemy(enemyToSpawn)
		if err != nil {
			continue
		}
		spawnedEnemiesIds = append(spawnedEnemiesIds, enemyId)
	}
	isSpawned = true
}

func spawnEnemy(enemyToSpawn *pb.Enemy) (uint32, error) {
	newEnemyId := enemyIds.getID()
	//TODO jakoś rozwiązać problem z dzieleniem przez scalling factor przeciwników bo jak przesyłam przy spawnie info
	// to je muszę spowrotem mnożyć xDDDDDDDDDD
	//TODO na razie tylko potwory typu MELEE będą respione
	enemyConfig := config.EnemyData[0]
	enemies[newEnemyId] = g.NewEnemy(
		newEnemyId,
		enemyToSpawn.X/SCALLING_FACTOR,
		enemyToSpawn.Y/SCALLING_FACTOR,
		enemyConfig.Type,
		enemyConfig.Name,
		enemyConfig.HP,
		enemyConfig.Damage,
		enemyConfig.TextureData,
		enemyConfig.CollisionData,
	)
	log.Printf("Spawned enemy with id: %d, position %f %f, hp: %f\n", newEnemyId, enemyToSpawn.X, enemyToSpawn.Y, enemyConfig.HP)

	return newEnemyId, nil
}

func convertToCollision(obstacle *pb.Obstacle) g.Coordinate {
	return g.Coordinate{
		X:      float32(obstacle.Left / SCALLING_FACTOR),
		Y:      float32(obstacle.Top / SCALLING_FACTOR),
		Height: float32(obstacle.Height / SCALLING_FACTOR),
		Width:  float32(obstacle.Width / SCALLING_FACTOR),
	}
}

func handleEnemyHpUpdate(enemy *pb.Enemy) {

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
	var err error
	config, err = u.NewJsonParser().ParseConfig("utils/config.json")
	if err != nil {
		return
	}
	ch := make(chan uint32, 32)
	go handleUDP(ch)
	go listenConnectionUpdates()
	go handleTCP(ch)

	for {
	}
}

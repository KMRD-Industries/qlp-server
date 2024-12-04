package main

import (
	"bufio"
	"bytes"
	"compress/zlib"
	"errors"
	"flag"
	"fmt"
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
	BUF_SIZE        = 8192
	SCALLING_FACTOR = 16
	PLAYER_MIN_ID   = 1
	PLAYER_MAX_ID   = 10
	ENEMY_MIN_ID    = PLAYER_MAX_ID + 1
	ENEMY_MAX_ID    = ENEMY_MIN_ID + 100
	ITEM_MIN_ID     = ENEMY_MAX_ID + 1
	ITEM_MAX_ID     = ITEM_MIN_ID + 100
	PREFIX_SIZE     = 3
	DIFF            = 4096
)

var (
	ipString  = flag.String("a", "127.0.0.1", "server ip address")
	ip        = net.ParseIP("127.0.0.1")
	addrPorts = make(map[uint32]netip.AddrPort, MAX_PLAYERS+1)
	tcpConns  = make(map[uint32]*net.TCPConn, MAX_PLAYERS+1)
	game      = newGame()
	gameLock  = sync.Mutex{}
	connLock  = sync.RWMutex{}
	enemyIds  = newIDPool(ENEMY_MIN_ID, ENEMY_MAX_ID)

	collisions        = make([]g.Coordinate, 0)
	enemies           = make(map[uint32]*g.Enemy)
	players           = make(map[uint32]g.Coordinate)
	algorithm         = g.NewAIAlgorithm()
	isSpawned         = false
	spawnedEnemiesIds = make([]uint32, 0)
	config            = u.Config{}
	isGraph           = false
)

func listenTCP() {
	stateUpdate := &pb.StateUpdate{
		Player: &pb.Player{
			Id: 0,
		},
		Variant: pb.StateVariant_CONNECTED,
	}

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
			initialInfo := game.createInitialInfo()
			id := initialInfo.Player.Id
			stateUpdate.Player = game.getProtoPlayer(id)
			gameLock.Unlock()

			// create message with prefix byte length of update
			// and the update itself
			serializedMsg, _ := proto.Marshal(stateUpdate)
			encoded := addPrefixAndPadding(serializedMsg)

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
	updateSeries := &pb.StateUpdateSeries{}

	for {
		connLock.RLock()
		for id, conn := range tcpConns {
			conn.SetReadDeadline(time.Now().Add(20 * time.Millisecond))
			reader := bufio.NewReaderSize(conn, BUF_SIZE)

			_, err := reader.Peek(PREFIX_SIZE)

			if errors.Is(err, io.EOF) {
				ch <- id

				gameLock.Lock()
				game.removePlayer(id)
				gameLock.Unlock()

				msg := &pb.StateUpdate{
					Player:  &pb.Player{Id: id},
					Variant: pb.StateVariant_DISCONNECTED,
				}

				for otherID, c := range tcpConns {
					if otherID != id {
						serializedMsg, _ := proto.Marshal(msg)
						encoded := addPrefixAndPadding(serializedMsg)

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

				if len(tcpConns) == 0 {
					gameLock.Lock()
					game = newGame()
					gameLock.Unlock()
				}
				connLock.Unlock()
				connLock.RLock()
			}

			if err != nil {
				//log.Println("there is not enough bytes to read size", err)
				continue
			}

			//sizeBuffer := make([]byte, 2)
			//_, err = io.ReadFull(reader, sizeBuffer)
			//
			//size := binary.BigEndian.Uint16(sizeBuffer)
			//_, err = reader.Peek(int(size))

			encodedPrefixMsg := make([]byte, PREFIX_SIZE)
			_, err = io.ReadFull(reader, encodedPrefixMsg)

			var prefixMsg pb.BytePrefix
			err = proto.Unmarshal(encodedPrefixMsg, &prefixMsg)
			if err != nil {
				log.Println("Couldn't unmarshall prefix message, err: ", err)
			}

			size := prefixMsg.GetBytes() - DIFF
			_, err = reader.Peek(int(size))
			if err != nil {
				log.Printf("there is not enough bytes to read for this message\nmessage size %d, err: %v\n", size, err)
				continue
			}

			messageBuffer := make([]byte, size)
			_, err = io.ReadFull(reader, messageBuffer)
			if err == nil {
				err = proto.Unmarshal(messageBuffer, updateSeries)
				if err != nil {
					continue
				}

				for _, update := range updateSeries.GetUpdates() {
					log.Printf("state update: %v\n", update)

					switch update.Variant {
					case pb.StateVariant_REQUEST_ITEM_GENERATOR:
						gameLock.Lock()
						update.Item = game.requestItemGenerator(update.Player.Id).intoProtoItem()
						gameLock.Unlock()
						serializedMsg, _ := proto.Marshal(update)
						encoded := addPrefixAndPadding(serializedMsg)

						conn.Write(encoded)
					case pb.StateVariant_MAP_DIMENSIONS_UPDATE:
						//TODO dodaj isMapUpdated, żeby kilku playerów nie mogło updatować mapy
						handleMapDimensionUpdate(update.CompressedMapDimensionsUpdate)
					case pb.StateVariant_ROOM_CHANGED:
						handleRoomChange(update, id)
					case pb.StateVariant_SPAWN_ENEMY_REQUEST:
						if !isSpawned {
							handleSpawnEnemyRequest(update.EnemyPositionsUpdate.EnemyPositions)
						}
						handleSendSpawnedEnemies()
					default:
						for otherID, otherConn := range tcpConns {
							if id != otherID {
								serializedMsg, _ := proto.Marshal(update)
								encoded := addPrefixAndPadding(serializedMsg)

								otherConn.Write(encoded)
							}
						}
					}
					continue
				}
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
			msg := &pb.StateUpdate{}

			err = proto.Unmarshal(b[:n], msg)
			if err != nil {
				log.Printf("Failed to deserialize: %v\n", err)
				continue
			}

			switch msg.Variant {
			case pb.StateVariant_PLAYER_POSITION_UPDATE:
				positionUpdate := msg.MovementUpdate
				senderAddrPort := sender.AddrPort()
				id := positionUpdate.EntityId

				connLock.RLock()
				// skip packets from disconnected player
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
			case pb.StateVariant_MAP_UPDATE:
				if isGraph {
					handleMapUpdate(msg, conn)
				}
			}
		}
	}
}

func handleSendSpawnedEnemies() {
	spawnedEnemies := make([]*pb.Enemy, 0, len(enemies))
	for _, enemy := range enemies {
		textureData := enemy.GetTextureData()
		collisionData := enemy.GetCollisionData()
		protoEnemy := &pb.Enemy{
			Id:        enemy.GetId(),
			PositionX: float32(enemy.GetPosition().X) * SCALLING_FACTOR,
			PositionY: float32(enemy.GetPosition().Y) * SCALLING_FACTOR,
			Type:      enemy.GetType(),
			Name:      enemy.GetName(),
			Hp:        enemy.GetHp(),
			Damage:    enemy.GetDamage(),
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
		Variant:              pb.StateVariant_SPAWN_ENEMY_REQUEST,
		EnemyPositionsUpdate: spawnedEnemiesMsg,
	}

	serializedMsg, err := proto.Marshal(responseMsg)
	if err != nil {
		log.Printf("Failed to serialize enemy spawn request response, err: %s\n", err)
	}

	encoded := addPrefixAndPadding(serializedMsg)

	for playerId, conn := range tcpConns {
		log.Printf("Sent spawned enemies to player %d\n", playerId)
		_, err2 := conn.Write(encoded)
		if err2 != nil {
			log.Printf("Couldn't send spawned enmies to the client, err: %s\n", err2)
		}
	}
}

func handleRoomChange(msg *pb.StateUpdate, id uint32) {
	log.Println("Room changed")
	log.Println("-------------------------------")
	enemies = make(map[uint32]*g.Enemy)
	players = make(map[uint32]g.Coordinate)
	isGraph = false
	isSpawned = false

	responseMsg := pb.StateUpdate{
		Variant: pb.StateVariant_ROOM_CHANGED,
		Room:    msg.Room,
	}

	serializedMsg, err := proto.Marshal(&responseMsg)
	if err != nil {
		fmt.Errorf("failed to serialize enemy spawn request response, err: %s\n", err)
	}

	encoded := addPrefixAndPadding(serializedMsg)
	for otherID, otherConn := range tcpConns {
		if id != otherID {
			otherConn.Write(encoded)
		}
	}
}

func addPrefixAndPadding(serializedMsg []byte) []byte {
	prefix := &pb.BytePrefix{}
	prefix.Bytes = uint32(len(serializedMsg) + DIFF)

	serialisedPrefix, err := proto.Marshal(prefix)
	if err != nil {
		fmt.Errorf("failed to serialize enemy spawn request response, err: %s\n", err)
	}

	return append(serialisedPrefix, serializedMsg...)
}

func handleMapDimensionUpdate(update []byte) {
	decompressedUpdate := decompressMessage(update)

	var maxHeight int32 = 0
	var maxWidth int32 = 0
	var minHeight int32 = math.MaxInt32
	var minWidth int32 = math.MaxInt32

	var mapDimensionUpdate pb.MapDimensionsUpdate
	if err := proto.Unmarshal(decompressedUpdate, &mapDimensionUpdate); err != nil {
		fmt.Errorf("failed to unmarshal decompressedUpd")
	}

	for _, obstacle := range mapDimensionUpdate.Obstacles {
		collisions = append(collisions, convertToCollision(obstacle))
		maxHeight = max(maxHeight, obstacle.Top)
		maxWidth = max(maxWidth, obstacle.Left)
		minHeight = min(minHeight, obstacle.Top)
		minWidth = min(minWidth, obstacle.Left)
	}

	algorithm.SetWidth(int((maxWidth-minWidth)/SCALLING_FACTOR) + 1)
	algorithm.SetHeight(int((maxHeight-minHeight)/SCALLING_FACTOR) + 1)
	algorithm.SetOffset(int(minWidth/SCALLING_FACTOR), int(minHeight/SCALLING_FACTOR))

	algorithm.SetCollision(collisions)
	algorithm.InitGraph()
	collisions = make([]g.Coordinate, 0)
	isGraph = true
	log.Printf("Map size is %d\n -------------------------\n", len(mapDimensionUpdate.Obstacles))
}

func decompressMessage(update []byte) []byte {
	zLibReader, err := zlib.NewReader(bytes.NewReader(update))
	if err != nil {
		fmt.Errorf("failed to create zlib reader: %v\n", err)
		return nil
	}
	defer zLibReader.Close()

	decompressedUpdate, err := io.ReadAll(zLibReader)
	if err != nil {
		fmt.Errorf("failed to decompress map dimension update data: %v\n", err)
	}
	return decompressedUpdate
}

func convertToProtoEnemy(enemy *g.Enemy) *pb.Enemy {
	return &pb.Enemy{
		Id:        enemy.GetId(),
		PositionX: enemy.GetDirectionX(),
		PositionY: enemy.GetDirectionY(),
	}
}

func handleSpawnEnemyRequest(enemiesToSpawn []*pb.Enemy) {
	for _, enemyToSpawn := range enemiesToSpawn {
		enemyId := spawnEnemy(enemyToSpawn)
		spawnedEnemiesIds = append(spawnedEnemiesIds, enemyId)
	}
	isSpawned = true
}

// TODO zrobić rozróżnianie na różnych przeciwników
func spawnEnemy(enemyToSpawn *pb.Enemy) uint32 {
	newEnemyId := enemyIds.getID()
	//TODO jakoś rozwiązać problem z dzieleniem przez scalling factor przeciwników bo jak przesyłam przy spawnie info
	// to je muszę spowrotem mnożyć xDDDDDDDDDD
	enemyConfig := config.EnemyData[0]
	enemies[newEnemyId] = g.NewEnemy(
		newEnemyId,
		int(enemyToSpawn.PositionX/SCALLING_FACTOR),
		int(enemyToSpawn.PositionY/SCALLING_FACTOR),
		enemyConfig.Type,
		enemyConfig.Name,
		enemyConfig.HP,
		enemyConfig.Damage,
		enemyConfig.TextureData,
		enemyConfig.CollisionData,
	)
	log.Printf("Spawned enemy with id: %d, position %f %f, hp: %f\n", newEnemyId, enemyToSpawn.PositionX, enemyToSpawn.PositionY, enemyConfig.HP)

	return newEnemyId
}

func convertToCollision(obstacle *pb.Obstacle) g.Coordinate {
	return g.Coordinate{
		X: int(obstacle.Left / SCALLING_FACTOR),
		Y: int(obstacle.Top / SCALLING_FACTOR),
	}
}

func handleMapUpdate(msg *pb.StateUpdate, conn *net.UDPConn) {
	update := msg.MapPositionsUpdate

	addPlayers(update.Players)
	addEnemies(update.Enemies)

	// TODO nie wiem czy nie da się jakoś sprytniej tego przypisywać - sprawdź to
	algorithm.SetPlayers(players)
	algorithm.SetEnemies(enemies)

	algorithm.GetEnemiesUpdate()
	algorithm.ClearGraph()

	enemiesToSend := make([]*pb.Enemy, 0, len(enemies))
	for _, enemy := range enemies {
		enemiesToSend = append(enemiesToSend, convertToProtoEnemy(enemy))
	}

	enemyPositionsUpdate := &pb.EnemyPositionsUpdate{
		EnemyPositions: enemiesToSend,
	}

	responseMsg := &pb.StateUpdate{
		Variant:              pb.StateVariant_MAP_UPDATE,
		EnemyPositionsUpdate: enemyPositionsUpdate,
	}

	serializedMsg, err := proto.Marshal(responseMsg)
	if err != nil {
		log.Printf("Failed to serialize enemy positions update, err: %s\n", err)
	}

	//log.Printf(">>Enemy's vector x: %f, y: %f\n", )
	for _, addrPort := range addrPorts {
		udpAddr := net.UDPAddrFromAddrPort(addrPort)
		conn.WriteToUDP(serializedMsg, udpAddr)
	}
}

func addPlayers(playersProto []*pb.Player) {
	//TODO napraw handlowanie tego że plpayer się rozłącza i dalej jest dodawany do grafu
	for _, player := range playersProto {
		players[player.GetId()] = g.Coordinate{
			X: int(player.PositionX / SCALLING_FACTOR),
			Y: int(player.PositionY / SCALLING_FACTOR),
		}
	}
}

func addEnemies(enemiesProto []*pb.Enemy) {
	for _, enemy := range enemiesProto {
		enemyOnBoard := enemies[enemy.GetId()]
		if enemyOnBoard != nil {
			enemies[enemy.GetId()].SetPosition(int(enemy.PositionX/SCALLING_FACTOR), int(enemy.PositionY/SCALLING_FACTOR))
		}
	}
}

func main() {
	var err error
	config, err = u.NewJsonParser().ParseConfig("utils/config.json")
	if err != nil {
		return
	}
	flag.Parse()

	if parsedIP := net.ParseIP(*ipString); parsedIP != nil {
		ip = parsedIP
	}

	log.Printf("Starting server on: %v\n", ip)

	ch := make(chan uint32, 32)
	go handleUDP(ch)
	go listenTCP()
	go handleTCP(ch)

	for {
	}
}

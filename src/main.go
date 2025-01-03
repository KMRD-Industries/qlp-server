package main

import (
	"bufio"
	"bytes"
	"compress/zlib"
	"errors"
	"flag"
	pb "github.com/kmrd-industries/qlp-proto-bindings/gen/go"
	"google.golang.org/protobuf/proto"
	"io"
	"log"
	"log/slog"
	"math"
	"net"
	"net/netip"
	"os"
	g "server/game-controllers"
	u "server/utils"
	"sync"
	"sync/atomic"
	"time"
)

const (
	MAX_PLAYERS     = 8
	SERVER_PORT     = 10823
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
	isSpawned         atomic.Bool
	isMapUpdated      atomic.Bool
	spawnedEnemiesIds = make([]uint32, 0)
	config            = u.Config{}
	logger            = slog.New(slog.NewTextHandler(os.Stderr, nil))
)

type SingleFlight struct {
	lock chan struct{}
}

func NewSingleFlight() *SingleFlight {
	return &SingleFlight{lock: make(chan struct{}, 1)}
}

func (s *SingleFlight) TryExecute(f func()) bool {
	select {
	case s.lock <- struct{}{}:
		f()
		<-s.lock
		return true
	default:
		return false
	}
}

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

func handleTCP(userCh chan uint32, graphCh chan bool) {
	updateSeries := &pb.StateUpdateSeries{}

	for {
		connLock.RLock()
		for id, conn := range tcpConns {
			conn.SetReadDeadline(time.Now().Add(20 * time.Millisecond))
			reader := bufio.NewReaderSize(conn, BUF_SIZE)

			_, err := reader.Peek(PREFIX_SIZE)

			if errors.Is(err, io.EOF) {
				userCh <- id

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

			encodedPrefixMsg := make([]byte, PREFIX_SIZE)
			_, err = io.ReadFull(reader, encodedPrefixMsg)

			var prefixMsg pb.BytePrefix
			err = proto.Unmarshal(encodedPrefixMsg, &prefixMsg)
			if err != nil {
				logger.Info("Couldn't unmarshall prefix message", "error", err)
			}

			size := prefixMsg.GetBytes() - DIFF
			_, err = reader.Peek(int(size))
			if err != nil {
				logger.Info("Not enough bytes to read", "message size", size, "error", err)
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
					logger.Info("Incoming state update", "update", update)

					switch update.Variant {
					case pb.StateVariant_REQUEST_ITEM_GENERATOR:
						gameLock.Lock()
						update.Item = game.requestItemGenerator(update.Player.Id).intoProtoItem()
						gameLock.Unlock()
						serializedMsg, _ := proto.Marshal(update)
						encoded := addPrefixAndPadding(serializedMsg)

						conn.Write(encoded)
					case pb.StateVariant_MAP_DIMENSIONS_UPDATE:
						if !isMapUpdated.Load() {
							isMapUpdated.Store(true)
							handleMapDimensionUpdate(update.CompressedMapDimensionsUpdate)
						}
					case pb.StateVariant_ROOM_CHANGED:
						graphCh <- false
						handleRoomChange(update, id)
					case pb.StateVariant_SPAWN_ENEMY_REQUEST:
						if !isSpawned.Load() {
							handleSpawnEnemyRequest(update.EnemySpawnerPositions)
						}
						handleSendSpawnedEnemies()
						graphCh <- true
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

func handleUDP(userCh chan uint32, graphCh chan bool, sf *SingleFlight) {
	addr := net.UDPAddr{
		Port: SERVER_PORT,
		IP:   ip,
	}
	b := make([]byte, BUF_SIZE)
	isGraph := false

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
		case id := <-userCh:
			delete(addrPorts, id)
		default:
		}

		select {
		case graph := <-graphCh:
			isGraph = graph
		default:
		}

		if err == nil {
			movementUpdate := &pb.MovementUpdate{}

			err = proto.Unmarshal(b[:n], movementUpdate)
			if err != nil {
				logger.Info("Failed to deserialize", "error", err)
				continue
			}

			switch movementUpdate.Variant {
			case pb.MovementVariant_PLAYER_MOVEMENT_UPDATE:
				senderAddrPort := sender.AddrPort()
				id := movementUpdate.EntityId

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
			case pb.MovementVariant_MAP_UPDATE:
				sf.TryExecute(func() {
					if isGraph {
						handleMapUpdate(movementUpdate.MapPositionsUpdate, conn)
					}
				})
			}
		}
	}
}

func handleSendSpawnedEnemies() {
	responseMsg := &pb.StateUpdate{
		Variant: pb.StateVariant_SPAWN_ENEMY_REQUEST,
	}

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
		responseMsg.EnemySpawnerPositions = append(responseMsg.GetEnemySpawnerPositions(), protoEnemy)
	}

	serializedMsg, err := proto.Marshal(responseMsg)
	if err != nil {
		logger.Info("Failed to serialize enemy spawn request response", "error", err)
	}

	encoded := addPrefixAndPadding(serializedMsg)

	for playerId, conn := range tcpConns {
		logger.Debug("Sent spawned enemies to player", "playerId", playerId)
		_, err2 := conn.Write(encoded)
		if err2 != nil {
			logger.Info("Couldn't send spawned enemies to the client", "error", err2)
		}
	}
}

func handleRoomChange(msg *pb.StateUpdate, id uint32) {
	enemies = make(map[uint32]*g.Enemy)
	players = make(map[uint32]g.Coordinate)
	isSpawned.Store(false)
	isMapUpdated.Store(false)

	responseMsg := pb.StateUpdate{
		Variant: pb.StateVariant_ROOM_CHANGED,
		Room:    msg.Room,
	}

	serializedMsg, err := proto.Marshal(&responseMsg)
	if err != nil {
		logger.Info("Failed to serialize enemy spawn request response", "error", err)
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
		logger.Info("Failed to serialize enemy spawn request response", "error", err)
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
		logger.Info("Failed to unmarshal decompressedUpd")
	}

	for _, obstacle := range mapDimensionUpdate.Obstacles {
		collisions = append(collisions, convertToCollision(obstacle))
		maxHeight = max(maxHeight, int32(obstacle.Top))
		maxWidth = max(maxWidth, int32(obstacle.Left))
		minHeight = min(minHeight, int32(obstacle.Top))
		minWidth = min(minWidth, int32(obstacle.Left))
	}

	algorithm.Mutex.Lock()
	defer algorithm.Mutex.Unlock()
	algorithm.SetWidth(int((maxWidth-minWidth)/SCALLING_FACTOR) + 1)
	algorithm.SetHeight(int((maxHeight-minHeight)/SCALLING_FACTOR) + 1)
	algorithm.SetOffset(int(minWidth/SCALLING_FACTOR), int(minHeight/SCALLING_FACTOR))

	algorithm.SetCollision(collisions)
	algorithm.InitGraph()
	collisions = make([]g.Coordinate, 0)
}

func decompressMessage(update []byte) []byte {
	zLibReader, err := zlib.NewReader(bytes.NewReader(update))
	if err != nil {
		logger.Info("Failed to create zlib reader", "error", err)
		return nil
	}
	defer zLibReader.Close()

	decompressedUpdate, err := io.ReadAll(zLibReader)
	if err != nil {
		logger.Info("Failed to decompress map dimension update data", "error", err)
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
	isSpawned.Store(true)
}

func spawnEnemy(enemyToSpawn *pb.Enemy) uint32 {
	newEnemyId := enemyIds.getID()
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

	return newEnemyId
}

func convertToCollision(obstacle *pb.Obstacle) g.Coordinate {
	return g.Coordinate{
		X: int(math.Ceil(float64(obstacle.Left / SCALLING_FACTOR))),
		Y: int(math.Ceil(float64(obstacle.Top / SCALLING_FACTOR))),
	}
}

func handleMapUpdate(update *pb.MapPositionsUpdate, conn *net.UDPConn) {
	algorithm.Mutex.Lock()

	addPlayers(update.Players)
	addEnemies(update.Enemies)

	algorithm.SetPlayers(players)
	algorithm.SetEnemies(enemies)

	algorithm.CreateDistancesMap()
	algorithm.ClearGraph()

	algorithm.Mutex.Unlock()

	responseMsg := &pb.MovementUpdate{
		Variant: pb.MovementVariant_MAP_UPDATE,
	}

	for _, enemy := range enemies {
		enemyToSend := convertToProtoEnemy(enemy)
		responseMsg.EnemyPositions = append(responseMsg.EnemyPositions, enemyToSend)
	}

	serializedMsg, err := proto.Marshal(responseMsg)
	if err != nil {
		logger.Info("Failed to serialize enemy positions update", "error", err)
	}

	for _, addrPort := range addrPorts {
		udpAddr := net.UDPAddrFromAddrPort(addrPort)
		conn.WriteToUDP(serializedMsg, udpAddr)
	}
}

func addPlayers(playersProto []*pb.Player) {
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
		logger.Info("Error while parsing config", err)
		return
	}
	flag.Parse()

	if parsedIP := net.ParseIP(*ipString); parsedIP != nil {
		ip = parsedIP
	}

	log.Printf("Starting server on: %v\n", ip)

	userCh := make(chan uint32, 32)
	graphCh := make(chan bool)
	isSpawned.Store(false)
	isMapUpdated.Store(false)

	//mapDimensionsCh <- false
	sf := NewSingleFlight()

	go handleUDP(userCh, graphCh, sf)
	go listenTCP()
	go handleTCP(userCh, graphCh)

	for {
	}
}

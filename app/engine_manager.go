package app

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"sync"
	"sync/atomic"
	"time"

	"wiredscriptengine/internal/logger"
	"wiredscriptengine/internal/webserver"

	"github.com/wailsapp/wails/v2/pkg/runtime"
	g "xabbo.b7c.io/goearth"
	"xabbo.b7c.io/goearth/in"
	"xabbo.b7c.io/goearth/out"
)

var appInfo AppInfo

func init() {
	data, err := os.ReadFile("wails.json")
	if err != nil {
		appInfo = AppInfo{
			Name:        "G-Controller",
			Version:     "0.1.0",
			Description: "G-Controller",
			Author:      "cebolla1",
		}
		return
	}

	var config struct {
		Name string `json:"name"`
		Info struct {
			ProductName    string `json:"productName"`
			ProductVersion string `json:"productVersion"`
			Comments       string `json:"comments"`
		} `json:"info"`
		Author struct {
			Name string `json:"name"`
		} `json:"author"`
	}
	if err := json.Unmarshal(data, &config); err != nil {
		appInfo = AppInfo{
			Name:        "G-Controller",
			Version:     "0.1.0",
			Description: "Move with Joystick, send chat and more...",
			Author:      "cebolla1",
		}
		return
	}

	appInfo = AppInfo{
		Name:        config.Info.ProductName,
		Version:     config.Info.ProductVersion,
		Description: config.Info.Comments,
		Author:      config.Author.Name,
	}
}

type EngineManager struct {
	ctx       context.Context
	ext       *g.Ext
	hub       *webserver.Hub
	mu        sync.RWMutex
	connected bool
	roomID    int
	hotel     string
	canEdit   bool
	canRead   bool

	userX     int
	userY     int
	userDir   int
	userID    int

	moveDir atomic.Int32
	moveWg  sync.WaitGroup
	stopCh  chan struct{}
}

func NewEngineManager(ctx context.Context) *EngineManager {
	ext := g.NewExt(g.ExtInfo{
		Title:       appInfo.Name,
		Description: appInfo.Description,
		Author:      appInfo.Author,
		Version:     appInfo.Version,
	})

	em := &EngineManager{
		ctx: ctx,
		ext: ext,
	}

	logger.EnableTrace(true)

	ext.Initialized(em.onInitialized)
	ext.Connected(em.onConnected)
	ext.Disconnected(em.onDisconnected)

	ext.Intercept(in.Chat, in.Whisper, in.Shout).With(em.handleChat)
	ext.Intercept(in.OpenConnection).With(em.handleOpenConnection)
	ext.Intercept(in.UserUpdate).With(em.handleUserUpdate)
	ext.Intercept(in.WiredVariablesForObject).With(em.handleWiredVariables)
	ext.Intercept(in.WiredPermissions).With(em.handleWiredPermissions)

	return em
}

func (em *EngineManager) SetHub(hub *webserver.Hub) {
	em.hub = hub
}

func (em *EngineManager) Start() {
	logger.Info("Starting GoEarth extension...")
	em.ext.Run()
}

func (em *EngineManager) Stop() {
	logger.Info("Stopping GoEarth extension...")
}

func (em *EngineManager) GetStatus() EngineStatus {
	em.mu.RLock()
	defer em.mu.RUnlock()

	return EngineStatus{
		Connected: em.connected,
		RoomID:    em.roomID,
		Hotel:     em.hotel,
		CanEdit:   em.canEdit,
		CanRead:   em.canRead,
	}
}

func (em *EngineManager) GetUserPosition() UserPosition {
	em.mu.RLock()
	defer em.mu.RUnlock()
	return UserPosition{
		ID:   em.userID,
		X:    em.userX,
		Y:    em.userY,
		Dir:  em.userDir,
	}
}

func (em *EngineManager) SendChat(msg string) {
	if em.ext == nil || !em.ext.IsConnected() {
		return
	}
	go em.ext.Send(out.Chat, msg, 0, 2)
	logger.Info("Sent chat: %s", msg)
}

func (em *EngineManager) SendMoveAvatar(targetX, targetY int) {
	if em.ext == nil || !em.ext.IsConnected() {
		return
	}
	logger.Info("MoveAvatar -> x=%d y=%d", targetX, targetY)
	go em.ext.Send(out.MoveAvatar, targetX, targetY)
}

func (em *EngineManager) SendLook(dir int) {
	if em.ext == nil || !em.ext.IsConnected() {
		return
	}
	dx, dy := em.directionDelta(dir)
	em.mu.RLock()
	targetX := em.userX + dx
	targetY := em.userY + dy
	em.mu.RUnlock()
	go em.ext.Send(out.LookTo, targetX, targetY)
}


func (em *EngineManager) HandleMove(dir int, x, y float64) {
	if dir == 0 && (x != 0 || y != 0) {
		dir = em.calculateDirection(x, y)
	}

	if dir == 0 {
		return
	}

	if em.moveDir.Load() == 0 {
		em.stopCh = make(chan struct{})
		em.moveWg.Add(1)
		go em.moveLoop()
		logger.Info("Movement started dir=%d", dir)
	}

	em.moveDir.Store(int32(dir))
}

func (em *EngineManager) StopMove() {
	if em.moveDir.Load() != 0 {
		em.moveDir.Store(0)
		close(em.stopCh)
		em.moveWg.Wait()
		logger.Info("Movement stopped")
	}
}

func (em *EngineManager) moveLoop() {
	defer em.moveWg.Done()
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	var predictX, predictY int
	initialized := false

	for {
		select {
		case <-em.stopCh:
			return
		case <-ticker.C:
		}

		dir := int(em.moveDir.Load())
		if dir == 0 {
			return
		}

		dx, dy := em.directionDelta(dir)

		em.mu.RLock()
		if !initialized {
			predictX = em.userX
			predictY = em.userY
			initialized = true
		}
		predictX += dx
		predictY += dy
		em.mu.RUnlock()

		if em.ext.IsConnected() {
			em.ext.Send(out.MoveAvatar, predictX, predictY)
		}
	}
}

func (em *EngineManager) calculateDirection(x, y float64) int {
	if y < -0.3 && x > -0.3 && x < 0.3 {
		return 1
	} else if y > 0.3 && x > -0.3 && x < 0.3 {
		return 5
	} else if x < -0.3 && y > -0.3 && y < 0.3 {
		return 7
	} else if x > 0.3 && y > -0.3 && y < 0.3 {
		return 3
	} else if x < -0.3 && y < -0.3 {
		return 8
	} else if x > 0.3 && y < -0.3 {
		return 2
	} else if x < -0.3 && y > 0.3 {
		return 6
	} else if x > 0.3 && y > 0.3 {
		return 4
	}
	return 0
}

func (em *EngineManager) directionDelta(dir int) (int, int) {
	switch dir {
	case 1:
		return -1, -1
	case 2:
		return 0, -1
	case 3:
		return 1, -1
	case 4:
		return 1, 0
	case 5:
		return 1, 1
	case 6:
		return 0, 1
	case 7:
		return -1, 1
	case 8:
		return -1, 0
	}
	return 0, 0
}

func (em *EngineManager) updateHubStatus() {
	if em.hub == nil {
		return
	}
	em.mu.RLock()
	connected := em.connected
	roomID := em.roomID
	hotel := em.hotel
	canEdit := em.canEdit
	canRead := em.canRead
	userX := em.userX
	userY := em.userY
	userDir := em.userDir
	em.mu.RUnlock()
	em.hub.UpdateStatus(connected, roomID, hotel, canEdit, canRead, userX, userY, userDir)
}

func (em *EngineManager) onInitialized(e g.InitArgs) {
	logger.Info("Extension initialized")
	runtime.EventsEmit(em.ctx, "engine:initialized")
}

func (em *EngineManager) onConnected(e g.ConnectArgs) {
	em.mu.Lock()
	em.connected = true
	em.hotel = extractHotelCode(e.Host)
	em.mu.Unlock()

	logger.Info("Game connected (%s) - Hotel: %s", e.Host, em.hotel)

	runtime.EventsEmit(em.ctx, "engine:connected", map[string]interface{}{
		"hotel": em.hotel,
	})
	em.updateHubStatus()
}

func (em *EngineManager) onDisconnected() {
	em.mu.Lock()
	em.connected = false
	em.roomID = 0
	em.userX = 0
	em.userY = 0
	if em.moveDir.Load() != 0 {
		em.moveDir.Store(0)
		close(em.stopCh)
		em.moveWg.Wait()
	}
	em.mu.Unlock()

	logger.Info("Game disconnected")
	runtime.EventsEmit(em.ctx, "engine:disconnected")
	em.updateHubStatus()
}

func (em *EngineManager) handleChat(e *g.Intercept) {
	e.Packet.ReadInt()
	msg := e.Packet.ReadString()

	runtime.EventsEmit(em.ctx, "game:chat", msg)
	if em.hub != nil {
		em.hub.OnGameChat(msg)
	}
}

func (em *EngineManager) handleUserUpdate(e *g.Intercept) {
	packet := e.Packet
	if packet.Length() < 20 {
		return
	}

	userID := packet.ReadInt()
	packet.ReadInt()
	x := packet.ReadInt()
	y := packet.ReadInt()
	packet.ReadString()
	direction := packet.ReadInt()
	packet.ReadInt()
	packet.ReadInt()
	packet.ReadString()

	em.mu.Lock()
	em.userID = userID
	em.userX = x
	em.userY = y
	em.userDir = direction
	em.mu.Unlock()

	logger.Trace("UserUpdate: id=%d pos=(%d,%d) dir=%d", userID, x, y, direction)
	em.updateHubStatus()
}

func (em *EngineManager) handleOpenConnection(e *g.Intercept) {
	packet := e.Packet
	roomID := packet.ReadInt()

	em.mu.Lock()
	em.roomID = roomID
	em.mu.Unlock()

	logger.Info("Room ID detected: %d", roomID)

	runtime.EventsEmit(em.ctx, "engine:room-detected", map[string]interface{}{
		"roomId": roomID,
	})
	em.updateHubStatus()
}

func (em *EngineManager) handleWiredVariables(e *g.Intercept) {
	packet := e.Packet
	if packet.Length() < 8 {
		return
	}

	packet.ReadInt()
	count := packet.ReadInt()

	currentRoomID := 0
	for i := 0; i < count; i++ {
		if packet.Pos >= packet.Length() {
			break
		}

		key := packet.ReadString()
		value := packet.ReadInt()

		if key == "-380" {
			currentRoomID = value
		}
	}

	if currentRoomID == 0 {
		return
	}

	em.mu.Lock()
	em.roomID = currentRoomID
	em.mu.Unlock()

	logger.Info("Room ID detected: %d", currentRoomID)

	runtime.EventsEmit(em.ctx, "engine:room-detected", map[string]interface{}{
		"roomId": currentRoomID,
	})
	em.updateHubStatus()
}

func (em *EngineManager) handleWiredPermissions(e *g.Intercept) {
	packet := e.Packet

	if packet.Length() < 2 {
		em.mu.Lock()
		em.canEdit = false
		em.canRead = false
		em.mu.Unlock()
		logger.Info("WiredPermissions: No access")
		em.updateHubStatus()
		return
	}
	canEdit := packet.ReadBool()
	canRead := packet.ReadBool()

	em.mu.Lock()
	em.canEdit = canEdit
	em.canRead = canRead
	em.mu.Unlock()

	logger.Info("WiredPermissions: edit=%v, read=%v", canEdit, canRead)
	em.updateHubStatus()
}

func extractHotelCode(host string) string {
	hotel := "es"
	if len(host) > 0 {
		fmt.Scan(host, "%s", &hotel)
	}
	return hotel
}

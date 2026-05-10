package app

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"sync"
	"sync/atomic"
	"time"

	"g-controller/internal/navigator"
	"g-controller/internal/room"
	"g-controller/internal/webserver"

	"github.com/wailsapp/wails/v2/pkg/runtime"
	g "xabbo.b7c.io/goearth"
	"xabbo.b7c.io/goearth/in"
	"xabbo.b7c.io/goearth/out"
)

var appInfo AppInfo

func init() {
	data, err := os.ReadFile("wails.json")
	if err != nil {
		return
	}

	var config struct {
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
	ctx             context.Context
	ext             *g.Ext
	hub             *webserver.Hub
	roomMgr         *room.Manager
	mu              sync.RWMutex
	connected       bool
	roomID          int
	hotel           string
	canEdit         bool
	canRead         bool
	navigatorResult *navigator.SearchResult

	userX   int
	userY   int
	userDir int
	userID  int

	moveDir atomic.Int32
	serverX atomic.Int32
	serverY atomic.Int32
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

	em.roomMgr = room.New(ext, em.onUserPosition, em.onLocalUserFound)

	ext.Initialized(em.onInitialized)
	ext.Connected(em.onConnected)
	ext.Disconnected(em.onDisconnected)
	ext.Activated(em.onActivated)

	ext.Intercept(in.Chat, in.Whisper, in.Shout).With(em.handleChat)
	ext.Intercept(in.OpenConnection).With(em.handleOpenConnection)
	ext.Intercept(in.WiredPermissions).With(em.handleWiredPermissions)
	ext.Intercept(in.NavigatorSearchResultBlocks).With(em.handleNavigatorSearchResultBlocks)

	return em
}

func (em *EngineManager) SetHub(hub *webserver.Hub) {
	em.hub = hub
}

func (em *EngineManager) Start() {
	em.ext.Run()
}

func (em *EngineManager) Stop() {
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
		ID:  em.userID,
		X:   em.userX,
		Y:   em.userY,
		Dir: em.userDir,
	}
}

func (em *EngineManager) SendChat(msg string) {
	if em.ext == nil || !em.ext.IsConnected() {
		return
	}
	go em.ext.Send(out.Chat, msg, 0, 2)
}

func (em *EngineManager) SendMoveAvatar(targetX, targetY int) {
	if em.ext == nil || !em.ext.IsConnected() {
		return
	}
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

func (em *EngineManager) RequestEnterRoom(flatId int) {
	if em.ext == nil || !em.ext.IsConnected() {
		return
	}
	go em.ext.Send(out.GetGuestRoom, flatId, 0, 1)
}

func (em *EngineManager) SearchNavigator(view, query string) {
	if em.ext == nil || !em.ext.IsConnected() {
		return
	}
	go em.ext.Send(out.NewNavigatorSearch, view, query)
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
	}

	em.moveDir.Store(int32(dir))
}

func (em *EngineManager) StopMove() {
	if em.moveDir.Load() != 0 {
		em.moveDir.Store(0)
		close(em.stopCh)
		em.moveWg.Wait()
	}
}

func (em *EngineManager) moveLoop() {
	defer em.moveWg.Done()
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	em.mu.RLock()
	predictX := em.userX
	predictY := em.userY
	em.mu.RUnlock()

	sendMove := func() bool {
		dir := int(em.moveDir.Load())
		if dir == 0 {
			return false
		}
		sx := int(em.serverX.Load())
		sy := int(em.serverY.Load())
		if abs(sx-predictX) > 1 || abs(sy-predictY) > 1 {
			predictX = sx
			predictY = sy
		}
		dx, dy := em.directionDelta(dir)
		predictX += dx
		predictY += dy
		if em.ext.IsConnected() {
			em.ext.Send(out.MoveAvatar, predictX, predictY)
		}
		return true
	}

	if !sendMove() {
		return
	}

	for {
		select {
		case <-em.stopCh:
			return
		case <-ticker.C:
			if !sendMove() {
				return
			}
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

func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
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
	runtime.EventsEmit(em.ctx, "engine:initialized")
}

func (em *EngineManager) onActivated() {
	runtime.WindowShow(em.ctx)
	runtime.EventsEmit(em.ctx, "engine:activated")
}

func (em *EngineManager) onConnected(e g.ConnectArgs) {
	em.mu.Lock()
	em.connected = true
	em.hotel = extractHotelCode(e.Host)
	em.mu.Unlock()

	em.roomMgr.ProbeLocalIndex()

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

func (em *EngineManager) onUserPosition(x, y, dir int) {
	em.mu.Lock()
	em.userX = x
	em.userY = y
	em.userDir = dir
	em.mu.Unlock()

	em.serverX.Store(int32(x))
	em.serverY.Store(int32(y))

	em.updateHubStatus()
}

func (em *EngineManager) onLocalUserFound(index int) {
	em.mu.Lock()
	em.userID = index
	em.mu.Unlock()

}

func (em *EngineManager) handleOpenConnection(e *g.Intercept) {
	packet := e.Packet
	roomID := packet.ReadInt()

	em.mu.Lock()
	em.roomID = roomID
	em.mu.Unlock()

	runtime.EventsEmit(em.ctx, "engine:room-detected", map[string]interface{}{
		"roomId": roomID,
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
		em.updateHubStatus()
		return
	}
	canEdit := packet.ReadBool()
	canRead := packet.ReadBool()

	em.mu.Lock()
	em.canEdit = canEdit
	em.canRead = canRead
	em.mu.Unlock()

	em.updateHubStatus()
}

func (em *EngineManager) handleNavigatorSearchResultBlocks(e *g.Intercept) {
	result := navigator.Parse(e.Packet)

	em.mu.Lock()
	em.navigatorResult = &result
	em.mu.Unlock()

	if em.hub != nil {
		em.hub.BroadcastNavigatorResult(result)
	}
}

func extractHotelCode(host string) string {
	hotel := "es"
	if len(host) > 0 {
		fmt.Scan(host, "%s", &hotel)
	}
	return hotel
}

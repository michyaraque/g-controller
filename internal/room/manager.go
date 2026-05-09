package room

import (
	"fmt"
	"time"

	g "xabbo.b7c.io/goearth"
	"xabbo.b7c.io/goearth/in"
	"xabbo.b7c.io/goearth/out"
)

const (
	EntityTypeHabbo int = 1
	EntityTypePet   int = 2
	EntityTypeBot   int = 4
)

type Manager struct {
	ext            *g.Ext
	users          map[int]Entity
	localUserID    int
	onPosition     func(x, y, dir int)
	onLocalUserFound func(index int)
}

func New(ext *g.Ext, onPosition func(x, y, dir int), onLocalUserFound func(index int)) *Manager {
	m := &Manager{
		ext:            ext,
		users:          make(map[int]Entity),
		onPosition:     onPosition,
		onLocalUserFound: onLocalUserFound,
	}

	ext.Intercept(in.Users).With(m.onUsers)
	ext.Intercept(in.UserRemove).With(m.onUserRemove)
	ext.Intercept(in.RoomReady).With(m.onRoomReady)
	ext.Intercept(in.UserUpdate).With(m.onUserUpdate)
	ext.Intercept(in.Expression).With(m.onExpression)

	return m
}

func (m *Manager) ProbeLocalIndex() {
	if m.ext != nil && m.ext.IsConnected() {
		go m.ext.Send(out.AvatarExpression, 0)
	}
}

func (m *Manager) SetLocalUserID(id int) {
	m.localUserID = id
}

func (m *Manager) onExpression(e *g.Intercept) {
	pkt := e.Packet
	index := pkt.ReadInt()
	_ = pkt.ReadInt()

	if m.localUserID == 0 {
		m.localUserID = index
		if m.onLocalUserFound != nil {
			m.onLocalUserFound(index)
		}
	}
}

func (m *Manager) onUsers(e *g.Intercept) {
	pkt := e.Packet
	count := pkt.ReadInt()

	for i := 0; i < count; i++ {
		_ = pkt.ReadInt()
		name := pkt.ReadString()
		_ = pkt.ReadString()
		_ = pkt.ReadString()
		index := pkt.ReadInt()
		x := pkt.ReadInt()
		y := pkt.ReadInt()
		_ = pkt.ReadString()
		_ = pkt.ReadInt()
		entityType := pkt.ReadInt()

		if entityType == EntityTypeHabbo {
			m.users[index] = Entity{
				Name:     name,
				JoinTime: time.Now(),
				Tile:     Tile{X: x, Y: y},
			}
			_ = pkt.ReadString()
			_ = pkt.ReadInt()
			_ = pkt.ReadInt()
			_ = pkt.ReadString()
			_ = pkt.ReadString()
			_ = pkt.ReadInt()
			_ = pkt.ReadBool()
		} else if entityType == EntityTypePet {
			_ = pkt.ReadInt()
			_ = pkt.ReadInt()
			_ = pkt.ReadString()
			_ = pkt.ReadInt()
			_ = pkt.ReadBool()
			_ = pkt.ReadBool()
			_ = pkt.ReadBool()
			_ = pkt.ReadBool()
			_ = pkt.ReadBool()
			_ = pkt.ReadBool()
			_ = pkt.ReadInt()
			_ = pkt.ReadString()
		} else if entityType == EntityTypeBot {
			_ = pkt.ReadString()
			_ = pkt.ReadInt()
			_ = pkt.ReadString()
			followCount := pkt.ReadInt()
			for f := 0; f < followCount; f++ {
				_ = pkt.ReadShort()
			}
		}
	}

	_ = len(m.users)
}

func (m *Manager) onUserRemove(e *g.Intercept) {
	indexStr := e.Packet.ReadString()
	var index int
	fmt.Sscanf(indexStr, "%d", &index)

	delete(m.users, index)
	m.localUserID = 0
	m.users = make(map[int]Entity)
}

func (m *Manager) onRoomReady(e *g.Intercept) {
	_ = e.Packet.ReadString()
	_ = e.Packet.ReadInt()

	m.users = make(map[int]Entity)
	m.localUserID = 0

	m.ProbeLocalIndex()
}

func (m *Manager) onUserUpdate(e *g.Intercept) {
	pkt := e.Packet
	_ = pkt.ReadInt()
	index := pkt.ReadInt()
	x := pkt.ReadInt()
	y := pkt.ReadInt()
	_ = pkt.ReadString()
	direction := pkt.ReadInt()
	_ = pkt.ReadInt()
	_ = pkt.ReadInt()
	_ = pkt.ReadString()

	if ent, ok := m.users[index]; ok {
		ent.Tile = Tile{X: x, Y: y}
		m.users[index] = ent
	}

	if m.localUserID > 0 && index == m.localUserID && m.onPosition != nil {
		m.onPosition(x, y, direction)
	}
}

func (m *Manager) GetEntity(index int) (Entity, bool) {
	ent, ok := m.users[index]
	return ent, ok
}

func (m *Manager) GetAll() map[int]Entity {
	result := make(map[int]Entity, len(m.users))
	for k, v := range m.users {
		result[k] = v
	}
	return result
}

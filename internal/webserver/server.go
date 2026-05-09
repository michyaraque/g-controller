package webserver

import (
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
	"sync"

	"g-controller/internal/navigator"

	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

type Server struct {
	ctx     context.Context
	server  *http.Server
	hub     *Hub
	port    int
	addr    string
	running bool
}

type ActionHandler interface {
	HandleMove(dir int, x, y float64)
	StopMove()
	SendLook(dir int)
	SendChat(msg string)
	RequestEnterRoom(flatId int)
	SearchNavigator(view, query string)
}

type Hub struct {
	mu              sync.RWMutex
	clients         map[*websocket.Conn]bool
	broadcast       chan []byte
	actions         ActionHandler
	engineConnected bool
	roomId          int
	hotel           string
	canEdit         bool
	canRead         bool
	userX           int
	userY           int
	userDir         int
}

type Movement struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
}

func NewHub(actions ActionHandler) *Hub {
	return &Hub{
		clients:   make(map[*websocket.Conn]bool),
		broadcast: make(chan []byte, 256),
		actions:   actions,
	}
}

func (h *Hub) Run() {
	for msg := range h.broadcast {
		h.mu.RLock()
		for client := range h.clients {
			if err := client.WriteMessage(websocket.TextMessage, msg); err != nil {
				client.Close()
				delete(h.clients, client)
			}
		}
		h.mu.RUnlock()
	}
}

func (h *Hub) AddClient(conn *websocket.Conn) {
	h.mu.Lock()
	h.clients[conn] = true
	h.mu.Unlock()

	h.broadcastStatus()
}

func (h *Hub) RemoveClient(conn *websocket.Conn) {
	h.mu.Lock()
	delete(h.clients, conn)
	h.mu.Unlock()
}

func (h *Hub) broadcastMessage(msg WebSocketMessage) {
	data, err := json.Marshal(msg)
	if err != nil {
		return
	}
	select {
	case h.broadcast <- data:
	default:
	}
}

func (h *Hub) broadcastStatus() {
	h.mu.RLock()
	defer h.mu.RUnlock()

	msg := StatusMessage{
		Type:        "status",
		Connected:   h.engineConnected,
		RoomId:      h.roomId,
		Hotel:       h.hotel,
		CanEdit:     h.canEdit,
		CanRead:     h.canRead,
		ClientCount: len(h.clients),
		UserX:       h.userX,
		UserY:       h.userY,
		UserDir:     h.userDir,
	}
	data, _ := json.Marshal(msg)
	select {
	case h.broadcast <- data:
	default:
	}
}

func (h *Hub) UpdateStatus(connected bool, roomId int, hotel string, canEdit, canRead bool, userX, userY, userDir int) {
	h.mu.Lock()
	h.engineConnected = connected
	h.roomId = roomId
	h.hotel = hotel
	h.canEdit = canEdit
	h.canRead = canRead
	h.userX = userX
	h.userY = userY
	h.userDir = userDir
	h.mu.Unlock()

	h.broadcastStatus()
}

func (h *Hub) OnGameChat(msg string) {
	h.broadcastMessage(WebSocketMessage{
		Type:    "chat",
		Message: msg,
	})
}

type WebSocketMessage struct {
	Type      string `json:"type"`
	Message   string `json:"message,omitempty"`
	Action    string `json:"action,omitempty"`
	Dir       int    `json:"dir,omitempty"`
	Direction struct {
		X float64 `json:"x"`
		Y float64 `json:"y"`
	} `json:"direction,omitempty"`
	Data    interface{} `json:"data,omitempty"`
}

type StatusMessage struct {
	Type        string `json:"type"`
	Connected   bool   `json:"connected"`
	RoomId      int    `json:"roomId"`
	Hotel       string `json:"hotel"`
	CanEdit     bool   `json:"canEdit"`
	CanRead     bool   `json:"canRead"`
	ClientCount int    `json:"clientCount"`
	UserX       int    `json:"userX"`
	UserY       int    `json:"userY"`
	UserDir     int    `json:"userDir"`
}

func (h *Hub) BroadcastNavigatorResult(result navigator.SearchResult) {
	msg := WebSocketMessage{
		Type:  "navigator",
		Data:  result,
	}
	data, err := json.Marshal(msg)
	if err != nil {
		return
	}
	select {
	case h.broadcast <- data:
	default:
	}
}

func NewServer(ctx context.Context, port int, actions ActionHandler) *Server {
	hub := NewHub(actions)

	mux := http.NewServeMux()

	mux.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			log.Printf("WebSocket upgrade error: %v", err)
			return
		}

		hub.AddClient(conn)

		go func() {
			defer func() {
				hub.RemoveClient(conn)
				conn.Close()
			}()

			for {
				_, message, err := conn.ReadMessage()
				if err != nil {
					return
				}

				var msg WebSocketMessage
				if err := json.Unmarshal(message, &msg); err != nil {
					continue
				}

				switch msg.Type {
				case "chat":
					if msg.Message != "" {
						hub.actions.SendChat(msg.Message)
					}
				case "move":
					hub.actions.HandleMove(0, msg.Direction.X, msg.Direction.Y)
				case "look":
					hub.actions.SendLook(msg.Dir)
				case "move_stop":
					hub.actions.StopMove()
				case "enter_room":
					if rm, ok := msg.Data.(float64); ok {
						hub.actions.RequestEnterRoom(int(rm))
					}
				case "navigator_search":
					if d, ok := msg.Data.(map[string]interface{}); ok {
						view, _ := d["view"].(string)
						query, _ := d["query"].(string)
						hub.actions.SearchNavigator(view, query)
					}
				}
			}
		}()
	})

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Write([]byte(mobileHTML))
	})

	mux.HandleFunc("/manifest.json", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(manifestJSON))
	})

	mux.HandleFunc("/sw.js", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/javascript")
		w.Write([]byte(`self.addEventListener("install", e => self.skipWaiting());self.addEventListener("activate", e => e.waitUntil(self.clients.claim()));self.addEventListener("fetch", e => e.respondWith(fetch(e.request).catch(() => new Response("offline"))))`))
	})

	mux.HandleFunc("/icon-192.png", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "image/png")
		w.Write(icon192)
	})

	mux.HandleFunc("/icon-512.png", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "image/png")
		w.Write(icon512)
	})

	s := &Server{
		ctx:  ctx,
		hub:  hub,
		port: port,
		server: &http.Server{
			Handler: mux,
		},
	}

	return s
}

func (s *Server) Start() error {
	listener, err := net.Listen("tcp", fmt.Sprintf(":%d", s.port))
	if err != nil {
		log.Printf("Port %d in use, falling back to random port", s.port)
		listener, err = net.Listen("tcp", ":0")
		if err != nil {
			return fmt.Errorf("failed to start mobile web server: %w", err)
		}
	}

	addr := listener.Addr().(*net.TCPAddr)
	s.port = addr.Port

	localIP := getLocalIP()
	s.addr = fmt.Sprintf("http://%s:%d", localIP, addr.Port)

	go func() {
		log.Printf("Mobile web server running at %s", s.addr)
		if err := s.server.Serve(listener); err != http.ErrServerClosed {
			log.Printf("Mobile web server error: %v", err)
		}
	}()

	go s.hub.Run()
	s.running = true
	return nil
}

func (s *Server) Stop() error {
	s.running = false
	if s.server != nil {
		return s.server.Close()
	}
	return nil
}

func (s *Server) IsRunning() bool {
	return s.running
}

func (s *Server) GetURL() string {
	return s.addr
}

func (s *Server) GetHub() *Hub {
	return s.hub
}

func getLocalIP() string {
	addrs, err := net.InterfaceAddrs()
	if err == nil {
		for _, a := range addrs {
			if ipnet, ok := a.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
				ip := ipnet.IP.To4()
				if ip == nil {
					continue
				}
				if ip[0] == 192 && ip[1] == 168 {
					return ip.String()
				}
			}
		}
		for _, a := range addrs {
			if ipnet, ok := a.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
				ip := ipnet.IP.To4()
				if ip == nil {
					continue
				}
				if ip[0] == 10 || (ip[0] == 172 && ip[1] >= 16 && ip[1] <= 31) {
					return ip.String()
				}
			}
		}
	}
	return "127.0.0.1"
}

//go:embed web.html
var mobileHTML string

//go:embed manifest.json
var manifestJSON string

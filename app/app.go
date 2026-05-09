package app

import (
	"context"
	"encoding/base64"
	"sync"

	"wiredscriptengine/internal/logger"
	"wiredscriptengine/internal/webserver"

	"github.com/skip2/go-qrcode"
	wailsRuntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

const MobileServerPort = 8080

type App struct {
	ctx           context.Context
	engineManager *EngineManager
	webServer     *webserver.Server
	webServerMu   sync.RWMutex
}

func NewApp() *App {
	loadConfig()
	return &App{}
}

func (a *App) Startup(ctx context.Context) {
	a.ctx = ctx
	logger.Info("App started")

	a.engineManager = NewEngineManager(ctx)
	a.webServer = webserver.NewServer(ctx, MobileServerPort, a.engineManager)
	a.engineManager.SetHub(a.webServer.GetHub())

	go a.engineManager.Start()

	if IsMobileServerEnabled() {
		a.startMobileServer()
	}
}

func (a *App) DomReady(ctx context.Context) {
	wailsRuntime.EventsOn(ctx, "log", func(optionalData ...interface{}) {
		logger.Info("Frontend log: %v", optionalData)
	})
}

func (a *App) startMobileServer() {
	if err := a.webServer.Start(); err != nil {
		logger.Error("Failed to start mobile web server: %v", err)
	} else {
		logger.Info("Mobile web server URL: %s", a.webServer.GetURL())
		wailsRuntime.EventsEmit(a.ctx, "mobile-server:url", a.webServer.GetURL())
	}
}

func (a *App) Shutdown(ctx context.Context) {
	logger.Info("Shutting down...")
	a.webServerMu.Lock()
	if a.webServer != nil {
		a.webServer.Stop()
	}
	a.webServerMu.Unlock()
	if a.engineManager != nil {
		a.engineManager.Stop()
	}
}

func (a *App) ToggleWebServer() string {
	a.webServerMu.RLock()
	running := a.webServer.IsRunning()
	a.webServerMu.RUnlock()

	if running {
		a.webServer.Stop()
		wailsRuntime.EventsEmit(a.ctx, "mobile-server:stopped")
		return ""
	}

	a.startMobileServer()
	url := a.webServer.GetURL()
	wailsRuntime.EventsEmit(a.ctx, "mobile-server:url", url)
	return url
}

func (a *App) IsWebServerRunning() bool {
	a.webServerMu.RLock()
	defer a.webServerMu.RUnlock()
	return a.webServer != nil && a.webServer.IsRunning()
}

func (a *App) GetWebServerURL() string {
	a.webServerMu.RLock()
	defer a.webServerMu.RUnlock()
	if a.webServer == nil || !a.webServer.IsRunning() {
		return ""
	}
	return a.webServer.GetURL()
}

func (a *App) GenerateQR(url string) string {
	if url == "" {
		return ""
	}
	qr, err := qrcode.Encode(url, qrcode.Medium, 256)
	if err != nil {
		logger.Error("QR encode error: %v", err)
		return ""
	}
	return "data:image/png;base64," + base64.StdEncoding.EncodeToString(qr)
}

func (a *App) GetEngineStatus() EngineStatus {
	if a.engineManager == nil {
		return EngineStatus{Connected: false, RoomID: 0}
	}
	return a.engineManager.GetStatus()
}

func (a *App) GetUserPosition() UserPosition {
	if a.engineManager == nil {
		return UserPosition{}
	}
	return a.engineManager.GetUserPosition()
}

func (a *App) GetAppInfo() AppInfo {
	return appInfo
}

type EngineStatus struct {
	Connected bool   `json:"connected"`
	RoomID    int    `json:"roomId"`
	Hotel     string `json:"hotel"`
	CanEdit   bool   `json:"canEdit"`
	CanRead   bool   `json:"canRead"`
}

type UserPosition struct {
	ID  int `json:"id"`
	X   int `json:"x"`
	Y   int `json:"y"`
	Dir int `json:"dir"`
}

type AppInfo struct {
	Name        string `json:"name"`
	Version     string `json:"version"`
	Description string `json:"description"`
	Author      string `json:"author"`
}

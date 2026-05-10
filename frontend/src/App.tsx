import { useState, useEffect } from 'react'
import TitleBar from '@/components/TitleBar'
import Console from '@/components/Console'
import { GetEngineStatus, GetUserPosition, ToggleWebServer, IsWebServerRunning, GetWebServerURL, GenerateQR, StartMobileServer } from '@wails/go/app/App'

type EngineStatus = {
  connected: boolean
  roomId: number
  hotel: string
  canRead: boolean
  canEdit: boolean
}

type UserPosition = {
  id: number
  x: number
  y: number
  dir: number
}

function App() {
  const [engineStatus, setEngineStatus] = useState<EngineStatus>({
    connected: false,
    roomId: 0,
    hotel: '',
    canRead: false,
    canEdit: false
  })
  const [userPos, setUserPos] = useState<UserPosition>({ id: 0, x: 0, y: 0, dir: 0 })
  const [logs, setLogs] = useState<string[]>([])
  const [mobileURL, setMobileURL] = useState<string>('')
  const [qrData, setQrData] = useState<string>('')
  const [webServerRunning, setWebServerRunning] = useState(false)
  const [autoStart, setAutoStart] = useState(() => {
    return localStorage.getItem('autoStartMobileServer') === 'true'
  })

  useEffect(() => {
    checkServerStatus()
    updateEngineStatus()

    const interval = setInterval(() => {
      updateEngineStatus()
      checkServerStatus()
    }, 2000)

    window.runtime?.EventsOn('engine:activated', () => {
      if (autoStart) {
        StartMobileServer().then(url => {
          if (url) {
            setMobileURL(url)
            setWebServerRunning(true)
            generateQR(url)
          }
        }).catch(() => {})
      }
    })
    window.runtime?.EventsOn('engine:connected', () => updateEngineStatus())
    window.runtime?.EventsOn('engine:disconnected', () => updateEngineStatus())
    window.runtime?.EventsOn('engine:room-detected', () => updateEngineStatus())
    window.runtime?.EventsOn('game:chat', (msg: unknown) => {
      addLog(`Chat: ${String(msg)}`)
    })
    window.runtime?.EventsOn('mobile-server:url', (url: unknown) => {
      const u = String(url)
      setMobileURL(u)
      setWebServerRunning(true)
      generateQR(u)
    })
    window.runtime?.EventsOn('mobile-server:stopped', () => {
      setMobileURL('')
      setQrData('')
      setWebServerRunning(false)
    })

    return () => {
      clearInterval(interval)
      window.runtime?.EventsOff('engine:activated')
      window.runtime?.EventsOff('engine:connected')
      window.runtime?.EventsOff('engine:disconnected')
      window.runtime?.EventsOff('engine:room-detected')
      window.runtime?.EventsOff('game:chat')
      window.runtime?.EventsOff('mobile-server:url')
      window.runtime?.EventsOff('mobile-server:stopped')
    }
  }, [])

  const checkServerStatus = async () => {
    try {
      const running = await IsWebServerRunning()
      setWebServerRunning(running)
      if (running) {
        const url = await GetWebServerURL()
        if (url && url !== mobileURL) {
          setMobileURL(url)
          generateQR(url)
        }
      }
    } catch { }
  }

  const updateEngineStatus = async () => {
    try {
      const [status, pos] = await Promise.all([GetEngineStatus(), GetUserPosition()])
      setEngineStatus(status)
      setUserPos(pos)
    } catch { }
  }

  const addLog = (message: string) => {
    const ts = new Date().toLocaleTimeString()
    setLogs(prev => [...prev, `[${ts}] ${message}`])
  }

  const generateQR = async (url: string) => {
    try {
      const data = await GenerateQR(url)
      setQrData(data)
    } catch { }
  }

  const handleToggleServer = async () => {
    try {
      const url = await ToggleWebServer()
      if (url) {
        setMobileURL(url)
        await generateQR(url)
        addLog(`Mobile server: ${url}`)
      } else {
        setMobileURL('')
        setQrData('')
        addLog('Mobile server stopped')
      }
    } catch { }
  }

  const openMobileURL = () => {
    if (mobileURL) window.open(mobileURL, '_blank')
  }

  const copyURL = () => {
    if (mobileURL) {
      navigator.clipboard.writeText(mobileURL)
      addLog('URL copied')
    }
  }

  return (
    <div className="flex h-screen flex-col overflow-hidden bg-bg-primary text-text-primary">
      <TitleBar />

      <div className="flex flex-1 overflow-hidden">
        <div className="flex flex-1 flex-col gap-3 p-4 overflow-y-auto">
          <div className="rounded-lg border border-border bg-bg-secondary p-4">
            <div className="flex items-center gap-4 text-sm">
              <div className="flex items-center gap-2">
                <span className={`h-2 w-2 rounded-full ${engineStatus.connected ? 'bg-green-400' : 'bg-red-400'}`} />
                <span className={engineStatus.connected ? 'text-green-400' : 'text-red-400'}>
                  {engineStatus.connected ? 'Connected' : 'Disconnected'}
                </span>
              </div>
              {engineStatus.hotel && (
                <>
                  <span className="text-text-secondary">|</span>
                  <span>{engineStatus.hotel.toUpperCase()}</span>
                </>
              )}
              {engineStatus.roomId && (
                <>
                  <span className="text-text-secondary">|</span>
                  <span>Room {engineStatus.roomId}</span>
                </>
              )}
              {engineStatus.connected && (
                <>
                  <span className="text-text-secondary">|</span>
                  <span className="font-mono">({userPos.x}, {userPos.y})</span>
                </>
              )}
            </div>
          </div>

          <div className="rounded-lg border border-border bg-bg-secondary p-4">
            <div className="flex items-start gap-4">
              <div className="flex-1">
                <div className="flex items-center justify-between mb-2">
                  <span className="text-xs font-semibold uppercase tracking-wider text-text-secondary">Mobile Controller</span>
                  <button
                    onClick={handleToggleServer}
                    className={`relative h-5 w-9 rounded-full transition-colors ${webServerRunning ? 'bg-green-500' : 'bg-gray-600'}`}
                  >
                    <span
                      className={`absolute top-0.5 h-4 w-4 rounded-full bg-white transition-transform ${
                        webServerRunning ? 'left-4' : 'left-0.5'
                      }`}
                    />
                  </button>
                </div>
                <label className="flex items-center gap-2 mb-3 cursor-pointer">
                  <input
                    type="checkbox"
                    checked={autoStart}
                    onChange={(e) => {
                      setAutoStart(e.target.checked)
                      localStorage.setItem('autoStartMobileServer', String(e.target.checked))
                    }}
                    className="w-3 h-3 accent-green-500"
                  />
                  <span className="text-xs text-text-secondary">Auto-start on startup</span>
                </label>

                {webServerRunning && mobileURL ? (
                  <div className="flex items-start gap-3">
                    {qrData && (
                      <div className="shrink-0 rounded border border-border bg-white p-1.5">
                        <img src={qrData} alt="QR" className="h-24 w-24" />
                      </div>
                    )}
                    <div className="flex flex-col gap-2 min-w-0">
                      <div className="flex items-center gap-2">
                        <span className="font-mono text-sm truncate">{mobileURL}</span>
                        <button
                          onClick={copyURL}
                          className="shrink-0 text-xs text-text-secondary hover:text-text-primary transition-colors"
                        >
                          COPY
                        </button>
                      </div>
                      <button
                        onClick={openMobileURL}
                        className="rounded bg-primary-500 px-3 py-1.5 text-xs font-semibold text-white transition-colors hover:bg-primary-600 w-fit"
                      >
                        OPEN IN BROWSER
                      </button>
                    </div>
                  </div>
                ) : (
                  <p className="text-xs text-text-secondary">Toggle to start the mobile web server</p>
                )}
              </div>
            </div>
          </div>
        </div>
        <Console logs={logs} onClear={() => setLogs([])} />
      </div>
    </div>
  )
}

export default App

import { useEffect, useRef } from 'react'
import { Terminal, Trash2 } from 'lucide-react'

interface ConsoleProps {
  logs: string[]
  onClear: () => void
}

export default function Console({ logs, onClear }: ConsoleProps) {
  const logsEndRef = useRef<HTMLDivElement>(null)

  useEffect(() => {
    logsEndRef.current?.scrollIntoView({ behavior: 'smooth' })
  }, [logs])

  return (
    <div className="flex h-full w-72 flex-col border-l border-border bg-bg-secondary">
      <div className="flex items-center justify-between border-b border-border px-4 py-3">
        <div className="flex items-center gap-2">
          <Terminal size={16} />
          <h3 className="text-sm font-semibold">Console</h3>
        </div>
        <button
          onClick={onClear}
          className="rounded p-1.5 transition-colors hover:bg-tertiary"
          title="Clear logs"
        >
          <Trash2 size={14} />
        </button>
      </div>

      <div className="flex-1 overflow-y-auto p-3 font-mono text-xs">
        {logs.length === 0 ? (
          <div className="flex flex-col items-center justify-center py-12 text-center text-text-secondary">
            <Terminal size={32} className="mb-3 opacity-30" />
            <p className="text-xs italic">No logs yet...</p>
          </div>
        ) : (
          logs.map((log, i) => (
            <div
              key={i}
              className="mb-1.5 rounded px-2 py-1 text-text-primary"
            >
              {log}
            </div>
          ))
        )}
        <div ref={logsEndRef} />
      </div>
    </div>
  )
}

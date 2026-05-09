import { Minus, Square, X } from 'lucide-react'
import { WindowMinimise, WindowToggleMaximise, Quit } from '@wails/runtime/runtime'

export default function TitleBar() {
  return (
    <header
      className="flex h-8 select-none items-center justify-between border-b border-border bg-bg-secondary px-3"
      style={{ widows: 1 } as React.CSSProperties}
      onDoubleClick={WindowToggleMaximise}
    >
      <div className="flex items-center gap-2">
        <span className="text-xs font-semibold text-text-primary">G-Controller</span>
      </div>

      <div className="flex items-center gap-1" style={{ widows: 0 } as React.CSSProperties}>
        <button
          onPointerDown={(e) => e.stopPropagation()}
          onClick={WindowMinimise}
          className="flex h-7 w-9 items-center justify-center rounded transition-colors hover:bg-tertiary"
          title="Minimize"
        >
          <Minus size={14} className="text-text-secondary" />
        </button>

        <button
          onPointerDown={(e) => e.stopPropagation()}
          onClick={WindowToggleMaximise}
          className="flex h-7 w-9 items-center justify-center rounded transition-colors hover:bg-tertiary"
          title="Maximize"
        >
          <Square size={12} className="text-text-secondary" />
        </button>

        <button
          onPointerDown={(e) => e.stopPropagation()}
          onClick={Quit}
          className="flex h-7 w-9 items-center justify-center rounded transition-colors hover:bg-red-600"
          title="Close"
        >
          <X size={14} className="text-text-secondary hover:text-white" />
        </button>
      </div>
    </header>
  )
}

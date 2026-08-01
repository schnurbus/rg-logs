import type { CSSProperties, ReactNode } from 'react'
import { classBarStyle, type PlayerClass } from '../lib/classes'

type MeterBarProps = {
  /** 0–1 relative to the top row (or spell max). */
  ratio: number
  playerClass?: PlayerClass | null
  children: ReactNode
  /** Slightly dimmer bar for nested pet rows */
  muted?: boolean
  className?: string
  /** Rich hover details (e.g. spell hit breakdown). */
  tooltip?: ReactNode
}

/**
 * Warcraft-Logs-style horizontal contribution bar behind the cell content.
 * Width is relative to the best value in the current table (ratio = value / max).
 */
export function MeterBar({
  ratio,
  playerClass,
  children,
  muted,
  className,
  tooltip,
}: MeterBarProps) {
  const bar = classBarStyle(playerClass, ratio)
  const fillStyle: CSSProperties = {
    ...bar,
    opacity: muted ? 0.28 : 0.38,
  }

  return (
    <div
      className={[
        'relative min-w-0',
        tooltip ? 'group/meter overflow-visible' : 'overflow-hidden',
        className,
      ]
        .filter(Boolean)
        .join(' ')}
    >
      <div
        aria-hidden
        className="pointer-events-none absolute inset-y-0 left-0 rounded-sm transition-[width] duration-300 ease-out"
        style={fillStyle}
      />
      <div className="relative z-10 px-2 py-0.5">{children}</div>
      {tooltip ? (
        <div
          role="tooltip"
          className="pointer-events-none absolute bottom-full left-0 z-30 mb-1 hidden min-w-[12rem] max-w-xs rounded border border-border bg-surface-overlay px-2.5 py-2 text-xs text-text shadow-lg group-hover/meter:block"
        >
          {tooltip}
        </div>
      ) : null}
    </div>
  )
}

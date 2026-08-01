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
}: MeterBarProps) {
  const bar = classBarStyle(playerClass, ratio)
  const fillStyle: CSSProperties = {
    ...bar,
    opacity: muted ? 0.28 : 0.38,
  }

  return (
    <div
      className={['relative min-w-0 overflow-hidden', className]
        .filter(Boolean)
        .join(' ')}
    >
      <div
        aria-hidden
        className="pointer-events-none absolute inset-y-0 left-0 rounded-sm transition-[width] duration-300 ease-out"
        style={fillStyle}
      />
      <div className="relative z-10 px-2 py-0.5">{children}</div>
    </div>
  )
}

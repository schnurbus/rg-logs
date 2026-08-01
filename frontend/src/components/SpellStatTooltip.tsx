import { formatNumber, formatPercent } from '../lib/format'
import type { SpellStat } from '../types/api'

function attempts(s: SpellStat): number {
  return s.hits + s.misses
}

function avg(total: number, hits: number): number | null {
  if (hits <= 0) return null
  return total / hits
}

function SpectrumLine({
  label,
  hits,
  total,
  min,
  max,
  hitShare,
}: {
  label: string
  hits: number
  total: number
  min: number
  max: number
  /** hits / spell.hits for rate display */
  hitShare: number
}) {
  if (hits <= 0) return null
  const mean = avg(total, hits)
  return (
    <>
      <dt>{label}</dt>
      <dd className="text-right">
        <span className="text-text">
          {formatNumber(min)} – {formatNumber(max)}
        </span>
        <span className="ml-1.5">Ø {formatNumber(mean)}</span>
        <div className="opacity-80">
          {formatNumber(hits)}× ({formatPercent(hitShare)})
        </div>
      </dd>
    </>
  )
}

/** Compact hover panel for spell meter bars. */
export function SpellStatTooltip({ spell }: { spell: SpellStat }) {
  const totalAttempts = attempts(spell)
  const missRate = totalAttempts > 0 ? spell.misses / totalAttempts : 0
  const hasHits = spell.hits > 0
  const denom = spell.hits > 0 ? spell.hits : 1

  return (
    <div className="space-y-1.5 font-sans">
      <div className="font-medium text-text">
        {spell.spellName}
        {spell.pet ? (
          <span className="ml-1.5 text-xs font-normal text-text-muted">Pet</span>
        ) : null}
      </div>
      {hasHits ? (
        <dl className="grid grid-cols-[auto_1fr] gap-x-3 gap-y-1 font-mono text-text-muted">
          <SpectrumLine
            label="Normal"
            hits={spell.normalHits}
            total={spell.normalTotal}
            min={spell.normalMin}
            max={spell.normalMax}
            hitShare={spell.normalHits / denom}
          />
          <SpectrumLine
            label="Crit"
            hits={spell.crits}
            total={spell.critTotal}
            min={spell.critMin}
            max={spell.critMax}
            hitShare={spell.crits / denom}
          />
          <SpectrumLine
            label="Glancing"
            hits={spell.glancing}
            total={spell.glancingTotal}
            min={spell.glancingMin}
            max={spell.glancingMax}
            hitShare={spell.glancing / denom}
          />
        </dl>
      ) : (
        <div className="text-text-muted">Keine Treffer</div>
      )}
      {spell.misses > 0 || spell.metric === 'damage' ? (
        <div className="font-mono text-text-muted">
          Verfehlt: {formatNumber(spell.misses)}
          {spell.misses > 0 && totalAttempts > 0 ? (
            <span className="ml-1.5 opacity-80">({formatPercent(missRate)})</span>
          ) : null}
        </div>
      ) : null}
    </div>
  )
}

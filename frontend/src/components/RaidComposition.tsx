import { asPlayerClass, classTextStyle } from '../lib/classes'
import { formatNumber } from '../lib/format'
import { risingGodsProfileURL } from '../lib/risingGods'
import { asPlayerSpec, roleFromSpec, type RaidRole } from '../lib/specs'
import type { Participant } from '../types/api'
import { SpecIcon } from './SpecIcon'

const ROLE_ROWS: { role: RaidRole; label: string }[] = [
  { role: 'tank', label: 'Tanks' },
  { role: 'dps', label: 'DPS' },
  { role: 'healer', label: 'Heiler' },
]

type RaidCompositionProps = {
  players: Participant[]
}

function groupByRole(players: Participant[]): Record<RaidRole, Participant[]> {
  const groups: Record<RaidRole, Participant[]> = {
    tank: [],
    dps: [],
    healer: [],
  }
  for (const p of players) {
    if (!p.isPlayer) continue
    const role = roleFromSpec(asPlayerSpec(p.spec))
    groups[role].push(p)
  }
  for (const role of Object.keys(groups) as RaidRole[]) {
    groups[role].sort((a, b) =>
      a.name.localeCompare(b.name, undefined, { sensitivity: 'base' }),
    )
  }
  return groups
}

function averageGearScore(players: Participant[]): number | undefined {
  const scored = players.filter((p) => p.isPlayer && p.gearScore != null)
  if (scored.length === 0) return undefined
  const sum = scored.reduce((s, p) => s + (p.gearScore ?? 0), 0)
  return sum / scored.length
}

export function RaidComposition({ players }: RaidCompositionProps) {
  const raid = players.filter((p) => p.isPlayer)
  const byRole = groupByRole(raid)
  const avgGs = averageGearScore(raid)

  return (
    <section className="overflow-hidden rounded border border-border bg-surface-raised">
      <div className="flex flex-wrap items-center justify-between gap-2 border-b border-border bg-surface-overlay px-3 py-2">
        <h2 className="text-sm font-semibold">Raidzusammenstellung</h2>
        {avgGs != null ? (
          <p className="text-xs text-text-muted">
            Ø GearScore: {formatNumber(Math.round(avgGs))}
          </p>
        ) : null}
      </div>

      <div className="divide-y divide-border-subtle">
        {ROLE_ROWS.map(({ role, label }) => (
          <div
            key={role}
            className="flex flex-wrap items-start gap-x-4 gap-y-2 px-3 py-2.5"
          >
            <div className="w-16 shrink-0 pt-0.5 text-xs font-medium uppercase tracking-wide text-text-muted">
              {label}
            </div>
            <div className="flex min-w-0 flex-1 flex-wrap gap-x-4 gap-y-1.5">
              {byRole[role].length === 0 ? (
                <span className="text-sm text-text-muted">—</span>
              ) : (
                byRole[role].map((p) => {
                  const cls = asPlayerClass(p.class)
                  return (
                    <a
                      key={p.actorId || p.guid}
                      href={risingGodsProfileURL(p.name)}
                      target="_blank"
                      rel="noopener noreferrer"
                      title="Rising Gods Profil"
                      className="inline-flex items-center gap-1.5 text-sm font-medium hover:underline"
                      style={classTextStyle(cls)}
                    >
                      <SpecIcon
                        spec={p.spec}
                        playerClass={p.class}
                        name={p.name}
                      />
                      {p.name}
                    </a>
                  )
                })
              )}
            </div>
          </div>
        ))}
      </div>
    </section>
  )
}

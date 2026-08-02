import { useMemo } from 'react'
import {
  CartesianGrid,
  Legend,
  Line,
  LineChart,
  ResponsiveContainer,
  Tooltip,
  XAxis,
  YAxis,
} from 'recharts'
import { asPlayerClass, classColorHex } from '../lib/classes'
import { formatDuration, formatNumber } from '../lib/format'
import type { TimelinePlayers } from '../types/api'

type FightPlayerTimelineChartProps = {
  data: TimelinePlayers
  title?: string
}

export function FightPlayerTimelineChart({
  data,
  title = 'Timeline',
}: FightPlayerTimelineChartProps) {
  const chartData = useMemo(() => {
    if (data.series.length === 0) return []
    const n = data.series[0]?.points.length ?? 0
    const rows: Record<string, number>[] = []
    for (let i = 0; i < n; i++) {
      const row: Record<string, number> = {
        t: data.series[0].points[i]?.t ?? i * data.bucketMs,
      }
      for (const s of data.series) {
        row[s.actorId] = s.points[i]?.amount ?? 0
      }
      rows.push(row)
    }
    return rows
  }, [data])

  if (data.series.length === 0 || chartData.length === 0) {
    return (
      <p className="rounded border border-border bg-surface-raised px-4 py-8 text-center text-sm text-text-muted">
        Keine Timeline-Daten. Log ggf. erneut hochladen.
      </p>
    )
  }

  return (
    <div className="rounded border border-border bg-surface-raised p-3">
      <h3 className="mb-2 text-sm font-semibold">{title}</h3>
      <div className="h-72 w-full">
        <ResponsiveContainer width="100%" height="100%">
          <LineChart data={chartData} margin={{ top: 8, right: 12, left: 0, bottom: 0 }}>
            <CartesianGrid stroke="var(--color-border-subtle)" strokeDasharray="3 3" />
            <XAxis
              dataKey="t"
              tickFormatter={(v: number) => formatDuration(v)}
              stroke="var(--color-text-muted)"
              fontSize={11}
            />
            <YAxis
              tickFormatter={(v: number) => formatNumber(v)}
              stroke="var(--color-text-muted)"
              fontSize={11}
              width={56}
            />
            <Tooltip
              contentStyle={{
                background: 'var(--color-surface-overlay)',
                border: '1px solid var(--color-border)',
                borderRadius: 6,
                fontSize: 12,
              }}
              labelFormatter={(v) => formatDuration(Number(v))}
              formatter={(value, name) => {
                const series = data.series.find((s) => s.actorId === name)
                return [formatNumber(Number(value)), series?.name ?? String(name)]
              }}
            />
            <Legend
              formatter={(value) => {
                const series = data.series.find((s) => s.actorId === value)
                return series?.name ?? value
              }}
            />
            {data.series.map((s) => (
              <Line
                key={s.actorId}
                type="monotone"
                dataKey={s.actorId}
                name={s.actorId}
                stroke={classColorHex(asPlayerClass(s.class))}
                dot={false}
                strokeWidth={1.5}
                isAnimationActive={false}
              />
            ))}
          </LineChart>
        </ResponsiveContainer>
      </div>
    </div>
  )
}

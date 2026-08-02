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
import { formatDuration, formatNumber } from '../lib/format'
import type { TimelineSummary } from '../types/api'

type FightTimelineChartProps = {
  data: TimelineSummary
}

export function FightTimelineChart({ data }: FightTimelineChartProps) {
  if (data.points.length === 0) {
    return (
      <p className="rounded border border-border bg-surface-raised px-4 py-8 text-center text-sm text-text-muted">
        Keine Timeline-Daten. Log ggf. erneut hochladen.
      </p>
    )
  }

  return (
    <div className="rounded border border-border bg-surface-raised p-3">
      <h3 className="mb-2 text-sm font-semibold">Performance</h3>
      <div className="h-64 w-full">
        <ResponsiveContainer width="100%" height="100%">
          <LineChart data={data.points} margin={{ top: 8, right: 12, left: 0, bottom: 0 }}>
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
              formatter={(value, name) => [
                formatNumber(Number(value)),
                String(name),
              ]}
            />
            <Legend />
            <Line
              type="monotone"
              dataKey="damage"
              name="Schaden"
              stroke="#3b82f6"
              dot={false}
              strokeWidth={1.5}
              isAnimationActive={false}
            />
            <Line
              type="monotone"
              dataKey="healing"
              name="Heilung"
              stroke="#22c55e"
              dot={false}
              strokeWidth={1.5}
              isAnimationActive={false}
            />
            <Line
              type="monotone"
              dataKey="taken"
              name="Erlitten"
              stroke="#ef4444"
              dot={false}
              strokeWidth={1.5}
              isAnimationActive={false}
            />
          </LineChart>
        </ResponsiveContainer>
      </div>
    </div>
  )
}

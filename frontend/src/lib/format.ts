const numberFmt = new Intl.NumberFormat('de-DE', { maximumFractionDigits: 0 })
const rateFmt = new Intl.NumberFormat('de-DE', {
  maximumFractionDigits: 1,
  minimumFractionDigits: 0,
})
const percentFmt = new Intl.NumberFormat('de-DE', {
  style: 'percent',
  maximumFractionDigits: 1,
  minimumFractionDigits: 1,
})

export function formatNumber(n: number | undefined | null): string {
  if (n == null || !Number.isFinite(n)) return '—'
  return numberFmt.format(Math.round(n))
}

export function formatRate(n: number | undefined | null): string {
  if (n == null || !Number.isFinite(n)) return '—'
  return rateFmt.format(n)
}

export function formatPercent(ratio: number): string {
  if (!Number.isFinite(ratio)) return '—'
  return percentFmt.format(ratio)
}

export function formatBytes(bytes: number): string {
  if (!Number.isFinite(bytes) || bytes < 0) return '—'
  if (bytes < 1024) return `${bytes} B`
  if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`
  return `${(bytes / (1024 * 1024)).toFixed(2)} MB`
}

export function formatDuration(ms: number): string {
  if (!Number.isFinite(ms) || ms < 0) return '—'
  const totalSec = Math.floor(ms / 1000)
  const h = Math.floor(totalSec / 3600)
  const m = Math.floor((totalSec % 3600) / 60)
  const s = totalSec % 60
  if (h > 0) {
    return `${h}:${String(m).padStart(2, '0')}:${String(s).padStart(2, '0')}`
  }
  return `${m}:${String(s).padStart(2, '0')}`
}

export function formatDateTime(iso: string | null | undefined): string {
  if (!iso) return '—'
  const d = new Date(iso)
  if (Number.isNaN(d.getTime())) return iso
  return d.toLocaleString('de-DE')
}

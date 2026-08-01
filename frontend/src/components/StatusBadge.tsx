import type { UploadStatus } from '../types/api'

const STYLES: Record<UploadStatus, string> = {
  pending:
    'bg-surface-overlay text-text-muted ring-1 ring-border',
  processing:
    'bg-amber-950/50 text-warning ring-1 ring-warning/40',
  ready:
    'bg-emerald-950/40 text-success ring-1 ring-success/40',
  failed:
    'bg-red-950/40 text-danger ring-1 ring-danger/40',
}

const LABELS: Record<UploadStatus, string> = {
  pending: 'Pending',
  processing: 'Processing',
  ready: 'Ready',
  failed: 'Failed',
}

export function StatusBadge({ status }: { status: UploadStatus }) {
  return (
    <span
      className={`inline-flex items-center rounded px-2 py-0.5 text-xs font-medium tracking-wide ${STYLES[status]}`}
    >
      {LABELS[status]}
    </span>
  )
}

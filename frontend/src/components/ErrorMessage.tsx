export function Loading({ label = 'Laden…' }: { label?: string }) {
  return (
    <div className="flex items-center gap-3 py-8 text-text-muted" role="status">
      <span
        className="inline-block h-4 w-4 animate-spin rounded-full border-2 border-border border-t-accent"
        aria-hidden
      />
      <span className="text-sm">{label}</span>
    </div>
  )
}

export function ErrorMessage({
  message,
  onRetry,
}: {
  message: string
  onRetry?: () => void
}) {
  return (
    <div
      className="rounded border border-danger/40 bg-red-950/30 px-4 py-3 text-sm text-danger"
      role="alert"
    >
      <p>{message}</p>
      {onRetry ? (
        <button
          type="button"
          onClick={onRetry}
          className="mt-2 text-accent underline hover:text-accent-hover"
        >
          Erneut versuchen
        </button>
      ) : null}
    </div>
  )
}

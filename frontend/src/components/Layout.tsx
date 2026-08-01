import { Link, NavLink, Outlet } from 'react-router-dom'

const navClass = ({ isActive }: { isActive: boolean }) =>
  [
    'px-3 py-1.5 text-sm rounded transition-colors',
    isActive
      ? 'bg-surface-overlay text-text'
      : 'text-text-muted hover:text-text hover:bg-surface-overlay/60',
  ].join(' ')

export function Layout() {
  return (
    <div className="min-h-screen flex flex-col">
      <header className="border-b border-border bg-surface-raised">
        <div className="mx-auto flex max-w-6xl items-center justify-between gap-4 px-4 py-3">
          <Link to="/" className="text-text no-underline hover:no-underline">
            <span className="text-base font-semibold tracking-tight">
              rg-logs
            </span>
            <span className="ml-2 text-xs text-text-muted">
              Combat Log Analyzer
            </span>
          </Link>
          <nav className="flex items-center gap-1">
            <NavLink to="/" end className={navClass}>
              Upload
            </NavLink>
            <NavLink to="/uploads" className={navClass}>
              Uploads
            </NavLink>
          </nav>
        </div>
      </header>
      <main className="mx-auto w-full max-w-6xl flex-1 px-4 py-6">
        <Outlet />
      </main>
    </div>
  )
}

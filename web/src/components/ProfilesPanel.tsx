import type { ScanProfileInfo } from '../types'

interface Props {
  profiles: ScanProfileInfo[]
  loading: boolean
  error: string
  onRunProfile: (name: string) => void
}

export function ProfilesPanel({ profiles, loading, error, onRunProfile }: Props) {
  return (
    <section className="rounded-xl border border-border bg-surface p-5">
      <h2 className="mb-4 text-lg font-semibold">Scan Profiles</h2>
      {loading && <p className="text-sm text-muted">Loading profiles…</p>}
      {error && <p className="text-sm text-danger">{error}</p>}
      {!loading && !error && profiles.length === 0 && (
        <p className="text-sm text-muted">
          No scan profiles configured. Add profiles to your YAML config and start the server with{' '}
          <code className="rounded bg-surface-2 px-1">--config</code>.
        </p>
      )}
      <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
        {profiles.map((p) => (
          <div
            key={p.name}
            className="rounded-xl border border-border bg-surface-2 p-4"
          >
            <h3 className="font-medium">{p.name}</h3>
            <p className="mt-1 text-sm text-muted">
              Provider: <span className="text-text">{p.provider}</span>
            </p>
            <p className="mt-1 truncate text-sm text-muted" title={p.state_source}>
              {p.state_source}
            </p>
            {p.schedule && (
              <p className="mt-2 text-xs text-warning">⏱ {p.schedule}</p>
            )}
            <button
              onClick={() => onRunProfile(p.name)}
              className="mt-3 rounded-lg bg-primary px-3 py-1.5 text-sm font-medium text-white hover:bg-primary-hover"
            >
              Run Now
            </button>
          </div>
        ))}
      </div>
    </section>
  )
}

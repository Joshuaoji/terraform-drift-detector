import type { ScanSummary } from '../types'

interface Props {
  scans: ScanSummary[]
  loading: boolean
  error: string
  onView: (id: string) => void
}

const statusStyles: Record<string, string> = {
  pending: 'bg-gray-700 text-gray-300',
  running: 'bg-blue-900 text-blue-300',
  completed: 'bg-green-900 text-green-300',
  failed: 'bg-red-950 text-red-300',
}

export function ScanHistory({ scans, loading, error, onView }: Props) {
  return (
    <section className="rounded-xl border border-border bg-surface p-5">
      <div className="mb-4 flex items-center justify-between">
        <h2 className="text-lg font-semibold">Scan History</h2>
        <span className="rounded-full bg-surface-2 px-2.5 py-0.5 text-xs text-muted">
          {scans.length} scan{scans.length !== 1 ? 's' : ''}
        </span>
      </div>
      {loading && <p className="text-sm text-muted">Loading scans…</p>}
      {error && <p className="text-sm text-danger">{error}</p>}
      <div className="overflow-x-auto">
        <table className="w-full text-sm">
          <thead>
            <tr className="border-b border-border text-left text-xs uppercase tracking-wide text-muted">
              <th className="px-3 py-2">Status</th>
              <th className="px-3 py-2">Provider</th>
              <th className="px-3 py-2">State Source</th>
              <th className="px-3 py-2">Profile</th>
              <th className="px-3 py-2">Drifts</th>
              <th className="px-3 py-2">Created</th>
              <th className="px-3 py-2"></th>
            </tr>
          </thead>
          <tbody>
            {!loading && scans.length === 0 && (
              <tr>
                <td colSpan={7} className="px-3 py-6 text-center text-muted">
                  No scans yet. Trigger one from a profile or click New Scan.
                </td>
              </tr>
            )}
            {scans.map((s) => (
              <tr key={s.id} className="border-b border-border hover:bg-surface-2">
                <td className="px-3 py-3">
                  <span
                    className={`inline-block rounded-full px-2 py-0.5 text-xs capitalize ${statusStyles[s.status] || ''}`}
                  >
                    {s.status}
                  </span>
                </td>
                <td className="px-3 py-3">{s.provider}</td>
                <td className="max-w-[200px] truncate px-3 py-3" title={s.state_source}>
                  {s.state_source}
                </td>
                <td className="px-3 py-3">{s.profile_name || '—'}</td>
                <td className="px-3 py-3">
                  <span
                    className={
                      s.status === 'completed'
                        ? s.total_drifts > 0
                          ? 'font-semibold text-danger'
                          : 'font-semibold text-success'
                        : 'text-muted'
                    }
                  >
                    {s.status === 'completed' ? s.total_drifts : '—'}
                  </span>
                </td>
                <td className="px-3 py-3 whitespace-nowrap">{formatDate(s.created_at)}</td>
                <td className="px-3 py-3">
                  <button
                    onClick={() => onView(s.id)}
                    className="text-primary hover:underline"
                  >
                    View
                  </button>
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </section>
  )
}

function formatDate(iso: string) {
  if (!iso) return '—'
  return new Date(iso).toLocaleString()
}

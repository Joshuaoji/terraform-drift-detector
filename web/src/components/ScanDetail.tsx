import type { ScanRecord } from '../types'

interface Props {
  scan: ScanRecord | null
  loading: boolean
  error: string
  onClose: () => void
}

const statusStyles: Record<string, string> = {
  pending: 'bg-gray-700 text-gray-300',
  running: 'bg-blue-900 text-blue-300',
  completed: 'bg-green-900 text-green-300',
  failed: 'bg-red-950 text-red-300',
}

export function ScanDetail({ scan, loading, error, onClose }: Props) {
  if (!scan && !loading) return null

  return (
    <section className="rounded-xl border border-border bg-surface p-5">
      <div className="mb-4 flex items-center justify-between">
        <h2 className="text-lg font-semibold">Scan Detail</h2>
        <button
          onClick={onClose}
          className="rounded-lg bg-surface-2 px-3 py-1.5 text-sm hover:bg-border"
        >
          Close
        </button>
      </div>

      {loading && <p className="text-sm text-muted">Loading scan…</p>}
      {error && <p className="text-sm text-danger">{error}</p>}

      {scan && (
        <>
          <dl className="mb-6 grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
            <div>
              <dt className="text-xs text-muted">Scan ID</dt>
              <dd className="break-all text-sm font-medium">{scan.id}</dd>
            </div>
            <div>
              <dt className="text-xs text-muted">Status</dt>
              <dd>
                <span
                  className={`inline-block rounded-full px-2 py-0.5 text-xs capitalize ${statusStyles[scan.status] || ''}`}
                >
                  {scan.status}
                </span>
              </dd>
            </div>
            <div>
              <dt className="text-xs text-muted">Provider</dt>
              <dd className="text-sm font-medium">{scan.provider}</dd>
            </div>
            <div>
              <dt className="text-xs text-muted">State</dt>
              <dd className="break-all text-sm font-medium">{scan.state_source}</dd>
            </div>
            <div>
              <dt className="text-xs text-muted">Created</dt>
              <dd className="text-sm font-medium">{formatDate(scan.created_at)}</dd>
            </div>
            <div>
              <dt className="text-xs text-muted">Completed</dt>
              <dd className="text-sm font-medium">{formatDate(scan.completed_at)}</dd>
            </div>
          </dl>

          {scan.error && (
            <p className="mb-4 text-sm text-danger">{scan.error}</p>
          )}

          {!scan.report ? (
            <p className="text-sm text-muted">
              {scan.status === 'pending' || scan.status === 'running'
                ? 'Scan in progress…'
                : 'No report available.'}
            </p>
          ) : (
            <>
              <div className="mb-6 grid gap-4 sm:grid-cols-3 lg:grid-cols-5">
                {[
                  ['Total Drifts', scan.report.summary.total_drifts],
                  ['Missing in Cloud', scan.report.summary.missing_in_cloud],
                  ['Missing in State', scan.report.summary.missing_in_state],
                  ['Attribute', scan.report.summary.attribute_drifts],
                  ['Tag', scan.report.summary.tag_drifts],
                ].map(([label, value]) => (
                  <div
                    key={label}
                    className="rounded-xl bg-surface-2 p-4 text-center"
                  >
                    <div className="text-2xl font-bold">{value}</div>
                    <div className="mt-1 text-xs text-muted">{label}</div>
                  </div>
                ))}
              </div>

              {scan.report.drifts.length === 0 ? (
                <p className="text-sm text-muted">
                  No drift detected. Infrastructure matches Terraform state.
                </p>
              ) : (
                <div className="space-y-3">
                  {scan.report.drifts.map((d, i) => (
                    <div
                      key={`${d.resource_id}-${i}`}
                      className="rounded-xl border border-border bg-surface-2 p-4"
                    >
                      <div className="text-xs uppercase text-warning">{d.type}</div>
                      <h4 className="mt-1 font-medium">
                        {d.resource_type} — {d.resource_id}
                      </h4>
                      {d.message && (
                        <p className="mt-1 text-sm text-muted">{d.message}</p>
                      )}
                      <ul className="mt-2 space-y-1 text-sm text-muted">
                        {(d.attribute_changes || []).map((c) => (
                          <li key={c.attribute}>
                            <strong className="text-text">{c.attribute}</strong>: expected{' '}
                            {String(c.expected)} → actual {String(c.actual)}
                          </li>
                        ))}
                        {(d.tag_changes || []).map((t) => (
                          <li key={t.key}>
                            Tag <strong className="text-text">{t.key}</strong> ({t.change}):{' '}
                            {t.expected || ''} → {t.actual || ''}
                          </li>
                        ))}
                      </ul>
                    </div>
                  ))}
                </div>
              )}
            </>
          )}
        </>
      )}
    </section>
  )
}

function formatDate(iso?: string) {
  if (!iso) return '—'
  return new Date(iso).toLocaleString()
}

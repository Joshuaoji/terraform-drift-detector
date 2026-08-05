import type { CreateScanRequest, ScanProfileInfo, ScanRecord, ScanSummary } from '../types'

const API = '/api/v1'

async function fetchJSON<T>(url: string, opts: RequestInit = {}): Promise<T> {
  const res = await fetch(url, {
    headers: { 'Content-Type': 'application/json' },
    ...opts,
  })
  if (!res.ok) {
    const err = await res.json().catch(() => ({}))
    throw new Error((err as { error?: string }).error || res.statusText)
  }
  return res.json() as Promise<T>
}

export const api = {
  getProfiles: () => fetchJSON<ScanProfileInfo[]>(`${API}/profiles`),

  listScans: (limit = 50) =>
    fetchJSON<ScanSummary[]>(`${API}/scans?summary=true&limit=${limit}`),

  getScan: (id: string) => fetchJSON<ScanRecord>(`${API}/scans/${id}`),

  createScan: (body: CreateScanRequest) =>
    fetchJSON<ScanRecord>(`${API}/scans`, {
      method: 'POST',
      body: JSON.stringify(body),
    }),

  runProfile: (name: string) =>
    fetchJSON<ScanRecord>(`${API}/scans/profile/${encodeURIComponent(name)}`, {
      method: 'POST',
    }),
}

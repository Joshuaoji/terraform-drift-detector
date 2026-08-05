import { useCallback, useEffect, useState } from 'react'
import { api } from './api/client'
import { NewScanModal } from './components/NewScanModal'
import { ProfilesPanel } from './components/ProfilesPanel'
import { ScanDetail } from './components/ScanDetail'
import { ScanHistory } from './components/ScanHistory'
import type { ScanProfileInfo, ScanRecord, ScanSummary } from './types'

export default function App() {
  const [profiles, setProfiles] = useState<ScanProfileInfo[]>([])
  const [scans, setScans] = useState<ScanSummary[]>([])
  const [profilesLoading, setProfilesLoading] = useState(true)
  const [scansLoading, setScansLoading] = useState(true)
  const [profilesError, setProfilesError] = useState('')
  const [scansError, setScansError] = useState('')
  const [modalOpen, setModalOpen] = useState(false)
  const [activeScanId, setActiveScanId] = useState<string | null>(null)
  const [activeScan, setActiveScan] = useState<ScanRecord | null>(null)
  const [detailLoading, setDetailLoading] = useState(false)
  const [detailError, setDetailError] = useState('')

  const loadProfiles = useCallback(async () => {
    setProfilesLoading(true)
    setProfilesError('')
    try {
      setProfiles(await api.getProfiles())
    } catch (err) {
      setProfilesError(err instanceof Error ? err.message : 'Failed to load profiles')
    } finally {
      setProfilesLoading(false)
    }
  }, [])

  const loadScans = useCallback(async () => {
    setScansLoading(true)
    setScansError('')
    try {
      setScans(await api.listScans())
    } catch (err) {
      setScansError(err instanceof Error ? err.message : 'Failed to load scans')
    } finally {
      setScansLoading(false)
    }
  }, [])

  const loadDetail = useCallback(async (id: string) => {
    setDetailLoading(true)
    setDetailError('')
    try {
      setActiveScan(await api.getScan(id))
    } catch (err) {
      setDetailError(err instanceof Error ? err.message : 'Failed to load scan')
    } finally {
      setDetailLoading(false)
    }
  }, [])

  useEffect(() => {
    loadProfiles()
    loadScans()
  }, [loadProfiles, loadScans])

  const inProgress =
    scans.some((s) => s.status === 'pending' || s.status === 'running') ||
    activeScan?.status === 'pending' ||
    activeScan?.status === 'running'

  useEffect(() => {
    if (!inProgress) return
    const timer = setInterval(() => {
      loadScans()
      if (activeScanId) loadDetail(activeScanId)
    }, 3000)
    return () => clearInterval(timer)
  }, [inProgress, activeScanId, loadScans, loadDetail])

  async function handleRunProfile(name: string) {
    try {
      await api.runProfile(name)
      await loadScans()
    } catch (err) {
      alert(err instanceof Error ? err.message : 'Failed to start scan')
    }
  }

  function handleView(id: string) {
    setActiveScanId(id)
    loadDetail(id)
  }

  function handleCloseDetail() {
    setActiveScanId(null)
    setActiveScan(null)
    setDetailError('')
  }

  return (
    <div className="mx-auto min-h-screen max-w-6xl p-6">
      <header className="mb-8 flex flex-col gap-4 border-b border-border pb-6 sm:flex-row sm:items-center sm:justify-between">
        <div className="flex items-center gap-4">
          <span className="text-4xl text-primary">◈</span>
          <div>
            <h1 className="text-2xl font-semibold">Drift Detector</h1>
            <p className="text-sm text-muted">Terraform state vs cloud infrastructure</p>
          </div>
        </div>
        <div className="flex gap-3">
          <button
            onClick={() => {
              loadProfiles()
              loadScans()
            }}
            className="rounded-lg bg-surface-2 px-4 py-2 text-sm hover:bg-border"
          >
            Refresh
          </button>
          <button
            onClick={() => setModalOpen(true)}
            className="rounded-lg bg-primary px-4 py-2 text-sm font-medium text-white hover:bg-primary-hover"
          >
            New Scan
          </button>
        </div>
      </header>

      <main className="space-y-6">
        <ProfilesPanel
          profiles={profiles}
          loading={profilesLoading}
          error={profilesError}
          onRunProfile={handleRunProfile}
        />
        <ScanHistory
          scans={scans}
          loading={scansLoading}
          error={scansError}
          onView={handleView}
        />
        {(activeScanId || detailLoading) && (
          <ScanDetail
            scan={activeScan}
            loading={detailLoading}
            error={detailError}
            onClose={handleCloseDetail}
          />
        )}
      </main>

      <NewScanModal
        open={modalOpen}
        onClose={() => setModalOpen(false)}
        onCreated={loadScans}
      />
    </div>
  )
}

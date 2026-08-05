import { useEffect, useState } from 'react'
import type { FormEvent } from 'react'
import { api } from '../api/client'
import type { CreateScanRequest, Provider } from '../types'

interface Props {
  open: boolean
  onClose: () => void
  onCreated: () => void
}

export function NewScanModal({ open, onClose, onCreated }: Props) {
  const [submitting, setSubmitting] = useState(false)
  const [error, setError] = useState('')

  useEffect(() => {
    if (!open) setError('')
  }, [open])

  if (!open) return null

  async function handleSubmit(e: FormEvent<HTMLFormElement>) {
    e.preventDefault()
    setSubmitting(true)
    setError('')
    const fd = new FormData(e.currentTarget)
    const body: CreateScanRequest = {
      provider: fd.get('provider') as Provider,
      state_path: String(fd.get('state_path')),
      regions: splitCSV(String(fd.get('regions') || '')),
      resource_types: splitCSV(String(fd.get('resource_types') || '')),
    }
    try {
      await api.createScan(body)
      onCreated()
      onClose()
      e.currentTarget.reset()
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to start scan')
    } finally {
      setSubmitting(false)
    }
  }

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/60 p-4">
      <div className="w-full max-w-md rounded-xl border border-border bg-surface p-6 shadow-xl">
        <h3 className="mb-4 text-lg font-semibold">Trigger New Scan</h3>
        <form onSubmit={handleSubmit} className="space-y-4">
          <label className="block text-sm">
            <span className="mb-1 block text-muted">Provider</span>
            <select
              name="provider"
              required
              className="w-full rounded-lg border border-border bg-surface-2 px-3 py-2 text-sm"
            >
              <option value="aws">AWS</option>
              <option value="azure">Azure</option>
              <option value="gcp">GCP</option>
            </select>
          </label>
          <label className="block text-sm">
            <span className="mb-1 block text-muted">State path or URI</span>
            <input
              name="state_path"
              required
              placeholder="s3://bucket/key or ./terraform.tfstate"
              className="w-full rounded-lg border border-border bg-surface-2 px-3 py-2 text-sm"
            />
          </label>
          <label className="block text-sm">
            <span className="mb-1 block text-muted">Regions (comma-separated)</span>
            <input
              name="regions"
              placeholder="us-east-1, us-west-2"
              className="w-full rounded-lg border border-border bg-surface-2 px-3 py-2 text-sm"
            />
          </label>
          <label className="block text-sm">
            <span className="mb-1 block text-muted">Resource types (comma-separated)</span>
            <input
              name="resource_types"
              placeholder="aws_s3_bucket, aws_instance"
              className="w-full rounded-lg border border-border bg-surface-2 px-3 py-2 text-sm"
            />
          </label>
          {error && <p className="text-sm text-danger">{error}</p>}
          <div className="flex justify-end gap-3 pt-2">
            <button
              type="button"
              onClick={onClose}
              className="rounded-lg bg-surface-2 px-4 py-2 text-sm hover:bg-border"
            >
              Cancel
            </button>
            <button
              type="submit"
              disabled={submitting}
              className="rounded-lg bg-primary px-4 py-2 text-sm font-medium text-white hover:bg-primary-hover disabled:opacity-50"
            >
              {submitting ? 'Starting…' : 'Start Scan'}
            </button>
          </div>
        </form>
      </div>
    </div>
  )
}

function splitCSV(value: string): string[] {
  return value
    .split(',')
    .map((s) => s.trim())
    .filter(Boolean)
}

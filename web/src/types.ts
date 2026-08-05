export type Provider = 'aws' | 'azure' | 'gcp'
export type ScanStatus = 'pending' | 'running' | 'completed' | 'failed'

export interface ScanProfileInfo {
  name: string
  provider: Provider
  regions?: string[]
  resource_types?: string[]
  schedule?: string
  state_source: string
}

export interface ScanSummary {
  id: string
  status: ScanStatus
  provider: Provider
  state_source: string
  profile_name?: string
  created_at: string
  started_at?: string
  completed_at?: string
  error?: string
  total_drifts: number
}

export interface DriftSummary {
  total_drifts: number
  missing_in_cloud: number
  missing_in_state: number
  attribute_drifts: number
  tag_drifts: number
  resources_checked: number
}

export interface AttributeChange {
  attribute: string
  expected?: unknown
  actual?: unknown
}

export interface TagChange {
  key: string
  expected?: string
  actual?: string
  change: string
}

export interface DriftItem {
  type: string
  resource_type: string
  resource_id: string
  resource_name?: string
  terraform_ref?: string
  region?: string
  attribute_changes?: AttributeChange[]
  tag_changes?: TagChange[]
  message?: string
}

export interface DriftReport {
  scan_id: string
  started_at: string
  completed_at: string
  state_source: string
  provider: Provider
  summary: DriftSummary
  drifts: DriftItem[]
}

export interface ScanRecord {
  id: string
  status: ScanStatus
  provider: Provider
  state_source: string
  created_at: string
  started_at?: string
  completed_at?: string
  error?: string
  options?: Record<string, unknown>
  report?: DriftReport
}

export interface CreateScanRequest {
  provider: Provider
  state_path: string
  regions?: string[]
  resource_types?: string[]
  account_id?: string
  project_id?: string
}

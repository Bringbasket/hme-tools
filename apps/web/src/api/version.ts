import { http, unwrap } from './http'

export interface SystemVersion {
  state: string
  action?: 'check' | 'update'
  message: string
  currentVersion: string
  latestVersion: string | null
  currentRevision: string
  latestRevision: string | null
  updateAvailable: boolean | null
  requestId: string | null
  requestedAt: number | null
  startedAt: number | null
  finishedAt: number | null
  updatedAt: number | null
  error: string | null
  canRequestUpdate: boolean
  repositoryUrl: string
}

export function getSystemVersion() {
  return unwrap<SystemVersion>(http.get('/system/version'))
}

export function checkSystemVersion() {
  return unwrap<SystemVersion>(http.post('/system/version/check'))
}

export function requestSystemUpdate() {
  return unwrap<SystemVersion>(http.post('/system/version/update'))
}

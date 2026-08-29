import type { AxiosRequestConfig } from 'axios'
import { http } from '@/api/http'
import type { ActivityLogPage, AppleLoginResult, AutoRefreshStatus, BatchShareLinkItem, CreateScheduleStatus, MailAlias, MailboxSettings, MailboxSettingsInput, MailboxStatus, MailMessage, SessionStatus, ShareLink } from './types'

// Axios already prefixes requests with /api/v1.
const base = '/mail'

interface MailEnvelope<T> {
  ok: boolean
  data: T
  error?: { code?: string; message?: string } | null
}

const mailAccountStorageKey = 'running-mail-account-id'

export async function mailRequest<T>(path: string, config: AxiosRequestConfig = {}): Promise<T> {
  const headers = { ...(config.headers || {}), 'X-Mail-Account-ID': localStorage.getItem(mailAccountStorageKey) || 'default' }
  const response = await http.request<MailEnvelope<T>>({ ...config, url: path, headers })
  const envelope = response.data
  if (!envelope?.ok) {
    throw new Error(envelope?.error?.message || `邮件请求失败（HTTP ${response.status}）`)
  }
  return envelope.data
}

export const mailAPI = {
  aliases: () => mailRequest<MailAlias[]>(`${base}/aliases`),
  aliasAction: (id: string, action: 'enable' | 'disable' | 'delete') => mailRequest(`${base}/aliases/${encodeURIComponent(id)}/${action}`, { method: 'POST', data: {} }),
  updateAlias: (id: string, label: string, note: string) => mailRequest<MailAlias>(`${base}/aliases/${encodeURIComponent(id)}/update`, { method: 'POST', data: { label, note } }),
  shareLinks: (id: string) => mailRequest<{ alias: string; links: ShareLink[] }>(`${base}/aliases/${encodeURIComponent(id)}/share-links`),
  createShareLink: (id: string, expiresInSeconds: number | null) => mailRequest<ShareLink>(`${base}/aliases/${encodeURIComponent(id)}/share-links`, { method: 'POST', data: { expiresInSeconds } }),
  createBatchShareLinks: (count: number, expiresInSeconds: number | null, scope: 'all' | 'gpt_registered' | 'gpt_unregistered' = 'all') => mailRequest<{ items: BatchShareLinkItem[]; count: number; scope: string }>(`${base}/aliases/batch-share-links`, { method: 'POST', data: { count, expiresInSeconds, scope } }),
  revokeShareLink: (id: string) => mailRequest(`${base}/share-links/${encodeURIComponent(id)}/revoke`, { method: 'POST', data: {} }),
  clearInactiveShareLinks: () => mailRequest<{ cleared: boolean; deleted: number }>(`${base}/share-links/clear-inactive`, { method: 'POST', data: {} }),
  mailboxStatus: () => mailRequest<MailboxStatus>(`${base}/mail/sync/status`),
  mailboxRun: () => mailRequest<MailboxStatus>(`${base}/mail/sync/run`, { method: 'POST', data: {} }),
  mailboxSettings: () => mailRequest<MailboxSettings>(`${base}/mail/settings`),
  updateMailboxSettings: (payload: MailboxSettingsInput) => mailRequest<MailboxSettings>(`${base}/mail/settings`, { method: 'PUT', data: payload }),
  testMailboxSettings: (payload: MailboxSettingsInput) => mailRequest<{ connected: boolean }>(`${base}/mail/settings/test`, { method: 'POST', data: payload }),
  mailboxWait: (revision: number, timeout = 25) => mailRequest<MailboxStatus>(`${base}/mail/sync/wait?revision=${revision}&timeout=${timeout}`, { timeout: (timeout + 5) * 1000 }),
  mailboxMessages: (alias: string, limit = 10) => mailRequest<{ configured: boolean; alias: string; messages: MailMessage[]; sync: MailboxStatus }>(`${base}/mail/messages?alias=${encodeURIComponent(alias)}&limit=${limit}`),
  mailboxRecent: (limit = 200) => mailRequest<{ days: number; messages: MailMessage[]; sync: MailboxStatus }>(`${base}/mail/recent?limit=${limit}`),
  mailboxMessage: (alias: string, uid: number) => mailRequest<MailMessage>(`${base}/mail/messages/${uid}?alias=${encodeURIComponent(alias)}`),
  hideMailboxMessage: (alias: string, uid: number, sync: MailboxStatus) => mailRequest(`${base}/mail/messages/${uid}/hide`, { method: 'POST', data: { alias, uidValidity: sync.uidValidity, mailboxGeneration: sync.mailboxGeneration } }),
  hideMailboxMessages: (messages: Array<{ alias: string; uid: number }>, sync: MailboxStatus) => mailRequest(`${base}/mail/messages/hide-batch`, { method: 'POST', data: { messages, uidValidity: sync.uidValidity, mailboxGeneration: sync.mailboxGeneration } }),
  clearMailboxMessages: () => mailRequest<{ cleared: boolean }>(`${base}/mail/messages/clear`, { method: 'POST', data: {} }),
  session: () => mailRequest<SessionStatus>(`${base}/session/status`),
  refreshSession: () => mailRequest<SessionStatus>(`${base}/session/refresh`, { method: 'POST', data: {} }),
  importSession: (curlText: string) => mailRequest<{ imported: boolean; icloudRegion: string; host: string }>(`${base}/session/import`, { method: 'POST', data: { curl_text: curlText } }),
  startAppleLogin: (payload: { appleId: string; password: string; channel: 'icloud_web' | 'apple_account'; twoFactorMethod: 'trusted_device' | 'phone' }) => mailRequest<AppleLoginResult>(`${base}/session/apple-login/start`, { method: 'POST', data: payload }),
  verifyAppleLogin: (pendingId: string, code: string) => mailRequest<AppleLoginResult>(`${base}/session/apple-login/verify`, { method: 'POST', data: { pendingId, code } }),
  autoRefresh: () => mailRequest<AutoRefreshStatus>(`${base}/auto-refresh`),
  updateAutoRefresh: (payload: { enabled?: boolean; intervalSeconds?: number }) => mailRequest<AutoRefreshStatus>(`${base}/auto-refresh`, { method: 'POST', data: payload }),
  runAutoRefresh: () => mailRequest<{ autoRefresh: AutoRefreshStatus; session: SessionStatus }>(`${base}/auto-refresh/run`, { method: 'POST', data: {} }),
  createSchedule: () => mailRequest<CreateScheduleStatus>(`${base}/create-schedule`),
  updateCreateSchedule: (payload: Partial<Pick<CreateScheduleStatus, 'enabled' | 'batchSize' | 'aliasIntervalSeconds' | 'intervalSeconds' | 'label' | 'note'>>) => mailRequest<CreateScheduleStatus>(`${base}/create-schedule`, { method: 'POST', data: payload }),
  runCreateSchedule: () => mailRequest<CreateScheduleStatus>(`${base}/create-schedule/run`, { method: 'POST', data: {} }),
  stopCreateSchedule: () => mailRequest<CreateScheduleStatus>(`${base}/create-schedule/stop`, { method: 'POST', data: {} }),
  activityLogs: (query: { page: number; pageSize: number; search?: string; level?: string; category?: string; source?: string; start?: string; end?: string }) => {
    const params = new URLSearchParams()
    Object.entries(query).forEach(([key, value]) => { if (value !== undefined && value !== '') params.set(key, String(value)) })
    return mailRequest<ActivityLogPage>(`${base}/activity-logs?${params}`)
  },
  clearActivityLogs: () => mailRequest<{ cleared: boolean }>(`${base}/activity-logs/clear`, { method: 'POST', data: {} }),
}

const API_BASE = '/api/me/amp'

function getAuthHeader(): HeadersInit {
  const token = localStorage.getItem('token')
  return {
    'Content-Type': 'application/json',
    ...(token ? { Authorization: `Bearer ${token}` } : {}),
  }
}

async function handleResponse<T>(response: Response): Promise<T> {
  if (!response.ok) {
    const error = await response.json().catch(() => ({ error: '请求失败' }))
    throw new Error(error.error || '请求失败')
  }
  return response.json()
}

// Types
export type WebSearchMode = 'upstream' | 'builtin_free' | 'local_duckduckgo'

export interface ModelMapping {
  from: string
  to: string
  regex: boolean
  thinkingLevel?: 'low' | 'medium' | 'high' | 'xhigh' | ''
}

export interface AmpSettings {
  upstreamUrl: string
  apiKeySet: boolean
  forceModelMappings: boolean
  modelMappings: ModelMapping[]
  enabled: boolean
  webSearchMode?: WebSearchMode
}

export interface UpdateAmpSettingsRequest {
  upstreamUrl?: string
  upstreamApiKey?: string
  forceModelMappings?: boolean
  modelMappings?: ModelMapping[]
  enabled?: boolean
  webSearchMode?: WebSearchMode
}

export interface TestResult {
  success: boolean
  message: string
  latencyMs?: number
}

export interface APIKey {
  id: string
  name: string
  prefix: string
  lastUsedAt: string | null
  createdAt: string
}

export interface CreateAPIKeyResponse {
  id: string
  name: string
  prefix: string
  apiKey: string
  createdAt: string
  message: string
}

export interface APIKeyRevealResponse {
  id: string
  name: string
  prefix: string
  apiKey: string
  createdAt: string
}

export interface BootstrapInfo {
  proxyBaseUrl: string
  configExample: string
  hasSettings: boolean
  hasApiKey: boolean
}

// Settings API
export async function getAmpSettings(): Promise<AmpSettings> {
  const response = await fetch(`${API_BASE}/settings`, {
    headers: getAuthHeader(),
  })
  return handleResponse<AmpSettings>(response)
}

export async function updateAmpSettings(data: UpdateAmpSettingsRequest): Promise<AmpSettings> {
  const response = await fetch(`${API_BASE}/settings`, {
    method: 'PUT',
    headers: getAuthHeader(),
    body: JSON.stringify(data),
  })
  return handleResponse<AmpSettings>(response)
}

export async function testAmpConnection(): Promise<TestResult> {
  const response = await fetch(`${API_BASE}/settings/test`, {
    method: 'POST',
    headers: getAuthHeader(),
  })
  return handleResponse<TestResult>(response)
}

// API Key
export async function getAPIKeys(): Promise<APIKey[]> {
  const response = await fetch(`${API_BASE}/api-keys`, {
    headers: getAuthHeader(),
  })
  const data = await handleResponse<{ apiKeys: APIKey[] }>(response)
  return data.apiKeys || []
}

export async function createAPIKey(name: string): Promise<CreateAPIKeyResponse> {
  const response = await fetch(`${API_BASE}/api-keys`, {
    method: 'POST',
    headers: getAuthHeader(),
    body: JSON.stringify({ name }),
  })
  return handleResponse<CreateAPIKeyResponse>(response)
}

export async function deleteAPIKey(id: string): Promise<void> {
  const response = await fetch(`${API_BASE}/api-keys/${id}`, {
    method: 'DELETE',
    headers: getAuthHeader(),
  })
  if (!response.ok) {
    const error = await response.json().catch(() => ({ error: '删除失败' }))
    throw new Error(error.error || '删除失败')
  }
}

export async function getAPIKey(id: string): Promise<APIKeyRevealResponse> {
  const response = await fetch(`${API_BASE}/api-keys/${id}`, {
    headers: getAuthHeader(),
  })
  return handleResponse<APIKeyRevealResponse>(response)
}

// Bootstrap API
export async function getAmpBootstrap(): Promise<BootstrapInfo> {
  const response = await fetch(`${API_BASE}/bootstrap`, {
    headers: getAuthHeader(),
  })
  return handleResponse<BootstrapInfo>(response)
}

// Request Logs Types
export interface RequestLog {
  id: string
  createdAt: string
  userId: string
  username?: string
  apiKeyId: string
  originalModel?: string
  mappedModel?: string
  provider?: string
  channelId?: string
  endpoint?: string
  method: string
  path: string
  statusCode: number
  latencyMs: number
  isStreaming: boolean
  inputTokens?: number
  outputTokens?: number
  cacheReadInputTokens?: number
  cacheCreationInputTokens?: number
  errorType?: string
  requestId?: string
  costMicros?: number
  costUsd?: string
  pricingModel?: string
  thinkingLevel?: string
}

export interface RequestLogListResponse {
  items: RequestLog[]
  total: number
  page: number
  pageSize: number
}

export interface UsageSummary {
  groupKey: string
  inputTokensSum: number
  outputTokensSum: number
  cacheReadInputTokensSum: number
  cacheCreationInputTokensSum: number
  requestCount: number
  errorCount: number
  costMicrosSum?: number
  costUsdSum?: string
}

export interface UsageSummaryResponse {
  items: UsageSummary[]
}

export interface RequestLogListParams {
  page?: number
  pageSize?: number
  apiKeyId?: string
  model?: string
  status?: number
  isStreaming?: boolean
  from?: string
  to?: string
}

// Request Logs API
export async function getRequestLogs(params: RequestLogListParams = {}, signal?: AbortSignal): Promise<RequestLogListResponse> {
  const searchParams = new URLSearchParams()
  if (params.page) searchParams.set('page', params.page.toString())
  if (params.pageSize) searchParams.set('pageSize', params.pageSize.toString())
  if (params.apiKeyId) searchParams.set('apiKeyId', params.apiKeyId)
  if (params.model) searchParams.set('model', params.model)
  if (params.status !== undefined) searchParams.set('status', params.status.toString())
  if (params.isStreaming !== undefined) searchParams.set('isStreaming', params.isStreaming.toString())
  if (params.from) searchParams.set('from', params.from)
  if (params.to) searchParams.set('to', params.to)

  const query = searchParams.toString()
  const response = await fetch(`${API_BASE}/request-logs${query ? `?${query}` : ''}`, {
    headers: getAuthHeader(),
    signal,
  })
  return handleResponse<RequestLogListResponse>(response)
}

export async function getUsageSummary(params: { from?: string; to?: string; groupBy?: string } = {}, signal?: AbortSignal): Promise<UsageSummaryResponse> {
  const searchParams = new URLSearchParams()
  if (params.from) searchParams.set('from', params.from)
  if (params.to) searchParams.set('to', params.to)
  if (params.groupBy) searchParams.set('groupBy', params.groupBy)

  const query = searchParams.toString()
  const response = await fetch(`${API_BASE}/usage/summary${query ? `?${query}` : ''}`, {
    headers: getAuthHeader(),
    signal,
  })
  return handleResponse<UsageSummaryResponse>(response)
}

// Admin API for request logs
const ADMIN_API_BASE = '/api/admin'

export interface AdminRequestLogListParams extends RequestLogListParams {
  userId?: string
}

export async function getAdminRequestLogs(params: AdminRequestLogListParams = {}, signal?: AbortSignal): Promise<RequestLogListResponse> {
  const searchParams = new URLSearchParams()
  if (params.page) searchParams.set('page', params.page.toString())
  if (params.pageSize) searchParams.set('pageSize', params.pageSize.toString())
  if (params.userId) searchParams.set('userId', params.userId)
  if (params.apiKeyId) searchParams.set('apiKeyId', params.apiKeyId)
  if (params.model) searchParams.set('model', params.model)
  if (params.status !== undefined) searchParams.set('status', params.status.toString())
  if (params.isStreaming !== undefined) searchParams.set('isStreaming', params.isStreaming.toString())
  if (params.from) searchParams.set('from', params.from)
  if (params.to) searchParams.set('to', params.to)

  const query = searchParams.toString()
  const response = await fetch(`${ADMIN_API_BASE}/request-logs${query ? `?${query}` : ''}`, {
    headers: getAuthHeader(),
    signal,
  })
  return handleResponse<RequestLogListResponse>(response)
}

export async function getAdminDistinctModels(signal?: AbortSignal): Promise<{ models: string[] }> {
  const response = await fetch(`${ADMIN_API_BASE}/request-logs/models`, {
    headers: getAuthHeader(),
    signal,
  })
  return handleResponse<{ models: string[] }>(response)
}

export async function getAdminUsageSummary(params: { from?: string; to?: string; groupBy?: string; userId?: string } = {}, signal?: AbortSignal): Promise<UsageSummaryResponse> {
  const searchParams = new URLSearchParams()
  if (params.from) searchParams.set('from', params.from)
  if (params.to) searchParams.set('to', params.to)
  if (params.groupBy) searchParams.set('groupBy', params.groupBy)
  if (params.userId) searchParams.set('userId', params.userId)

  const query = searchParams.toString()
  const response = await fetch(`${ADMIN_API_BASE}/usage/summary${query ? `?${query}` : ''}`, {
    headers: getAuthHeader(),
    signal,
  })
  return handleResponse<UsageSummaryResponse>(response)
}

// Request Log Detail API
export interface RequestLogDetail {
  requestId: string
  requestHeaders: Record<string, string>
  requestBody: string
  translatedRequestBody?: string
  responseHeaders: Record<string, string>
  responseBody: string
  translatedResponseBody?: string
  createdAt: string
}

export async function getAdminRequestLogDetail(logId: string, signal?: AbortSignal): Promise<RequestLogDetail> {
  const response = await fetch(`${ADMIN_API_BASE}/request-logs/${logId}/detail`, {
    headers: getAuthHeader(),
    signal,
  })
  return handleResponse<RequestLogDetail>(response)
}

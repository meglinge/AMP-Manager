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
}

export interface UpdateAmpSettingsRequest {
  upstreamUrl?: string
  upstreamApiKey?: string
  forceModelMappings?: boolean
  modelMappings?: ModelMapping[]
  enabled?: boolean
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
  isActive: boolean
  lastUsedAt: string | null
  revokedAt: string | null
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

export async function revokeAPIKey(id: string): Promise<void> {
  const response = await fetch(`${API_BASE}/api-keys/${id}/revoke`, {
    method: 'POST',
    headers: getAuthHeader(),
  })
  if (!response.ok) {
    const error = await response.json().catch(() => ({ error: '撤销失败' }))
    throw new Error(error.error || '撤销失败')
  }
}

// Bootstrap API
export async function getAmpBootstrap(): Promise<BootstrapInfo> {
  const response = await fetch(`${API_BASE}/bootstrap`, {
    headers: getAuthHeader(),
  })
  return handleResponse<BootstrapInfo>(response)
}

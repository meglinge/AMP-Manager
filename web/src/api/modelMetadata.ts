const API_BASE = '/api/admin/model-metadata'

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

export interface ModelMetadata {
  id: string
  modelPattern: string
  displayName: string
  contextLength: number
  maxCompletionTokens: number
  provider: string
  createdAt: string
  updatedAt: string
}

export interface ModelMetadataRequest {
  modelPattern: string
  displayName: string
  contextLength: number
  maxCompletionTokens: number
  provider: string
}

export async function listModelMetadata(): Promise<ModelMetadata[]> {
  const response = await fetch(API_BASE, {
    headers: getAuthHeader(),
  })
  const data = await handleResponse<{ metadata: ModelMetadata[] }>(response)
  return data.metadata || []
}

export async function createModelMetadata(data: ModelMetadataRequest): Promise<ModelMetadata> {
  const response = await fetch(API_BASE, {
    method: 'POST',
    headers: getAuthHeader(),
    body: JSON.stringify(data),
  })
  return handleResponse<ModelMetadata>(response)
}

export async function updateModelMetadata(id: string, data: ModelMetadataRequest): Promise<ModelMetadata> {
  const response = await fetch(`${API_BASE}/${id}`, {
    method: 'PUT',
    headers: getAuthHeader(),
    body: JSON.stringify(data),
  })
  return handleResponse<ModelMetadata>(response)
}

export async function deleteModelMetadata(id: string): Promise<void> {
  const response = await fetch(`${API_BASE}/${id}`, {
    method: 'DELETE',
    headers: getAuthHeader(),
  })
  if (!response.ok) {
    const error = await response.json().catch(() => ({ error: '删除失败' }))
    throw new Error(error.error || '删除失败')
  }
}

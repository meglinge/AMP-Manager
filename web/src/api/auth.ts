const API_BASE = '/api'

export interface RegisterRequest {
  username: string
  password: string
}

export interface LoginRequest {
  username: string
  password: string
}

export interface AuthResponse {
  id: string
  username: string
  token?: string
  isAdmin?: boolean
  message: string
}

export interface ApiError {
  error: string
  details?: string
}

export async function register(data: RegisterRequest): Promise<AuthResponse> {
  const response = await fetch(`${API_BASE}/manage/auth/register`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(data),
  })

  if (!response.ok) {
    const error: ApiError = await response.json()
    throw new Error(error.error || '注册失败')
  }

  return response.json()
}

export async function login(data: LoginRequest): Promise<AuthResponse> {
  const response = await fetch(`${API_BASE}/manage/auth/login`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(data),
  })

  if (!response.ok) {
    const error: ApiError = await response.json()
    throw new Error(error.error || '登录失败')
  }

  return response.json()
}

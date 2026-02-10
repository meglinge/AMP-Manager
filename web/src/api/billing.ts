import type { LimitType, WindowMode, SubscriptionStatus } from './subscription'

const API_BASE = '/api'

function getAuthHeaders(): HeadersInit {
  const token = localStorage.getItem('token')
  if (!token) {
    return {}
  }
  return {
    Authorization: `Bearer ${token}`,
  }
}

// 统一的 fetch + JSON 解析 helper
async function fetchJson<T>(url: string, options?: RequestInit): Promise<T> {
  const res = await fetch(url, options)
  
  let data: unknown
  let text = ''
  
  try {
    data = await res.json()
  } catch {
    try {
      text = await res.text()
    } catch {
      text = ''
    }
  }
  
  if (!res.ok) {
    const errorObj = data as { error?: string; message?: string } | undefined
    const errorMessage = errorObj?.error || errorObj?.message || 
      `请求失败 (${res.status})${text ? ': ' + text.slice(0, 100) : ''}`
    throw new Error(errorMessage)
  }
  
  return data as T
}

export interface ModelPrice {
  model: string
  provider?: string | null
  source: string
  inputCostPerToken: number
  outputCostPerToken: number
  cacheReadInputPerToken: number
  cacheCreationPerToken: number
  updatedAt: string
}

export interface PriceListResponse {
  items: ModelPrice[]
  total: number
}

export interface PriceStats {
  modelCount: number
  source: string
  fetchedAt: string
}

// 获取价格列表
export async function listPrices(): Promise<PriceListResponse> {
  return fetchJson<PriceListResponse>(`${API_BASE}/admin/prices`, {
    headers: getAuthHeaders(),
  })
}

// 获取价格服务状态
export async function getPriceStats(): Promise<PriceStats> {
  return fetchJson<PriceStats>(`${API_BASE}/admin/prices/stats`, {
    headers: getAuthHeaders(),
  })
}

// 手动刷新价格
export async function refreshPrices(): Promise<{ message: string; modelCount: number; fetchedAt: string }> {
  return fetchJson(`${API_BASE}/admin/prices/refresh`, {
    method: 'POST',
    headers: getAuthHeaders(),
  })
}

// --- User Billing API Types ---

export interface WindowRemaining {
  limitType: LimitType
  windowMode: WindowMode
  limitMicros: number
  usedMicros: number
  leftMicros: number
  windowStart: string
  windowEnd: string
}

export interface BillingStateSubscription {
  id: string
  userId: string
  planId: string
  planName: string
  startsAt: string
  expiresAt: string | null
  status: SubscriptionStatus
  limits: {
    id: string
    planId: string
    limitType: LimitType
    windowMode: WindowMode
    limitMicros: number
    createdAt: string
    updatedAt: string
  }[]
  createdAt: string
  updatedAt: string
}

export interface BillingStateResponse {
  balanceMicros: number
  balanceUsd: string
  subscription: BillingStateSubscription | null
  windows: WindowRemaining[] | null
  primarySource: 'subscription' | 'balance'
  secondarySource: 'subscription' | 'balance'
}

export interface UserBillingSetting {
  userId: string
  primarySource: 'subscription' | 'balance'
  secondarySource: 'subscription' | 'balance'
  createdAt: string
  updatedAt: string
}

// --- User Billing API ---

export async function getBillingState(): Promise<BillingStateResponse> {
  return fetchJson<BillingStateResponse>(`${API_BASE}/me/billing/state`, {
    headers: getAuthHeaders(),
  })
}

export async function updateBillingPriority(primarySource: 'subscription' | 'balance'): Promise<UserBillingSetting> {
  return fetchJson<UserBillingSetting>(`${API_BASE}/me/billing/priority`, {
    method: 'PUT',
    headers: {
      ...getAuthHeaders(),
      'Content-Type': 'application/json',
    },
    body: JSON.stringify({ primarySource }),
  })
}

export async function getMySubscription(): Promise<{ subscription: BillingStateSubscription | null }> {
  return fetchJson<{ subscription: BillingStateSubscription | null }>(`${API_BASE}/me/subscription`, {
    headers: getAuthHeaders(),
  })
}

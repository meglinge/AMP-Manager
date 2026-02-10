const API_BASE = '/api'

function getAuthHeaders() {
  const token = localStorage.getItem('token')
  return {
    'Content-Type': 'application/json',
    Authorization: `Bearer ${token}`,
  }
}

export type LimitType = 'daily' | 'weekly' | 'monthly' | 'rolling_5h' | 'total'
export type WindowMode = 'fixed' | 'sliding'
export type SubscriptionStatus = 'active' | 'paused' | 'expired' | 'cancelled'

export interface SubscriptionPlanLimit {
  id: string
  planId: string
  limitType: LimitType
  windowMode: WindowMode
  limitMicros: number
  createdAt: string
  updatedAt: string
}

export interface SubscriptionPlanResponse {
  id: string
  name: string
  description: string
  enabled: boolean
  limits: SubscriptionPlanLimit[]
  createdAt: string
  updatedAt: string
}

export interface PlanLimitRequest {
  limitType: LimitType
  windowMode: WindowMode
  limitMicros: number
}

export interface SubscriptionPlanRequest {
  name: string
  description: string
  enabled: boolean
  limits: PlanLimitRequest[]
}

export interface UserSubscriptionResponse {
  id: string
  userId: string
  planId: string
  planName: string
  startsAt: string
  expiresAt: string | null
  status: SubscriptionStatus
  limits: SubscriptionPlanLimit[]
  createdAt: string
  updatedAt: string
}

// Plan CRUD
export async function getPlans(): Promise<SubscriptionPlanResponse[]> {
  const res = await fetch(`${API_BASE}/admin/subscriptions/plans`, {
    headers: getAuthHeaders(),
  })
  if (!res.ok) {
    const data = await res.json()
    throw new Error(data.error || '获取套餐列表失败')
  }
  const data = await res.json()
  return data.plans || []
}

export async function createPlan(req: SubscriptionPlanRequest): Promise<SubscriptionPlanResponse> {
  const res = await fetch(`${API_BASE}/admin/subscriptions/plans`, {
    method: 'POST',
    headers: getAuthHeaders(),
    body: JSON.stringify(req),
  })
  if (!res.ok) {
    const data = await res.json()
    throw new Error(data.error || '创建套餐失败')
  }
  return res.json()
}

export async function getPlan(id: string): Promise<SubscriptionPlanResponse> {
  const res = await fetch(`${API_BASE}/admin/subscriptions/plans/${id}`, {
    headers: getAuthHeaders(),
  })
  if (!res.ok) {
    const data = await res.json()
    throw new Error(data.error || '获取套餐详情失败')
  }
  return res.json()
}

export async function updatePlan(id: string, req: SubscriptionPlanRequest): Promise<SubscriptionPlanResponse> {
  const res = await fetch(`${API_BASE}/admin/subscriptions/plans/${id}`, {
    method: 'PUT',
    headers: getAuthHeaders(),
    body: JSON.stringify(req),
  })
  if (!res.ok) {
    const data = await res.json()
    throw new Error(data.error || '更新套餐失败')
  }
  return res.json()
}

export async function deletePlan(id: string): Promise<void> {
  const res = await fetch(`${API_BASE}/admin/subscriptions/plans/${id}`, {
    method: 'DELETE',
    headers: getAuthHeaders(),
  })
  if (!res.ok) {
    const data = await res.json()
    throw new Error(data.error || '删除套餐失败')
  }
}

export async function setPlanEnabled(id: string, enabled: boolean): Promise<void> {
  const res = await fetch(`${API_BASE}/admin/subscriptions/plans/${id}/enabled`, {
    method: 'PATCH',
    headers: getAuthHeaders(),
    body: JSON.stringify({ enabled }),
  })
  if (!res.ok) {
    const data = await res.json()
    throw new Error(data.error || '更新套餐状态失败')
  }
}

// User subscription management
export async function getUserSubscription(userId: string): Promise<UserSubscriptionResponse | null> {
  const res = await fetch(`${API_BASE}/admin/users/${userId}/subscription`, {
    headers: getAuthHeaders(),
  })
  if (!res.ok) {
    if (res.status === 404) return null
    const data = await res.json()
    throw new Error(data.error || '获取用户订阅失败')
  }
  const data = await res.json()
  return data.subscription || null
}

export async function assignSubscription(userId: string, req: { planId: string; expiresAt?: string }): Promise<UserSubscriptionResponse> {
  const res = await fetch(`${API_BASE}/admin/users/${userId}/subscription`, {
    method: 'POST',
    headers: getAuthHeaders(),
    body: JSON.stringify(req),
  })
  if (!res.ok) {
    const data = await res.json()
    throw new Error(data.error || '分配订阅失败')
  }
  return res.json()
}

export async function cancelSubscription(userId: string): Promise<void> {
  const res = await fetch(`${API_BASE}/admin/users/${userId}/subscription`, {
    method: 'DELETE',
    headers: getAuthHeaders(),
  })
  if (!res.ok) {
    const data = await res.json()
    throw new Error(data.error || '取消订阅失败')
  }
}

export async function updateSubscriptionExpiry(userId: string, expiresAt: string): Promise<void> {
  const res = await fetch(`${API_BASE}/admin/users/${userId}/subscription`, {
    method: 'PATCH',
    headers: getAuthHeaders(),
    body: JSON.stringify({ expiresAt }),
  })
  if (!res.ok) {
    const data = await res.json()
    throw new Error(data.error || '更新到期时间失败')
  }
}

const API_BASE = '/api'

function getAuthHeaders() {
  const token = localStorage.getItem('token')
  return {
    'Content-Type': 'application/json',
    Authorization: `Bearer ${token}`,
  }
}

export interface DashboardPeriodStats {
  requestCount: number
  inputTokensSum: number
  outputTokensSum: number
  costMicros: number
  costUsd: string
  errorCount: number
}

export interface DashboardTopModel {
  model: string
  requestCount: number
  costMicros: number
  costUsd: string
}

export interface DashboardDailyTrend {
  date: string
  costMicros: number
  costUsd: string
  requests: number
}

export interface DashboardCacheHitRate {
  provider: string
  totalInputTokens: number
  cacheReadTokens: number
  cacheCreationTokens: number
  requestCount: number
  hitRate: string
}

export interface DashboardData {
  balance: {
    balanceMicros: number
    balanceUsd: string
  }
  today: DashboardPeriodStats
  week: DashboardPeriodStats
  month: DashboardPeriodStats
  topModels: DashboardTopModel[]
  dailyTrend: DashboardDailyTrend[]
  cacheHitRates: DashboardCacheHitRate[]
}

export async function getDashboard(): Promise<DashboardData> {
  const res = await fetch(`${API_BASE}/me/dashboard`, {
    headers: getAuthHeaders(),
  })
  if (!res.ok) throw new Error('获取仪表盘数据失败')
  return res.json()
}

const API_BASE = '/api'

function getAuthHeaders() {
  const token = localStorage.getItem('token')
  return {
    'Content-Type': 'application/json',
    Authorization: `Bearer ${token}`,
  }
}

export interface UserInfo {
  id: string
  username: string
  isAdmin: boolean
  createdAt: string
  updatedAt: string
}

export async function listUsers(): Promise<UserInfo[]> {
  const res = await fetch(`${API_BASE}/admin/users`, {
    headers: getAuthHeaders(),
  })
  if (!res.ok) throw new Error('获取用户列表失败')
  return res.json()
}

export async function setUserAdmin(userId: string, isAdmin: boolean): Promise<void> {
  const res = await fetch(`${API_BASE}/admin/users/${userId}/admin`, {
    method: 'PATCH',
    headers: getAuthHeaders(),
    body: JSON.stringify({ isAdmin }),
  })
  if (!res.ok) {
    const data = await res.json()
    throw new Error(data.error || '设置权限失败')
  }
}

export async function deleteUser(userId: string): Promise<void> {
  const res = await fetch(`${API_BASE}/admin/users/${userId}`, {
    method: 'DELETE',
    headers: getAuthHeaders(),
  })
  if (!res.ok) {
    const data = await res.json()
    throw new Error(data.error || '删除用户失败')
  }
}

export async function resetUserPassword(userId: string, newPassword: string): Promise<void> {
  const res = await fetch(`${API_BASE}/admin/users/${userId}/reset-password`, {
    method: 'POST',
    headers: getAuthHeaders(),
    body: JSON.stringify({ newPassword }),
  })
  if (!res.ok) {
    const data = await res.json()
    throw new Error(data.error || '重置密码失败')
  }
}

export async function changePassword(oldPassword: string, newPassword: string): Promise<void> {
  const res = await fetch(`${API_BASE}/me/password`, {
    method: 'PUT',
    headers: getAuthHeaders(),
    body: JSON.stringify({ oldPassword, newPassword }),
  })
  if (!res.ok) {
    const data = await res.json()
    throw new Error(data.error || '修改密码失败')
  }
}

export async function changeUsername(newUsername: string): Promise<void> {
  const res = await fetch(`${API_BASE}/me/username`, {
    method: 'PUT',
    headers: getAuthHeaders(),
    body: JSON.stringify({ newUsername }),
  })
  if (!res.ok) {
    const data = await res.json()
    throw new Error(data.error || '修改用户名失败')
  }
}

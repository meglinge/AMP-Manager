import { useState, useEffect } from 'react'
import {
  listUsers,
  setUserAdmin,
  deleteUser,
  resetUserPassword,
  UserInfo,
} from '../api/users'

export default function UserManagement() {
  const [users, setUsers] = useState<UserInfo[]>([])
  const [loading, setLoading] = useState(true)
  const [message, setMessage] = useState<{ type: 'success' | 'error'; text: string } | null>(null)
  const [resetPasswordModal, setResetPasswordModal] = useState<{ userId: string; username: string } | null>(null)
  const [newPassword, setNewPassword] = useState('')

  useEffect(() => {
    fetchUsers()
  }, [])

  const fetchUsers = async () => {
    try {
      const data = await listUsers()
      setUsers(data)
    } catch (err) {
      showMessage('error', err instanceof Error ? err.message : '获取用户列表失败')
    } finally {
      setLoading(false)
    }
  }

  const showMessage = (type: 'success' | 'error', text: string) => {
    setMessage({ type, text })
    setTimeout(() => setMessage(null), 3000)
  }

  const handleToggleAdmin = async (user: UserInfo) => {
    try {
      await setUserAdmin(user.id, !user.isAdmin)
      showMessage('success', `已${user.isAdmin ? '取消' : '设置'}管理员权限`)
      fetchUsers()
    } catch (err) {
      showMessage('error', err instanceof Error ? err.message : '操作失败')
    }
  }

  const handleDelete = async (user: UserInfo) => {
    if (!confirm(`确定要删除用户 ${user.username} 吗？`)) return

    try {
      await deleteUser(user.id)
      showMessage('success', '用户已删除')
      fetchUsers()
    } catch (err) {
      showMessage('error', err instanceof Error ? err.message : '删除失败')
    }
  }

  const handleResetPassword = async () => {
    if (!resetPasswordModal || !newPassword) return

    try {
      await resetUserPassword(resetPasswordModal.userId, newPassword)
      showMessage('success', '密码已重置')
      setResetPasswordModal(null)
      setNewPassword('')
    } catch (err) {
      showMessage('error', err instanceof Error ? err.message : '重置密码失败')
    }
  }

  const formatDate = (dateStr: string) => {
    return new Date(dateStr).toLocaleString('zh-CN')
  }

  if (loading) {
    return <div className="text-center text-gray-500">加载中...</div>
  }

  return (
    <div className="space-y-6">
      {message && (
        <div
          className={`rounded-md p-4 ${
            message.type === 'success' ? 'bg-green-100 text-green-800' : 'bg-red-100 text-red-800'
          }`}
        >
          {message.text}
        </div>
      )}

      <div className="rounded-lg bg-white p-6 shadow-md">
        <h3 className="mb-4 text-lg font-bold text-gray-800">用户列表</h3>

        <div className="overflow-x-auto">
          <table className="w-full text-left text-sm">
            <thead className="bg-gray-50">
              <tr>
                <th className="px-4 py-3 font-medium text-gray-700">用户名</th>
                <th className="px-4 py-3 font-medium text-gray-700">角色</th>
                <th className="px-4 py-3 font-medium text-gray-700">创建时间</th>
                <th className="px-4 py-3 font-medium text-gray-700">操作</th>
              </tr>
            </thead>
            <tbody className="divide-y">
              {users.map((user) => (
                <tr key={user.id} className="hover:bg-gray-50">
                  <td className="px-4 py-3 font-medium">{user.username}</td>
                  <td className="px-4 py-3">
                    <span
                      className={`rounded-full px-2 py-1 text-xs ${
                        user.isAdmin
                          ? 'bg-blue-100 text-blue-800'
                          : 'bg-gray-100 text-gray-800'
                      }`}
                    >
                      {user.isAdmin ? '管理员' : '普通用户'}
                    </span>
                  </td>
                  <td className="px-4 py-3 text-gray-500">{formatDate(user.createdAt)}</td>
                  <td className="px-4 py-3">
                    <div className="flex gap-2">
                      <button
                        onClick={() => handleToggleAdmin(user)}
                        className={`rounded px-3 py-1 text-xs text-white ${
                          user.isAdmin
                            ? 'bg-yellow-500 hover:bg-yellow-600'
                            : 'bg-blue-500 hover:bg-blue-600'
                        }`}
                      >
                        {user.isAdmin ? '取消管理员' : '设为管理员'}
                      </button>
                      <button
                        onClick={() => setResetPasswordModal({ userId: user.id, username: user.username })}
                        className="rounded bg-gray-500 px-3 py-1 text-xs text-white hover:bg-gray-600"
                      >
                        重置密码
                      </button>
                      <button
                        onClick={() => handleDelete(user)}
                        className="rounded bg-red-500 px-3 py-1 text-xs text-white hover:bg-red-600"
                      >
                        删除
                      </button>
                    </div>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      </div>

      {/* 重置密码弹窗 */}
      {resetPasswordModal && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black bg-opacity-50">
          <div className="w-96 rounded-lg bg-white p-6 shadow-xl">
            <h3 className="mb-4 text-lg font-bold">重置密码</h3>
            <p className="mb-4 text-sm text-gray-600">
              为用户 <span className="font-medium">{resetPasswordModal.username}</span> 设置新密码
            </p>
            <input
              type="password"
              placeholder="新密码 (至少6位)"
              value={newPassword}
              onChange={(e) => setNewPassword(e.target.value)}
              className="mb-4 w-full rounded-md border px-3 py-2"
            />
            <div className="flex justify-end gap-2">
              <button
                onClick={() => {
                  setResetPasswordModal(null)
                  setNewPassword('')
                }}
                className="rounded bg-gray-300 px-4 py-2 text-gray-700 hover:bg-gray-400"
              >
                取消
              </button>
              <button
                onClick={handleResetPassword}
                disabled={newPassword.length < 6}
                className="rounded bg-blue-600 px-4 py-2 text-white hover:bg-blue-700 disabled:bg-gray-400"
              >
                确认重置
              </button>
            </div>
          </div>
        </div>
      )}
    </div>
  )
}

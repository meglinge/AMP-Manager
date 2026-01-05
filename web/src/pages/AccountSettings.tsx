import { useState } from 'react'
import { changePassword, changeUsername } from '../api/users'

interface Props {
  username: string
  onUsernameChange: (newUsername: string) => void
}

export default function AccountSettings({ username, onUsernameChange }: Props) {
  const [message, setMessage] = useState<{ type: 'success' | 'error'; text: string } | null>(null)

  const [oldPassword, setOldPassword] = useState('')
  const [newPassword, setNewPassword] = useState('')
  const [confirmPassword, setConfirmPassword] = useState('')
  const [changingPassword, setChangingPassword] = useState(false)

  const [newUsername, setNewUsername] = useState('')
  const [changingUsername, setChangingUsername] = useState(false)

  const showMessage = (type: 'success' | 'error', text: string) => {
    setMessage({ type, text })
    setTimeout(() => setMessage(null), 3000)
  }

  const handleChangePassword = async (e: React.FormEvent) => {
    e.preventDefault()

    if (newPassword !== confirmPassword) {
      showMessage('error', '两次输入的密码不一致')
      return
    }

    if (newPassword.length < 6) {
      showMessage('error', '新密码至少需要6个字符')
      return
    }

    setChangingPassword(true)
    try {
      await changePassword(oldPassword, newPassword)
      showMessage('success', '密码修改成功')
      setOldPassword('')
      setNewPassword('')
      setConfirmPassword('')
    } catch (err) {
      showMessage('error', err instanceof Error ? err.message : '修改失败')
    } finally {
      setChangingPassword(false)
    }
  }

  const handleChangeUsername = async (e: React.FormEvent) => {
    e.preventDefault()

    if (newUsername.length < 3) {
      showMessage('error', '用户名至少需要3个字符')
      return
    }

    setChangingUsername(true)
    try {
      await changeUsername(newUsername)
      showMessage('success', '用户名修改成功，请重新登录')
      onUsernameChange(newUsername)
      localStorage.setItem('username', newUsername)
      setNewUsername('')
    } catch (err) {
      showMessage('error', err instanceof Error ? err.message : '修改失败')
    } finally {
      setChangingUsername(false)
    }
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

      {/* 修改密码 */}
      <div className="rounded-lg bg-white p-6 shadow-md">
        <h3 className="mb-4 text-lg font-bold text-gray-800">修改密码</h3>
        <form onSubmit={handleChangePassword} className="space-y-4">
          <div>
            <label className="block text-sm font-medium text-gray-700">当前密码</label>
            <input
              type="password"
              value={oldPassword}
              onChange={(e) => setOldPassword(e.target.value)}
              className="mt-1 w-full rounded-md border px-3 py-2"
              required
            />
          </div>
          <div>
            <label className="block text-sm font-medium text-gray-700">新密码</label>
            <input
              type="password"
              value={newPassword}
              onChange={(e) => setNewPassword(e.target.value)}
              className="mt-1 w-full rounded-md border px-3 py-2"
              required
              minLength={6}
            />
          </div>
          <div>
            <label className="block text-sm font-medium text-gray-700">确认新密码</label>
            <input
              type="password"
              value={confirmPassword}
              onChange={(e) => setConfirmPassword(e.target.value)}
              className="mt-1 w-full rounded-md border px-3 py-2"
              required
            />
          </div>
          <button
            type="submit"
            disabled={changingPassword}
            className="rounded-md bg-blue-600 px-4 py-2 text-white hover:bg-blue-700 disabled:bg-gray-400"
          >
            {changingPassword ? '修改中...' : '修改密码'}
          </button>
        </form>
      </div>

      {/* 修改用户名 */}
      <div className="rounded-lg bg-white p-6 shadow-md">
        <h3 className="mb-4 text-lg font-bold text-gray-800">修改用户名</h3>
        <p className="mb-4 text-sm text-gray-500">当前用户名: {username}</p>
        <form onSubmit={handleChangeUsername} className="space-y-4">
          <div>
            <label className="block text-sm font-medium text-gray-700">新用户名</label>
            <input
              type="text"
              value={newUsername}
              onChange={(e) => setNewUsername(e.target.value)}
              className="mt-1 w-full rounded-md border px-3 py-2"
              required
              minLength={3}
              maxLength={32}
            />
          </div>
          <button
            type="submit"
            disabled={changingUsername}
            className="rounded-md bg-blue-600 px-4 py-2 text-white hover:bg-blue-700 disabled:bg-gray-400"
          >
            {changingUsername ? '修改中...' : '修改用户名'}
          </button>
        </form>
      </div>
    </div>
  )
}

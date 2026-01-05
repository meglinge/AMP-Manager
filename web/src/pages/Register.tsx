import { useState, FormEvent } from 'react'
import { register, RegisterRequest } from '../api/auth'

interface Props {
  onSwitch: () => void
  onSuccess: (username: string, token?: string, isAdmin?: boolean) => void
}

export default function Register({ onSwitch, onSuccess }: Props) {
  const [formData, setFormData] = useState<RegisterRequest>({
    username: '',
    password: '',
  })
  const [confirmPassword, setConfirmPassword] = useState('')
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState('')

  const handleSubmit = async (e: FormEvent) => {
    e.preventDefault()
    setError('')

    if (formData.password !== confirmPassword) {
      setError('两次输入的密码不一致')
      return
    }

    setLoading(true)
    try {
      const result = await register(formData)
      onSuccess(result.username, result.token)
    } catch (err) {
      setError(err instanceof Error ? err.message : '注册失败')
    } finally {
      setLoading(false)
    }
  }

  return (
    <div className="w-full max-w-md rounded-lg bg-white p-8 shadow-md">
      <h2 className="mb-6 text-center text-2xl font-bold text-gray-800">
        用户注册
      </h2>

      {error && (
        <div className="mb-4 rounded bg-red-100 p-3 text-red-700">{error}</div>
      )}

      <form onSubmit={handleSubmit} className="space-y-4">
        <div>
          <label className="block text-sm font-medium text-gray-700">
            用户名
          </label>
          <input
            type="text"
            required
            minLength={3}
            maxLength={32}
            value={formData.username}
            onChange={(e) =>
              setFormData({ ...formData, username: e.target.value })
            }
            className="mt-1 w-full rounded-md border border-gray-300 px-3 py-2 focus:border-blue-500 focus:outline-none focus:ring-1 focus:ring-blue-500"
            placeholder="请输入用户名"
          />
        </div>

        <div>
          <label className="block text-sm font-medium text-gray-700">
            密码
          </label>
          <input
            type="password"
            required
            minLength={6}
            maxLength={128}
            value={formData.password}
            onChange={(e) =>
              setFormData({ ...formData, password: e.target.value })
            }
            className="mt-1 w-full rounded-md border border-gray-300 px-3 py-2 focus:border-blue-500 focus:outline-none focus:ring-1 focus:ring-blue-500"
            placeholder="请输入密码（至少6位）"
          />
        </div>

        <div>
          <label className="block text-sm font-medium text-gray-700">
            确认密码
          </label>
          <input
            type="password"
            required
            value={confirmPassword}
            onChange={(e) => setConfirmPassword(e.target.value)}
            className="mt-1 w-full rounded-md border border-gray-300 px-3 py-2 focus:border-blue-500 focus:outline-none focus:ring-1 focus:ring-blue-500"
            placeholder="请再次输入密码"
          />
        </div>

        <button
          type="submit"
          disabled={loading}
          className="w-full rounded-md bg-blue-600 py-2 text-white hover:bg-blue-700 focus:outline-none focus:ring-2 focus:ring-blue-500 focus:ring-offset-2 disabled:bg-blue-300"
        >
          {loading ? '注册中...' : '注册'}
        </button>
      </form>

      <p className="mt-4 text-center text-sm text-gray-600">
        已有账号？{' '}
        <button onClick={onSwitch} className="text-blue-600 hover:underline">
          立即登录
        </button>
      </p>
    </div>
  )
}

import { useState, useEffect, useRef } from 'react'
import {
  uploadDatabase,
  downloadDatabase,
  listBackups,
  restoreBackup,
  deleteBackup,
  Backup,
} from '../api/system'

export default function SystemSettings() {
  const [backups, setBackups] = useState<Backup[]>([])
  const [loading, setLoading] = useState(false)
  const [message, setMessage] = useState<{ type: 'success' | 'error'; text: string } | null>(null)
  const fileInputRef = useRef<HTMLInputElement>(null)

  useEffect(() => {
    fetchBackups()
  }, [])

  const fetchBackups = async () => {
    try {
      const data = await listBackups()
      setBackups(data)
    } catch (err) {
      console.error('获取备份列表失败:', err)
    }
  }

  const showMessage = (type: 'success' | 'error', text: string) => {
    setMessage({ type, text })
    setTimeout(() => setMessage(null), 5000)
  }

  const handleUpload = async (e: React.ChangeEvent<HTMLInputElement>) => {
    const file = e.target.files?.[0]
    if (!file) return

    if (!file.name.endsWith('.db')) {
      showMessage('error', '请选择 .db 数据库文件')
      return
    }

    if (!confirm('上传新数据库将覆盖现有数据，确定继续吗？')) {
      if (fileInputRef.current) fileInputRef.current.value = ''
      return
    }

    setLoading(true)
    try {
      const result = await uploadDatabase(file)
      showMessage('success', result.message)
      fetchBackups()
    } catch (err) {
      showMessage('error', err instanceof Error ? err.message : '上传失败')
    } finally {
      setLoading(false)
      if (fileInputRef.current) fileInputRef.current.value = ''
    }
  }

  const handleDownload = async () => {
    setLoading(true)
    try {
      await downloadDatabase()
      showMessage('success', '数据库下载成功')
    } catch (err) {
      showMessage('error', err instanceof Error ? err.message : '下载失败')
    } finally {
      setLoading(false)
    }
  }

  const handleRestore = async (filename: string) => {
    if (!confirm(`确定要恢复备份 ${filename} 吗？当前数据将被备份。`)) {
      return
    }

    setLoading(true)
    try {
      const result = await restoreBackup(filename)
      showMessage('success', result.message)
      fetchBackups()
    } catch (err) {
      showMessage('error', err instanceof Error ? err.message : '恢复失败')
    } finally {
      setLoading(false)
    }
  }

  const handleDelete = async (filename: string) => {
    if (!confirm(`确定要删除备份 ${filename} 吗？`)) {
      return
    }

    setLoading(true)
    try {
      await deleteBackup(filename)
      showMessage('success', '备份已删除')
      fetchBackups()
    } catch (err) {
      showMessage('error', err instanceof Error ? err.message : '删除失败')
    } finally {
      setLoading(false)
    }
  }

  const formatSize = (bytes: number) => {
    if (bytes < 1024) return bytes + ' B'
    if (bytes < 1024 * 1024) return (bytes / 1024).toFixed(1) + ' KB'
    return (bytes / 1024 / 1024).toFixed(1) + ' MB'
  }

  const formatDate = (dateStr: string) => {
    return new Date(dateStr).toLocaleString('zh-CN')
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

      {/* 数据库操作 */}
      <div className="rounded-lg bg-white p-6 shadow-md">
        <h3 className="mb-4 text-lg font-bold text-gray-800">数据库管理</h3>

        <div className="space-y-4">
          <div className="flex flex-wrap gap-4">
            <div>
              <input
                ref={fileInputRef}
                type="file"
                accept=".db"
                onChange={handleUpload}
                className="hidden"
                id="db-upload"
                disabled={loading}
              />
              <label
                htmlFor="db-upload"
                className={`inline-block cursor-pointer rounded-md px-4 py-2 text-white ${
                  loading ? 'bg-gray-400 cursor-not-allowed' : 'bg-blue-600 hover:bg-blue-700'
                }`}
              >
                上传数据库
              </label>
            </div>

            <button
              onClick={handleDownload}
              disabled={loading}
              className={`rounded-md px-4 py-2 text-white ${
                loading ? 'bg-gray-400 cursor-not-allowed' : 'bg-green-600 hover:bg-green-700'
              }`}
            >
              下载当前数据库
            </button>
          </div>

          <p className="text-sm text-gray-500">
            上传新数据库将自动备份当前数据库。更改生效需要重启服务。
          </p>
        </div>
      </div>

      {/* 备份列表 */}
      <div className="rounded-lg bg-white p-6 shadow-md">
        <h3 className="mb-4 text-lg font-bold text-gray-800">备份列表</h3>

        {backups.length === 0 ? (
          <p className="text-gray-500">暂无备份</p>
        ) : (
          <div className="overflow-x-auto">
            <table className="w-full text-left text-sm">
              <thead className="bg-gray-50">
                <tr>
                  <th className="px-4 py-3 font-medium text-gray-700">文件名</th>
                  <th className="px-4 py-3 font-medium text-gray-700">大小</th>
                  <th className="px-4 py-3 font-medium text-gray-700">备份时间</th>
                  <th className="px-4 py-3 font-medium text-gray-700">操作</th>
                </tr>
              </thead>
              <tbody className="divide-y">
                {backups.map((backup) => (
                  <tr key={backup.filename} className="hover:bg-gray-50">
                    <td className="px-4 py-3 font-mono text-xs">{backup.filename}</td>
                    <td className="px-4 py-3">{formatSize(backup.size)}</td>
                    <td className="px-4 py-3">{formatDate(backup.modTime)}</td>
                    <td className="px-4 py-3">
                      <div className="flex gap-2">
                        <button
                          onClick={() => handleRestore(backup.filename)}
                          disabled={loading}
                          className="rounded bg-yellow-500 px-3 py-1 text-xs text-white hover:bg-yellow-600 disabled:bg-gray-400"
                        >
                          恢复
                        </button>
                        <button
                          onClick={() => handleDelete(backup.filename)}
                          disabled={loading}
                          className="rounded bg-red-500 px-3 py-1 text-xs text-white hover:bg-red-600 disabled:bg-gray-400"
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
        )}
      </div>
    </div>
  )
}

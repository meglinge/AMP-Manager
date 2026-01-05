import { useState, useEffect, useRef } from 'react'
import {
  uploadDatabase,
  downloadDatabase,
  listBackups,
  restoreBackup,
  deleteBackup,
  Backup,
} from '../api/system'
import { Button } from '@/components/ui/button'
import { Card, CardHeader, CardTitle, CardDescription, CardContent } from '@/components/ui/card'
import { Alert, AlertDescription } from '@/components/ui/alert'
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table'
import { Badge } from '@/components/ui/badge'

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
        <Alert variant={message.type === 'error' ? 'destructive' : 'default'}>
          <AlertDescription>{message.text}</AlertDescription>
        </Alert>
      )}

      <Card>
        <CardHeader>
          <CardTitle>数据库管理</CardTitle>
          <CardDescription>上传、下载和管理系统数据库</CardDescription>
        </CardHeader>
        <CardContent className="space-y-4">
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
              <Button asChild disabled={loading}>
                <label htmlFor="db-upload" className="cursor-pointer">
                  上传数据库
                </label>
              </Button>
            </div>

            <Button variant="secondary" onClick={handleDownload} disabled={loading}>
              下载当前数据库
            </Button>
          </div>

          <p className="text-sm text-muted-foreground">
            上传新数据库将自动备份当前数据库。更改生效需要重启服务。
          </p>
        </CardContent>
      </Card>

      <Card>
        <CardHeader>
          <CardTitle>备份列表</CardTitle>
          <CardDescription>查看和管理数据库备份</CardDescription>
        </CardHeader>
        <CardContent>
          {backups.length === 0 ? (
            <div className="rounded-md border border-dashed p-8 text-center text-muted-foreground">
              暂无备份
            </div>
          ) : (
            <div className="overflow-x-auto">
              <Table>
                <TableHeader>
                  <TableRow>
                    <TableHead>文件名</TableHead>
                    <TableHead>大小</TableHead>
                    <TableHead>备份时间</TableHead>
                    <TableHead>操作</TableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {backups.map((backup) => (
                    <TableRow key={backup.filename}>
                      <TableCell className="font-mono text-xs">{backup.filename}</TableCell>
                      <TableCell>
                        <Badge variant="outline">{formatSize(backup.size)}</Badge>
                      </TableCell>
                      <TableCell className="text-muted-foreground">
                        {formatDate(backup.modTime)}
                      </TableCell>
                      <TableCell>
                        <div className="flex gap-2">
                          <Button
                            variant="outline"
                            size="sm"
                            onClick={() => handleRestore(backup.filename)}
                            disabled={loading}
                          >
                            恢复
                          </Button>
                          <Button
                            variant="destructive"
                            size="sm"
                            onClick={() => handleDelete(backup.filename)}
                            disabled={loading}
                          >
                            删除
                          </Button>
                        </div>
                      </TableCell>
                    </TableRow>
                  ))}
                </TableBody>
              </Table>
            </div>
          )}
        </CardContent>
      </Card>
    </div>
  )
}

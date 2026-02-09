import { useState, useEffect } from 'react'
import { motion, AnimatePresence } from '@/lib/motion'
import {
  listUsers,
  setUserAdmin,
  deleteUser,
  resetUserPassword,
  UserInfo,
} from '../api/users'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Badge } from '@/components/ui/badge'
import { Alert, AlertDescription } from '@/components/ui/alert'
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'
import { Switch } from '@/components/ui/switch'
import { CheckCircle2, XCircle, Trash2, KeyRound } from 'lucide-react'

export default function UserManagement() {
  const [users, setUsers] = useState<UserInfo[]>([])
  const [loading, setLoading] = useState(true)
  const [message, setMessage] = useState<{ type: 'success' | 'error'; text: string } | null>(null)
  const [resetPasswordModal, setResetPasswordModal] = useState<{ userId: string; username: string } | null>(null)
  const [newPassword, setNewPassword] = useState('')
  const [deleteConfirmModal, setDeleteConfirmModal] = useState<UserInfo | null>(null)

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

  const handleDelete = async () => {
    if (!deleteConfirmModal) return

    try {
      await deleteUser(deleteConfirmModal.id)
      showMessage('success', '用户已删除')
      setDeleteConfirmModal(null)
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
    return <div className="text-center text-muted-foreground">加载中...</div>
  }

  return (
    <motion.div initial={{ opacity: 0 }} animate={{ opacity: 1 }} className="space-y-6">
      <AnimatePresence>
        {message && (
          <motion.div initial={{ opacity: 0, y: -20, scale: 0.95 }} animate={{ opacity: 1, y: 0, scale: 1 }} exit={{ opacity: 0, y: -20, scale: 0.95 }} transition={{ type: 'spring', bounce: 0.3, duration: 0.5 }}>
            <Alert variant={message.type === 'success' ? 'default' : 'destructive'}>
              {message.type === 'success' ? (
                <CheckCircle2 className="h-4 w-4" />
              ) : (
                <XCircle className="h-4 w-4" />
              )}
              <AlertDescription>{message.text}</AlertDescription>
            </Alert>
          </motion.div>
        )}
      </AnimatePresence>

      <motion.div initial={{ opacity: 0, y: 30 }} animate={{ opacity: 1, y: 0 }} transition={{ type: 'spring', bounce: 0.2, duration: 0.6, delay: 0.1 }}>
      <Card>
        <CardHeader>
          <CardTitle>用户列表</CardTitle>
        </CardHeader>
        <CardContent>
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>用户名</TableHead>
                <TableHead>角色</TableHead>
                <TableHead>管理员权限</TableHead>
                <TableHead>创建时间</TableHead>
                <TableHead>操作</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {users.map((user) => (
                <TableRow key={user.id}>
                  <TableCell className="font-medium">{user.username}</TableCell>
                  <TableCell>
                    <Badge variant={user.isAdmin ? 'default' : 'secondary'}>
                      {user.isAdmin ? '管理员' : '普通用户'}
                    </Badge>
                  </TableCell>
                  <TableCell>
                    <div className="flex items-center gap-2">
                      <Switch
                        checked={user.isAdmin}
                        onCheckedChange={() => handleToggleAdmin(user)}
                      />
                      <span className="text-sm text-muted-foreground">
                        {user.isAdmin ? '已启用' : '已禁用'}
                      </span>
                    </div>
                  </TableCell>
                  <TableCell className="text-muted-foreground">
                    {formatDate(user.createdAt)}
                  </TableCell>
                  <TableCell>
                    <div className="flex gap-2">
                      <Button
                        variant="outline"
                        size="sm"
                        onClick={() => setResetPasswordModal({ userId: user.id, username: user.username })}
                      >
                        <KeyRound className="mr-1 h-4 w-4" />
                        重置密码
                      </Button>
                      <Button
                        variant="destructive"
                        size="sm"
                        onClick={() => setDeleteConfirmModal(user)}
                      >
                        <Trash2 className="mr-1 h-4 w-4" />
                        删除
                      </Button>
                    </div>
                  </TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        </CardContent>
      </Card>
      </motion.div>

      <Dialog open={!!resetPasswordModal} onOpenChange={(open) => !open && setResetPasswordModal(null)}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>重置密码</DialogTitle>
            <DialogDescription>
              为用户 <span className="font-medium">{resetPasswordModal?.username}</span> 设置新密码
            </DialogDescription>
          </DialogHeader>
          <div className="space-y-4 py-4">
            <div className="space-y-2">
              <Label htmlFor="newPassword">新密码</Label>
              <Input
                id="newPassword"
                type="password"
                placeholder="至少6位字符"
                value={newPassword}
                onChange={(e) => setNewPassword(e.target.value)}
              />
            </div>
          </div>
          <DialogFooter>
            <Button
              variant="outline"
              onClick={() => {
                setResetPasswordModal(null)
                setNewPassword('')
              }}
            >
              取消
            </Button>
            <Button onClick={handleResetPassword} disabled={newPassword.length < 6}>
              确认重置
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      <Dialog open={!!deleteConfirmModal} onOpenChange={(open) => !open && setDeleteConfirmModal(null)}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>确认删除</DialogTitle>
            <DialogDescription>
              确定要删除用户 <span className="font-medium">{deleteConfirmModal?.username}</span> 吗？此操作不可撤销。
            </DialogDescription>
          </DialogHeader>
          <DialogFooter>
            <Button variant="outline" onClick={() => setDeleteConfirmModal(null)}>
              取消
            </Button>
            <Button variant="destructive" onClick={handleDelete}>
              确认删除
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </motion.div>
  )
}

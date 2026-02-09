import { useState, useEffect } from 'react'
import { motion, AnimatePresence, staggerContainer, staggerItem } from '@/lib/motion'
import {
  listUsers,
  setUserAdmin,
  deleteUser,
  resetUserPassword,
  setUserGroups,
  topUpUser,
  UserInfo,
} from '../api/users'
import { listGroups, Group } from '../api/groups'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Badge } from '@/components/ui/badge'
import { Alert, AlertDescription } from '@/components/ui/alert'
import {
  Table,
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
import { Popover, PopoverContent, PopoverTrigger } from '@/components/ui/popover'
import { Checkbox } from '@/components/ui/checkbox'
import { CheckCircle2, XCircle, Trash2, KeyRound, Wallet } from 'lucide-react'

export default function UserManagement() {
  const [users, setUsers] = useState<UserInfo[]>([])
  const [groups, setGroups] = useState<Group[]>([])
  const [loading, setLoading] = useState(true)
  const [message, setMessage] = useState<{ type: 'success' | 'error'; text: string } | null>(null)
  const [resetPasswordModal, setResetPasswordModal] = useState<{ userId: string; username: string } | null>(null)
  const [newPassword, setNewPassword] = useState('')
  const [deleteConfirmModal, setDeleteConfirmModal] = useState<UserInfo | null>(null)
  const [topUpModal, setTopUpModal] = useState<{ userId: string; username: string } | null>(null)
  const [topUpAmount, setTopUpAmount] = useState('')

  useEffect(() => {
    fetchUsers()
    fetchGroups()
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

  const fetchGroups = async () => {
    try {
      const data = await listGroups()
      setGroups(data)
    } catch {}
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

  const handleTopUp = async () => {
    if (!topUpModal || !topUpAmount) return
    const amount = parseFloat(topUpAmount)
    if (isNaN(amount) || amount <= 0) {
      showMessage('error', '请输入有效金额')
      return
    }
    try {
      await topUpUser(topUpModal.userId, amount)
      showMessage('success', '充值成功')
      setTopUpModal(null)
      setTopUpAmount('')
      fetchUsers()
    } catch (err) {
      showMessage('error', err instanceof Error ? err.message : '充值失败')
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
                <TableHead>分组</TableHead>
                <TableHead>余额 (USD)</TableHead>
                <TableHead>管理员权限</TableHead>
                <TableHead>创建时间</TableHead>
                <TableHead>操作</TableHead>
              </TableRow>
            </TableHeader>
            <motion.tbody key="user-table-body" variants={staggerContainer} initial="hidden" animate="visible">
              {users.map((user) => (
                <motion.tr key={user.id} variants={staggerItem} layout className="border-b transition-colors hover:bg-muted/50 data-[state=selected]:bg-muted">
                  <TableCell className="font-medium">{user.username}</TableCell>
                  <TableCell>
                    <Badge variant={user.isAdmin ? 'default' : 'secondary'}>
                      {user.isAdmin ? '管理员' : '普通用户'}
                    </Badge>
                  </TableCell>
                  <TableCell>
                    <Popover>
                      <PopoverTrigger asChild>
                        <Button variant="outline" size="sm" className="h-auto min-h-[32px] py-1 px-2">
                          {user.groupNames && user.groupNames.length > 0 ? (
                            <div className="flex flex-wrap gap-1">
                              <AnimatePresence mode="popLayout">
                                {user.groupNames.map((name) => (
                                  <motion.span
                                    key={name}
                                    initial={{ opacity: 0, scale: 0.8 }}
                                    animate={{ opacity: 1, scale: 1 }}
                                    exit={{ opacity: 0, scale: 0.8 }}
                                    transition={{ type: 'spring', bounce: 0.3, duration: 0.3 }}
                                  >
                                    <Badge variant="secondary" className="text-xs">{name}</Badge>
                                  </motion.span>
                                ))}
                              </AnimatePresence>
                            </div>
                          ) : (
                            <span className="text-muted-foreground text-xs">未分组</span>
                          )}
                        </Button>
                      </PopoverTrigger>
                      <PopoverContent className="w-[200px] p-2" align="start">
                        <div className="space-y-2">
                          <p className="text-sm font-medium px-1">选择分组</p>
                          {groups.map(g => (
                            <label key={g.id} className="flex items-center gap-2 px-1 py-1 rounded hover:bg-muted cursor-pointer">
                              <Checkbox
                                checked={(user.groupIds || []).includes(g.id)}
                                onCheckedChange={async (checked) => {
                                  const currentIds = user.groupIds || []
                                  const newIds = checked
                                    ? [...currentIds, g.id]
                                    : currentIds.filter(id => id !== g.id)
                                  try {
                                    await setUserGroups(user.id, newIds)
                                    showMessage('success', '分组已更新')
                                    fetchUsers()
                                  } catch (err) {
                                    showMessage('error', err instanceof Error ? err.message : '设置分组失败')
                                  }
                                }}
                              />
                              <span className="text-sm">{g.name}</span>
                            </label>
                          ))}
                          {groups.length === 0 && (
                            <p className="text-xs text-muted-foreground px-1">暂无分组</p>
                          )}
                        </div>
                      </PopoverContent>
                    </Popover>
                  </TableCell>
                  <TableCell className="font-mono text-sm">
                    ${parseFloat(user.balanceUsd || '0').toFixed(4)}
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
                        onClick={() => setTopUpModal({ userId: user.id, username: user.username })}
                      >
                        <Wallet className="mr-1 h-4 w-4" />
                        充值
                      </Button>
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
                </motion.tr>
              ))}
            </motion.tbody>
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

      <Dialog open={!!topUpModal} onOpenChange={(open) => !open && setTopUpModal(null)}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>余额充值</DialogTitle>
            <DialogDescription>
              为用户 <span className="font-medium">{topUpModal?.username}</span> 充值
            </DialogDescription>
          </DialogHeader>
          <div className="space-y-4 py-4">
            <div className="space-y-2">
              <Label htmlFor="topUpAmount">充值金额 (USD)</Label>
              <Input
                id="topUpAmount"
                type="number"
                step="0.01"
                min="0.01"
                placeholder="例如: 10.00"
                value={topUpAmount}
                onChange={(e) => setTopUpAmount(e.target.value)}
              />
            </div>
          </div>
          <DialogFooter>
            <Button
              variant="outline"
              onClick={() => {
                setTopUpModal(null)
                setTopUpAmount('')
              }}
            >
              取消
            </Button>
            <Button onClick={handleTopUp} disabled={!topUpAmount || parseFloat(topUpAmount) <= 0}>
              确认充值
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </motion.div>
  )
}

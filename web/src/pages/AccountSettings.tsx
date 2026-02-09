import { useState } from 'react'
import { motion, AnimatePresence } from '@/lib/motion'
import { changePassword, changeUsername } from '../api/users'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Alert, AlertDescription } from '@/components/ui/alert'
import { CheckCircle2, XCircle } from 'lucide-react'

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

      <motion.div initial={{ opacity: 0, y: 30 }} animate={{ opacity: 1, y: 0 }} transition={{ type: 'spring', bounce: 0.25, duration: 0.6, delay: 0.1 }}>
      <Card>
        <CardHeader>
          <CardTitle>修改密码</CardTitle>
          <CardDescription>更新您的账户密码</CardDescription>
        </CardHeader>
        <CardContent>
          <form onSubmit={handleChangePassword} className="space-y-4">
            <div className="space-y-2">
              <Label htmlFor="oldPassword">当前密码</Label>
              <Input
                id="oldPassword"
                type="password"
                value={oldPassword}
                onChange={(e) => setOldPassword(e.target.value)}
                required
              />
            </div>
            <div className="space-y-2">
              <Label htmlFor="newPassword">新密码</Label>
              <Input
                id="newPassword"
                type="password"
                value={newPassword}
                onChange={(e) => setNewPassword(e.target.value)}
                required
                minLength={6}
              />
            </div>
            <div className="space-y-2">
              <Label htmlFor="confirmPassword">确认新密码</Label>
              <Input
                id="confirmPassword"
                type="password"
                value={confirmPassword}
                onChange={(e) => setConfirmPassword(e.target.value)}
                required
              />
            </div>
            <Button type="submit" disabled={changingPassword}>
              {changingPassword ? '修改中...' : '修改密码'}
            </Button>
          </form>
        </CardContent>
      </Card>
      </motion.div>

      <motion.div initial={{ opacity: 0, y: 30 }} animate={{ opacity: 1, y: 0 }} transition={{ type: 'spring', bounce: 0.25, duration: 0.6, delay: 0.2 }}>
      <Card>
        <CardHeader>
          <CardTitle>修改用户名</CardTitle>
          <CardDescription>当前用户名: {username}</CardDescription>
        </CardHeader>
        <CardContent>
          <form onSubmit={handleChangeUsername} className="space-y-4">
            <div className="space-y-2">
              <Label htmlFor="newUsername">新用户名</Label>
              <Input
                id="newUsername"
                type="text"
                value={newUsername}
                onChange={(e) => setNewUsername(e.target.value)}
                required
                minLength={3}
                maxLength={32}
              />
            </div>
            <Button type="submit" disabled={changingUsername}>
              {changingUsername ? '修改中...' : '修改用户名'}
            </Button>
          </form>
        </CardContent>
      </Card>
      </motion.div>
    </motion.div>
  )
}

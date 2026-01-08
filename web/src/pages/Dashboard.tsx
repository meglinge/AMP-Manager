import { useState } from 'react'
import AmpSettings from './AmpSettings'
import APIKeys from './APIKeys'
import RequestLogs from './RequestLogs'
import AdminRequestLogs from './AdminRequestLogs'
import Channels from './Channels'
import Models from './Models'
import ModelMetadata from './ModelMetadata'
import Prices from './Prices'
import SystemSettings from './SystemSettings'
import UserManagement from './UserManagement'
import AccountSettings from './AccountSettings'
import { Button } from '@/components/ui/button'
import { Card, CardContent } from '@/components/ui/card'
import { Badge } from '@/components/ui/badge'
import { Separator } from '@/components/ui/separator'
import {
  DropdownMenu,
  DropdownMenuTrigger,
  DropdownMenuContent,
  DropdownMenuItem,
} from '@/components/ui/dropdown-menu'

interface Props {
  username: string
  isAdmin: boolean
  onLogout: () => void
}

type Page = 'amp-settings' | 'api-keys' | 'request-logs' | 'channels' | 'models' | 'model-metadata' | 'prices' | 'system-settings' | 'user-management' | 'account-settings' | 'admin-request-logs'

export default function Dashboard({ username: initialUsername, isAdmin, onLogout }: Props) {
  const [currentPage, setCurrentPage] = useState<Page>('amp-settings')
  const [username, setUsername] = useState(initialUsername)

  const navItems: { key: Page; label: string; adminOnly?: boolean }[] = [
    { key: 'amp-settings', label: 'Amp 设置' },
    { key: 'api-keys', label: 'API Key 管理' },
    { key: 'request-logs', label: '请求日志' },
    { key: 'models', label: '可用模型' },
    { key: 'account-settings', label: '账户设置' },
    { key: 'channels', label: '渠道管理', adminOnly: true },
    { key: 'model-metadata', label: '模型元数据', adminOnly: true },
    { key: 'prices', label: '模型价格', adminOnly: true },
    { key: 'admin-request-logs', label: '全局日志', adminOnly: true },
    { key: 'user-management', label: '用户管理', adminOnly: true },
    { key: 'system-settings', label: '系统设置', adminOnly: true },
  ]

  const visibleNavItems = navItems.filter(item => !item.adminOnly || isAdmin)

  return (
    <div className="flex min-h-screen bg-muted/40">
      {/* 侧边栏 */}
      <aside className="w-64 border-r bg-background">
        <div className="flex h-16 items-center justify-center border-b px-4">
          <h1 className="text-xl font-bold tracking-tight">AMPManager</h1>
        </div>

        <nav className="flex flex-col gap-1 p-4">
          {visibleNavItems.map((item) => (
            <Button
              key={item.key}
              variant={currentPage === item.key ? 'default' : 'ghost'}
              className="w-full justify-start gap-2"
              onClick={() => setCurrentPage(item.key)}
            >
              {item.label}
              {item.adminOnly && (
                <Badge variant="secondary" className="ml-auto text-xs">
                  管理员
                </Badge>
              )}
            </Button>
          ))}
        </nav>
      </aside>

      {/* 主内容区 */}
      <div className="flex-1 flex flex-col">
        {/* 顶部导航 */}
        <header className="flex h-16 items-center justify-between border-b bg-background px-6">
          <h2 className="text-lg font-semibold">
            {visibleNavItems.find((item) => item.key === currentPage)?.label}
          </h2>
          <div className="flex items-center gap-3">
            <DropdownMenu>
              <DropdownMenuTrigger asChild>
                <Button variant="outline" className="gap-2">
                  <span>{username}</span>
                  {isAdmin && (
                    <Badge variant="default" className="text-xs">
                      管理员
                    </Badge>
                  )}
                </Button>
              </DropdownMenuTrigger>
              <DropdownMenuContent align="end" className="w-48">
                <DropdownMenuItem onClick={() => setCurrentPage('account-settings')}>
                  账户设置
                </DropdownMenuItem>
                <Separator className="my-1" />
                <DropdownMenuItem onClick={onLogout} className="text-destructive">
                  退出登录
                </DropdownMenuItem>
              </DropdownMenuContent>
            </DropdownMenu>
          </div>
        </header>

        {/* 页面内容 */}
        <main className="flex-1 p-6">
          <Card className="h-full">
            <CardContent className="p-6">
              {currentPage === 'amp-settings' && <AmpSettings />}
              {currentPage === 'api-keys' && <APIKeys />}
              {currentPage === 'request-logs' && <RequestLogs />}
              {currentPage === 'models' && <Models isAdmin={isAdmin} />}
              {currentPage === 'account-settings' && <AccountSettings username={username} onUsernameChange={setUsername} />}
              {currentPage === 'channels' && isAdmin && <Channels />}
              {currentPage === 'model-metadata' && isAdmin && <ModelMetadata />}
              {currentPage === 'prices' && isAdmin && <Prices />}
              {currentPage === 'admin-request-logs' && isAdmin && <AdminRequestLogs />}
              {currentPage === 'user-management' && isAdmin && <UserManagement />}
              {currentPage === 'system-settings' && isAdmin && <SystemSettings />}
            </CardContent>
          </Card>
        </main>
      </div>
    </div>
  )
}

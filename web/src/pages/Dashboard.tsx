import { useState } from 'react'
import AmpSettings from './AmpSettings'
import APIKeys from './APIKeys'
import Channels from './Channels'
import Models from './Models'
import ModelMetadata from './ModelMetadata'

interface Props {
  username: string
  isAdmin: boolean
  onLogout: () => void
}

type Page = 'amp-settings' | 'api-keys' | 'channels' | 'models' | 'model-metadata'

export default function Dashboard({ username, isAdmin, onLogout }: Props) {
  const [currentPage, setCurrentPage] = useState<Page>('amp-settings')

  const navItems: { key: Page; label: string; adminOnly?: boolean }[] = [
    { key: 'amp-settings', label: 'Amp 设置' },
    { key: 'api-keys', label: 'API Key 管理' },
    { key: 'models', label: '可用模型' },
    { key: 'channels', label: '渠道管理', adminOnly: true },
    { key: 'model-metadata', label: '模型元数据', adminOnly: true },
  ]

  const visibleNavItems = navItems.filter(item => !item.adminOnly || isAdmin)

  return (
    <div className="flex min-h-screen bg-gray-100">
      {/* 侧边栏 */}
      <aside className="w-64 bg-white shadow-md">
        <div className="flex h-16 items-center justify-center border-b">
          <h1 className="text-xl font-bold text-gray-800">AMPManager</h1>
        </div>

        <nav className="p-4">
          <ul className="space-y-2">
            {visibleNavItems.map((item) => (
              <li key={item.key}>
                <button
                  onClick={() => setCurrentPage(item.key)}
                  className={`w-full rounded-md px-4 py-2 text-left transition-colors ${
                    currentPage === item.key
                      ? 'bg-blue-600 text-white'
                      : 'text-gray-700 hover:bg-gray-100'
                  }`}
                >
                  {item.label}
                  {item.adminOnly && (
                    <span className="ml-2 text-xs opacity-70">(管理员)</span>
                  )}
                </button>
              </li>
            ))}
          </ul>
        </nav>
      </aside>

      {/* 主内容区 */}
      <div className="flex-1">
        {/* 顶部导航 */}
        <header className="flex h-16 items-center justify-between border-b bg-white px-6 shadow-sm">
          <h2 className="text-lg font-medium text-gray-800">
            {visibleNavItems.find((item) => item.key === currentPage)?.label}
          </h2>
          <div className="flex items-center gap-4">
            <span className="text-sm text-gray-600">
              欢迎, {username}
              {isAdmin && <span className="ml-1 text-blue-600">(管理员)</span>}
            </span>
            <button
              onClick={onLogout}
              className="rounded-md bg-gray-600 px-4 py-2 text-sm text-white hover:bg-gray-700"
            >
              退出登录
            </button>
          </div>
        </header>

        {/* 页面内容 */}
        <main className="p-6">
          {currentPage === 'amp-settings' && <AmpSettings />}
          {currentPage === 'api-keys' && <APIKeys />}
          {currentPage === 'models' && <Models isAdmin={isAdmin} />}
          {currentPage === 'channels' && isAdmin && <Channels />}
          {currentPage === 'model-metadata' && isAdmin && <ModelMetadata />}
        </main>
      </div>
    </div>
  )
}

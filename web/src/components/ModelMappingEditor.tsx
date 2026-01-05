import { useState, useEffect, useRef } from 'react'
import { ModelMapping } from '../api/amp'
import { listAvailableModels, AvailableModel } from '../api/models'

interface Props {
  mappings: ModelMapping[]
  onChange: (mappings: ModelMapping[]) => void
}

const THINKING_LEVELS = [
  { value: '', label: '无' },
  { value: 'low', label: 'Low' },
  { value: 'medium', label: 'Medium' },
  { value: 'high', label: 'High' },
  { value: 'xhigh', label: 'XHigh' },
]

export default function ModelMappingEditor({ mappings, onChange }: Props) {
  const [availableModels, setAvailableModels] = useState<AvailableModel[]>([])
  const [loadingModels, setLoadingModels] = useState(false)
  const [showDropdown, setShowDropdown] = useState<number | null>(null)
  const [searchTerm, setSearchTerm] = useState('')
  const [dropdownPos, setDropdownPos] = useState({ top: 0, left: 0 })
  const dropdownRefs = useRef<(HTMLTableCellElement | null)[]>([])

  useEffect(() => {
    loadModels()
  }, [])

  useEffect(() => {
    const handleClickOutside = (event: MouseEvent) => {
      if (showDropdown !== null) {
        const dropdown = document.getElementById('model-dropdown')
        const currentRef = dropdownRefs.current[showDropdown]
        if (dropdown && !dropdown.contains(event.target as Node) && 
            currentRef && !currentRef.contains(event.target as Node)) {
          setShowDropdown(null)
          setSearchTerm('')
        }
      }
    }
    const handleScroll = (event: Event) => {
      if (showDropdown !== null) {
        const dropdown = document.getElementById('model-dropdown')
        if (dropdown && dropdown.contains(event.target as Node)) {
          return
        }
        setShowDropdown(null)
        setSearchTerm('')
      }
    }
    document.addEventListener('mousedown', handleClickOutside)
    window.addEventListener('scroll', handleScroll, true)
    return () => {
      document.removeEventListener('mousedown', handleClickOutside)
      window.removeEventListener('scroll', handleScroll, true)
    }
  }, [showDropdown])

  const openDropdown = (index: number) => {
    const ref = dropdownRefs.current[index]
    if (ref) {
      const rect = ref.getBoundingClientRect()
      setDropdownPos({ top: rect.bottom, left: rect.left })
    }
    setShowDropdown(index)
  }

  const loadModels = async () => {
    try {
      setLoadingModels(true)
      const models = await listAvailableModels()
      setAvailableModels(models)
    } catch {
      // Silently fail - models will just not be available for selection
    } finally {
      setLoadingModels(false)
    }
  }

  const handleAdd = () => {
    onChange([...mappings, { from: '', to: '', regex: false, thinkingLevel: '' }])
  }

  const handleRemove = (index: number) => {
    onChange(mappings.filter((_, i) => i !== index))
  }

  const handleChange = (index: number, field: keyof ModelMapping, value: string | boolean) => {
    const newMappings = [...mappings]
    newMappings[index] = { ...newMappings[index], [field]: value }
    onChange(newMappings)
  }

  const handleSelectModel = (index: number, modelId: string) => {
    handleChange(index, 'to', modelId)
    setShowDropdown(null)
    setSearchTerm('')
  }

  const filteredModels = searchTerm
    ? availableModels.filter(m => 
        m.modelId.toLowerCase().includes(searchTerm.toLowerCase()) ||
        m.displayName.toLowerCase().includes(searchTerm.toLowerCase())
      )
    : availableModels

  const groupedModels = filteredModels.reduce((acc, model) => {
    const key = model.channelType
    if (!acc[key]) acc[key] = []
    acc[key].push(model)
    return acc
  }, {} as Record<string, AvailableModel[]>)

  return (
    <div className="space-y-3">
      <div className="flex items-center justify-between">
        <label className="block text-sm font-medium text-gray-700">
          Model Mappings
        </label>
        <button
          type="button"
          onClick={handleAdd}
          className="rounded-md bg-green-600 px-3 py-1 text-sm text-white hover:bg-green-700"
        >
          + 添加映射
        </button>
      </div>

      {mappings.length === 0 ? (
        <div className="rounded-md border border-dashed border-gray-300 p-4 text-center text-sm text-gray-500">
          暂无模型映射，点击上方按钮添加
        </div>
      ) : (
        <div className="rounded-md border border-gray-200">
          <table className="min-w-full divide-y divide-gray-200">
            <thead className="bg-gray-50">
              <tr>
                <th className="px-3 py-2 text-left text-xs font-medium uppercase text-gray-500">
                  From
                </th>
                <th className="px-3 py-2 text-left text-xs font-medium uppercase text-gray-500">
                  To
                </th>
                <th className="px-3 py-2 text-center text-xs font-medium uppercase text-gray-500">
                  思维强度
                </th>
                <th className="px-3 py-2 text-center text-xs font-medium uppercase text-gray-500">
                  Regex
                </th>
                <th className="px-3 py-2 text-center text-xs font-medium uppercase text-gray-500">
                  操作
                </th>
              </tr>
            </thead>
            <tbody className="divide-y divide-gray-200 bg-white">
              {mappings.map((mapping, index) => (
                <tr key={index}>
                  <td className="px-3 py-2">
                    <input
                      type="text"
                      value={mapping.from}
                      onChange={(e) => handleChange(index, 'from', e.target.value)}
                      placeholder="源模型名"
                      className="w-full rounded border border-gray-300 px-2 py-1 text-sm focus:border-blue-500 focus:outline-none"
                    />
                  </td>
                  <td 
                    className="relative px-3 py-2"
                    ref={el => { dropdownRefs.current[index] = el }}
                  >
                    <div className="flex items-center gap-1">
                      <input
                        type="text"
                        value={mapping.to}
                        onChange={(e) => handleChange(index, 'to', e.target.value)}
                        onFocus={() => {
                          if (availableModels.length > 0) {
                            openDropdown(index)
                          }
                        }}
                        placeholder="目标模型名"
                        className="w-full rounded border border-gray-300 px-2 py-1 text-sm focus:border-blue-500 focus:outline-none"
                      />
                      {availableModels.length > 0 && (
                        <button
                          type="button"
                          onClick={() => showDropdown === index ? setShowDropdown(null) : openDropdown(index)}
                          className="rounded border border-gray-300 px-2 py-1 text-gray-500 hover:bg-gray-100"
                        >
                          ▼
                        </button>
                      )}
                    </div>
                    
                    {showDropdown === index && (
                      <div 
                        id="model-dropdown"
                        className="fixed z-50 max-h-80 w-96 overflow-auto rounded-md border border-gray-200 bg-white shadow-xl"
                        style={{ top: dropdownPos.top, left: dropdownPos.left }}
                      >
                        <div className="sticky top-0 bg-white p-2">
                          <input
                            type="text"
                            value={searchTerm}
                            onChange={(e) => setSearchTerm(e.target.value)}
                            placeholder="搜索模型..."
                            className="w-full rounded border border-gray-300 px-2 py-1 text-sm"
                            autoFocus
                          />
                        </div>
                        {loadingModels ? (
                          <div className="p-3 text-center text-sm text-gray-500">加载中...</div>
                        ) : Object.keys(groupedModels).length === 0 ? (
                          <div className="p-3 text-center text-sm text-gray-500">无匹配模型</div>
                        ) : (
                          Object.entries(groupedModels).map(([type, models]) => (
                            <div key={type}>
                              <div className="bg-gray-100 px-3 py-1 text-xs font-medium uppercase text-gray-600">
                                {type}
                              </div>
                              {models.map((model) => (
                                <button
                                  key={`${model.channelName}-${model.modelId}`}
                                  type="button"
                                  onClick={() => handleSelectModel(index, model.modelId)}
                                  className="block w-full px-3 py-2 text-left text-sm hover:bg-blue-50"
                                >
                                  <div className="font-mono text-gray-900">{model.modelId}</div>
                                  <div className="text-xs text-gray-500">{model.channelName}</div>
                                </button>
                              ))}
                            </div>
                          ))
                        )}
                      </div>
                    )}
                  </td>
                  <td className="px-3 py-2">
                    <select
                      value={mapping.thinkingLevel || ''}
                      onChange={(e) => handleChange(index, 'thinkingLevel', e.target.value)}
                      className="w-full rounded border border-gray-300 px-2 py-1 text-sm focus:border-blue-500 focus:outline-none"
                    >
                      {THINKING_LEVELS.map((level) => (
                        <option key={level.value} value={level.value}>
                          {level.label}
                        </option>
                      ))}
                    </select>
                  </td>
                  <td className="px-3 py-2 text-center">
                    <input
                      type="checkbox"
                      checked={mapping.regex}
                      onChange={(e) => handleChange(index, 'regex', e.target.checked)}
                      className="h-4 w-4 rounded border-gray-300 text-blue-600 focus:ring-blue-500"
                    />
                  </td>
                  <td className="px-3 py-2 text-center">
                    <button
                      type="button"
                      onClick={() => handleRemove(index)}
                      className="text-red-600 hover:text-red-800"
                    >
                      删除
                    </button>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}

      <div className="rounded bg-gray-50 p-3 text-xs text-gray-600">
        <p><strong>说明:</strong></p>
        <ul className="mt-1 list-inside list-disc space-y-1">
          <li><strong>From:</strong> 请求中的模型名称（支持正则表达式）</li>
          <li><strong>To:</strong> 映射到的目标模型（可从列表选择或手动输入）</li>
          <li><strong>思维强度:</strong> 设置模型的推理/思考强度 (low/medium/high/xhigh)</li>
          <li><strong>Regex:</strong> 是否将 From 字段作为正则表达式匹配</li>
        </ul>
      </div>
    </div>
  )
}

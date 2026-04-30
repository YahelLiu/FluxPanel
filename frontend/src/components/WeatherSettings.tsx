import { useEffect, useState } from 'react'
import { Send, Save } from 'lucide-react'
import { Card, CardContent } from '@/components/ui/card'

interface WeatherConfig {
  id?: number
  api_key: string
  api_host: string
  enabled: boolean
}

interface WeatherSettingsProps {
  channels?: NotificationChannel[]
}

interface NotificationChannel {
  id: number
  name: string
  type: string
  enabled: boolean
}

export function WeatherSettings(_props: WeatherSettingsProps) {
  const [config, setConfig] = useState<WeatherConfig>({
    api_key: '',
    api_host: 'devapi.qweather.com',
    enabled: true,
  })
  const [testLocation, setTestLocation] = useState('北京')
  const [testing, setTesting] = useState(false)
  const [saving, setSaving] = useState(false)

  useEffect(() => {
    fetchConfig()
  }, [])

  const fetchConfig = async () => {
    try {
      const res = await fetch('/api/weather/config')
      const data = await res.json()
      setConfig({
        api_key: data.api_key || '',
        api_host: data.api_host || 'devapi.qweather.com',
        enabled: data.enabled ?? true,
      })
    } catch (error) {
      console.error('Failed to fetch weather config:', error)
    }
  }

  const saveConfig = async () => {
    setSaving(true)
    try {
      const res = await fetch('/api/weather/config', {
        method: 'PUT',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(config)
      })
      if (res.ok) {
        alert('保存成功')
      } else {
        const data = await res.json()
        alert(`保存失败: ${data.error}`)
      }
    } catch (error) {
      alert('保存失败')
    }
    setSaving(false)
  }

  const testWeather = async () => {
    setTesting(true)
    try {
      const res = await fetch('/api/weather/test', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          api_key: config.api_key,
          api_host: config.api_host,
          location: testLocation
        })
      })
      const data = await res.json()
      if (data.success) {
        const weather = data.weather
        const today = weather.daily?.[0]
        alert(`测试成功！\n${today?.fx_date}\n温度: ${today?.temp_min}°C ~ ${today?.temp_max}°C\n白天: ${today?.text_day}\n夜间: ${today?.text_night}`)
      } else {
        alert(`测试失败: ${data.error}`)
      }
    } catch (error) {
      alert('测试失败')
    }
    setTesting(false)
  }

  return (
    <div className="space-y-6">
      {/* 配置说明 */}
      <div className="p-4 bg-blue-50 dark:bg-blue-900/20 rounded-lg">
        <h4 className="font-medium text-blue-800 dark:text-blue-300 mb-2">🌤️ 天气推送说明</h4>
        <ul className="text-sm text-blue-700 dark:text-blue-400 space-y-1">
          <li>• 使用和风天气 API 获取天气预报数据</li>
          <li>• 天气推送会发送到已登录的微信 iLink</li>
          <li>• 在客户端卡片上为每个设备单独开启天气推送</li>
          <li>• 需要先在和风天气开放平台注册获取 API Key</li>
        </ul>
      </div>

      {/* API 配置 */}
      <Card>
        <CardContent className="p-4 bg-white dark:bg-gray-800 space-y-4">
          <h3 className="font-medium text-gray-900 dark:text-gray-100">API 配置</h3>

          <div className="grid grid-cols-2 gap-4">
            <div>
              <label className="block text-sm font-medium mb-1 text-gray-700 dark:text-gray-300">API Key *</label>
              <input
                type="text"
                value={config.api_key}
                onChange={e => setConfig({ ...config, api_key: e.target.value })}
                className="w-full px-3 py-2 border rounded-md bg-white dark:bg-gray-700 text-gray-900 dark:text-gray-100 border-gray-300 dark:border-gray-600"
                placeholder="在和风天气开放平台获取"
              />
            </div>
            <div>
              <label className="block text-sm font-medium mb-1 text-gray-700 dark:text-gray-300">API Host</label>
              <input
                type="text"
                value={config.api_host}
                onChange={e => setConfig({ ...config, api_host: e.target.value })}
                className="w-full px-3 py-2 border rounded-md bg-white dark:bg-gray-700 text-gray-900 dark:text-gray-100 border-gray-300 dark:border-gray-600"
                placeholder="devapi.qweather.com"
              />
            </div>
          </div>

          <div className="flex items-center gap-4">
            <label className="flex items-center gap-2">
              <input
                type="checkbox"
                checked={config.enabled}
                onChange={e => setConfig({ ...config, enabled: e.target.checked })}
                className="rounded"
              />
              <span className="text-sm text-gray-700 dark:text-gray-300">启用定时推送</span>
            </label>
            <button
              onClick={saveConfig}
              disabled={saving}
              className="flex items-center gap-1 px-4 py-2 bg-primary text-white rounded-md hover:bg-primary/90 disabled:opacity-50"
            >
              <Save className="h-4 w-4" />
              {saving ? '保存中...' : '保存配置'}
            </button>
          </div>
        </CardContent>
      </Card>

      {/* 测试配置 */}
      <Card>
        <CardContent className="p-4 bg-white dark:bg-gray-800 space-y-4">
          <h3 className="font-medium text-gray-900 dark:text-gray-100">测试 API 连接</h3>
          <div className="flex gap-4">
            <input
              type="text"
              value={testLocation}
              onChange={e => setTestLocation(e.target.value)}
              className="flex-1 px-3 py-2 border rounded-md bg-white dark:bg-gray-700 text-gray-900 dark:text-gray-100 border-gray-300 dark:border-gray-600"
              placeholder="输入城市名测试"
            />
            <button
              onClick={testWeather}
              disabled={testing || !config.api_key}
              className="flex items-center gap-1 px-4 py-2 bg-blue-500 text-white rounded-md hover:bg-blue-600 disabled:opacity-50"
            >
              <Send className="h-4 w-4" />
              {testing ? '测试中...' : '测试'}
            </button>
          </div>
        </CardContent>
      </Card>
    </div>
  )
}

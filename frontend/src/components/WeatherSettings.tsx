import { useEffect, useState } from 'react'
import { Send, Save } from 'lucide-react'

interface WeatherConfig {
  id?: number
  api_key: string
  api_host: string
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
    <div className="space-y-4">
      {/* 说明 */}
      <p className="text-sm text-gray-600 dark:text-gray-400">
        配置和风天气 API，用于获取天气预报数据。需要先在<a href="https://dev.qweather.com/" target="_blank" rel="noopener noreferrer" className="text-blue-500 hover:underline">和风天气开放平台</a>注册获取 API Key。
      </p>

      {/* 配置表单 */}
      <div className="border rounded-lg p-4 bg-white dark:bg-gray-800 border-gray-200 dark:border-gray-700 space-y-4">
        <span className="font-medium">API 配置</span>

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

        <div className="flex items-center gap-4 pt-2">
          <button
            onClick={saveConfig}
            disabled={saving}
            className="flex items-center gap-1 px-4 py-2 bg-primary text-white rounded-md hover:bg-primary/90 disabled:opacity-50"
          >
            <Save className="h-4 w-4" />
            {saving ? '保存中...' : '保存配置'}
          </button>
        </div>
      </div>

      {/* 测试 */}
      <div className="border rounded-lg p-4 bg-white dark:bg-gray-800 border-gray-200 dark:border-gray-700 space-y-3">
        <span className="font-medium">测试 API 连接</span>
        <div className="flex gap-2">
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
      </div>
    </div>
  )
}

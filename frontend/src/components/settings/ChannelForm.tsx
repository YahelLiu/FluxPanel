import { useEffect, useState } from 'react'
import { Check, QrCode } from 'lucide-react'
import QRCode from 'qrcode'
import type { NotificationChannel } from './ChannelsTab'

interface ChannelFormProps {
  channel: NotificationChannel | null
  onClose: () => void
  onSave: () => void
}

export function ChannelForm({ channel, onClose, onSave }: ChannelFormProps) {
  const [formData, setFormData] = useState({
    name: channel?.name || '',
    type: channel?.type || 'feishu',
    enabled: channel?.enabled ?? true,
    feishu: {
      webhook_url: channel?.feishu?.webhook_url || ''
    }
  })
  const [qrCodeUrl, setQrCodeUrl] = useState<string | null>(null)
  const [qrCodeDataUrl, setQrCodeDataUrl] = useState<string | null>(null)
  const [qrCodeKey, setQrCodeKey] = useState<string | null>(null)
  const [isLoggedIn, setIsLoggedIn] = useState(channel?.wechat_ilink?.logged_in || false)
  const [pollingStatus, setPollingStatus] = useState(false)

  useEffect(() => {
    if (qrCodeUrl) {
      QRCode.toDataURL(qrCodeUrl, {
        width: 200,
        margin: 2,
        color: { dark: '#000000', light: '#ffffff' }
      }).then(setQrCodeDataUrl).catch(console.error)
    } else {
      setQrCodeDataUrl(null)
    }
  }, [qrCodeUrl])

  const fetchQRCode = async () => {
    try {
      const res = await fetch('/api/notifications/channels/wechat-ilink/qrcode')
      const data = await res.json()
      setQrCodeUrl(data.qrcode_url)
      setQrCodeKey(data.qrcode)
      setPollingStatus(true)
    } catch (error) {
      console.error('Failed to fetch QR code:', error)
      alert('获取二维码失败')
    }
  }

  useEffect(() => {
    if (!pollingStatus || !qrCodeKey) return

    const poll = async () => {
      try {
        const res = await fetch(`/api/notifications/channels/wechat-ilink/status?qrcode=${qrCodeKey}`)
        const data = await res.json()
        if (data.status === 'success') {
          setPollingStatus(false)
          setIsLoggedIn(true)
          setQrCodeUrl(null)
          setQrCodeDataUrl(null)

          // 自动保存微信 iLink 渠道
          const createRes = await fetch('/api/notifications/channels', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({
              name: formData.name || '微信 iLink',
              type: 'wechat_ilink',
              mode: 'app',
              enabled: true
            })
          })
          if (createRes.ok) {
            alert('微信 iLink 登录成功，渠道已保存')
            onSave()
            onClose()
          } else {
            const errData = await createRes.json()
            alert(`保存失败: ${errData.error}`)
          }
        }
      } catch (error) {
        // Continue polling on error
      }
    }

    const interval = setInterval(poll, 2000)
    return () => clearInterval(interval)
  }, [pollingStatus, qrCodeKey, formData.name, onSave, onClose])

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    const payload: any = {
      name: formData.name,
      type: formData.type,
      mode: 'webhook',
      enabled: formData.enabled,
      feishu: {
        webhook_url: formData.feishu.webhook_url || undefined
      }
    }

    const url = channel ? `/api/notifications/channels/${channel.id}` : '/api/notifications/channels'
    const res = await fetch(url, {
      method: channel ? 'PUT' : 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(payload)
    })
    if (res.ok) onSave()
    else alert(`保存失败: ${(await res.json()).error}`)
  }

  return (
    <div className="fixed inset-0 bg-black/50 flex items-center justify-center z-50">
      <div className="bg-white dark:bg-gray-900 rounded-lg shadow-xl w-full max-w-lg border dark:border-gray-700">
        <form onSubmit={handleSubmit}>
          <div className="p-4 border-b dark:border-gray-700">
            <h3 className="text-lg font-semibold text-gray-900 dark:text-gray-100">
              {channel ? '编辑通知渠道' : '添加通知渠道'}
            </h3>
          </div>
          <div className="p-4 space-y-4 bg-white dark:bg-gray-900">
            <div>
              <label className="block text-sm font-medium mb-1 text-gray-700 dark:text-gray-300">
                名称 *
              </label>
              <input
                type="text"
                value={formData.name}
                onChange={e => setFormData({ ...formData, name: e.target.value })}
                className="w-full px-3 py-2 border rounded-md bg-white dark:bg-gray-800 text-gray-900 dark:text-gray-100 border-gray-300 dark:border-gray-600"
                placeholder="如: 运维告警群"
                required
              />
            </div>
            <div>
              <label className="block text-sm font-medium mb-1 text-gray-700 dark:text-gray-300">
                渠道类型
              </label>
              <select
                value={formData.type}
                onChange={e => setFormData({ ...formData, type: e.target.value as any })}
                className="w-full px-3 py-2 border rounded-md bg-white dark:bg-gray-800 text-gray-900 dark:text-gray-100 border-gray-300 dark:border-gray-600"
              >
                <option value="feishu">飞书群机器人</option>
                <option value="wechat_ilink">微信 iLink (扫码登录)</option>
              </select>
            </div>

            {formData.type === 'wechat_ilink' && (
              <div className="p-4 bg-gray-50 dark:bg-gray-800 rounded-lg border border-gray-200 dark:border-gray-700">
                {isLoggedIn ? (
                  <div className="flex items-center gap-2 text-green-600 dark:text-green-400">
                    <Check className="h-5 w-5" />
                    <span>已登录微信</span>
                  </div>
                ) : qrCodeDataUrl ? (
                  <div className="text-center">
                    <p className="text-sm text-gray-600 dark:text-gray-400 mb-3">请使用微信扫码登录</p>
                    <img src={qrCodeDataUrl} alt="微信登录二维码" className="mx-auto border-4 border-white shadow-lg" />
                    <p className="text-xs text-gray-500 dark:text-gray-400 mt-2">正在等待扫码...</p>
                  </div>
                ) : (
                  <div className="text-center">
                    <p className="text-sm text-gray-600 dark:text-gray-400 mb-3">微信 iLink 需要扫码登录</p>
                    <button
                      type="button"
                      onClick={fetchQRCode}
                      className="inline-flex items-center gap-2 px-4 py-2 bg-green-600 text-white rounded-md hover:bg-green-700"
                    >
                      <QrCode className="h-4 w-4" />
                      获取登录二维码
                    </button>
                  </div>
                )}
              </div>
            )}

            {formData.type === 'feishu' && (
              <div>
                <label className="block text-sm font-medium mb-1 text-gray-700 dark:text-gray-300">
                  Webhook 地址 *
                </label>
                <input
                  type="url"
                  value={formData.feishu.webhook_url}
                  onChange={e => setFormData({ ...formData, feishu: { ...formData.feishu, webhook_url: e.target.value } })}
                  className="w-full px-3 py-2 border rounded-md bg-white dark:bg-gray-800 text-gray-900 dark:text-gray-100 border-gray-300 dark:border-gray-600"
                  placeholder="https://open.feishu.cn/open-apis/bot/v2/hook/..."
                  required
                />
              </div>
            )}

            <div className="flex items-center gap-2">
              <input
                type="checkbox"
                id="enabled"
                checked={formData.enabled}
                onChange={e => setFormData({ ...formData, enabled: e.target.checked })}
                className="rounded"
              />
              <label htmlFor="enabled" className="text-sm text-gray-700 dark:text-gray-300">启用</label>
            </div>
          </div>

          {/* 飞书需要保存按钮，微信 iLink 扫码后自动保存 */}
          {formData.type === 'feishu' && (
            <div className="p-4 border-t dark:border-gray-700 flex justify-end gap-2">
              <button
                type="button"
                onClick={onClose}
                className="px-4 py-2 border rounded-md hover:bg-gray-50 dark:hover:bg-gray-800 text-gray-700 dark:text-gray-300 border-gray-300 dark:border-gray-600"
              >
                取消
              </button>
              <button
                type="submit"
                className="px-4 py-2 bg-primary text-primary-foreground rounded-md hover:bg-primary/90"
              >
                保存
              </button>
            </div>
          )}

          {/* 微信 iLink 只有取消按钮 */}
          {formData.type === 'wechat_ilink' && !isLoggedIn && (
            <div className="p-4 border-t dark:border-gray-700 flex justify-end">
              <button
                type="button"
                onClick={onClose}
                className="px-4 py-2 border rounded-md hover:bg-gray-50 dark:hover:bg-gray-800 text-gray-700 dark:text-gray-300 border-gray-300 dark:border-gray-600"
              >
                取消
              </button>
            </div>
          )}

          {/* 微信 iLink 已登录显示关闭按钮 */}
          {formData.type === 'wechat_ilink' && isLoggedIn && (
            <div className="p-4 border-t dark:border-gray-700 flex justify-end">
              <button
                type="button"
                onClick={onClose}
                className="px-4 py-2 bg-primary text-primary-foreground rounded-md hover:bg-primary/90"
              >
                关闭
              </button>
            </div>
          )}
        </form>
      </div>
    </div>
  )
}

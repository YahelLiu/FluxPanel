# 天气推送简化计划

## Context

天气推送功能当前需要复杂的渠道配置（`weather_config.channel_id` 和 `client_orders.channel_id`），用户反馈不工作。需要简化天气推送，让它直接发送到已登录的 iLink 用户，不再需要配置渠道。

## Implementation Plan

### 修改 `backend/notify/weather.go`

简化 `SendWeatherNotifications` 和 `SendWeatherToClient` 方法：
1. 直接使用 iLink 发送，不再通过渠道系统
2. 检查是否有已登录的 iLink，如果有就发送给该用户
3. 移除对 `channel_id` 的依赖

```go
// SendWeatherNotifications 发送天气通知（简化版）
func (w *WeatherService) SendWeatherNotifications(config models.WeatherConfig) {
    // 检查 iLink 是否已登录
    if !wecom.HasWechatILinkChannel() {
        log.Println("[weather] iLink not logged in, skipping weather notification")
        return
    }

    client := wecom.GetClient()
    if client == nil {
        log.Println("[weather] iLink client not available")
        return
    }

    // 获取 iLink 用户 ID
    ilinkUserID := wecom.GetILinkUserID()
    if ilinkUserID == "" {
        log.Println("[weather] iLink user ID not found")
        return
    }

    // 获取所有启用天气推送且有位置信息的客户端
    type ClientLocation struct {
        ClientID string
        Location string
    }
    var clientLocations []ClientLocation

    database.DB.Table("events").
        Select("DISTINCT events.client_id, events.data->>'location' as location").
        Where("events.data->>'location' IS NOT NULL AND events.data->>'location' != ''").
        Where("client_orders.weather_enabled = ?", true).
        Joins("LEFT JOIN client_orders ON events.client_id = client_orders.client_id").
        Scan(&clientLocations)

    if len(clientLocations) == 0 {
        log.Println("[weather] No clients with weather enabled")
        return
    }

    for _, cl := range clientLocations {
        weather, err := w.getWeather(config.ApiKey, config.ApiHost, cl.Location)
        if err != nil {
            log.Printf("[weather] Failed to get weather for %s: %v", cl.Location, err)
            continue
        }

        content := w.buildWeatherMessage(cl.Location, weather)
        
        if err := messaging.SendTextReply(context.Background(), client, ilinkUserID, content, "", ""); err != nil {
            log.Printf("[weather] Failed to send to %s: %v", ilinkUserID, err)
        } else {
            log.Printf("[weather] Sent to %s for location: %s", ilinkUserID, cl.Location)
        }

        w.saveWeatherRecord(cl.ClientID, cl.Location, weather)
    }
}
```

### 修改 `frontend/src/components/WeatherSettings.tsx`

移除渠道选择，简化为只需要配置 API Key：
- 移除 `channel_id` 配置
- 只保留 API Key 和 API Host 配置

### 修改 `frontend/src/components/Dashboard.tsx`

简化客户端天气推送设置：
- 移除渠道选择下拉框
- 只保留"天气推送"开关

## Files to Modify

| 文件 | 修改内容 |
|------|----------|
| `backend/notify/weather.go` | 简化发送逻辑，直接使用 iLink |
| `frontend/src/components/WeatherSettings.tsx` | 移除渠道选择 |
| `frontend/src/components/Dashboard.tsx` | 简化天气推送设置 |

## Verification

1. 配置天气 API Key
2. 在客户端卡片开启"天气推送"
3. 点击"立即发送天气"按钮
4. 确认收到天气消息

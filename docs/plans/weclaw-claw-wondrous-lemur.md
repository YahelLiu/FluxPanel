# 将微信登录集成到通知渠道系统

## Context

### 问题背景
当前的微信登录实现是独立的，与通知渠道系统分离：
- 微信凭证存储在 `wecom_credentials` 表
- 登录 API 在 `/api/wecom/login/*`
- 不在通知渠道管理界面中

用户希望：
1. 将微信 iLink 登录集成到通知渠道系统中
2. 在前端通知渠道管理界面可以添加/管理微信渠道
3. 微信渠道只能添加一个（单例模式）
4. 登录时返回二维码图片而非 JSON

### 现有通知渠道架构

**数据模型：**
```go
// models/notification.go
type NotificationChannel struct {
    ID          uint
    Name        string
    Type        NotificationType  // "feishu" | "wechat_work"
    Mode        NotificationMode  // "webhook" | "app"
    Enabled     bool
    Trigger     TriggerCondition
    Feishu      FeishuConfig     // JSONB
    WechatWork  WechatWorkConfig // JSONB
    Description string
}

type WechatWorkConfig struct {
    WebhookURL string
    CorpID     string
    AgentID    string
    Secret     string
    UserIDs    []string
}
```

**API：**
- `GET /api/notifications/channels` - 列表
- `POST /api/notifications/channels` - 创建
- `PUT /api/notifications/channels/:id` - 更新
- `DELETE /api/notifications/channels/:id` - 删除
- `POST /api/notifications/channels/:id/test` - 测试

**通知发送：**
- `notify/factory.go` - 缓存通知器
- `notify/service.go` - 分发通知

---

## 实现方案

### 方案：添加新的渠道类型 `wechat_ilink`

在现有通知渠道框架内添加新的微信 iLink 渠道类型。

#### 优点
- 复用现有通知渠道管理界面
- 统一的通知发送流程
- 保持架构一致性

#### 关键变化
1. 新增 `NotificationTypeWechatILink` 渠道类型
2. 新增 `WechatILinkConfig` 配置结构
3. 修改前端显示登录二维码
4. 限制只能创建一个微信 iLink 渠道

---

## 详细步骤

### Phase 1: 数据模型

#### 1.1 添加新的渠道类型
**文件：** `backend/models/notification.go`

```go
const (
    NotificationTypeFeishu     NotificationType = "feishu"
    NotificationTypeWechatWork NotificationType = "wechat_work"
    NotificationTypeWechatILink NotificationType = "wechat_ilink"  // 新增
)

// WechatILinkConfig 微信 iLink 配置
type WechatILinkConfig struct {
    BotToken    string `json:"bot_token,omitempty"`
    ILinkBotID  string `json:"ilink_bot_id,omitempty"`
    BaseURL     string `json:"base_url,omitempty"`
    ILinkUserID string `json:"ilink_user_id,omitempty"`
    UserIDs     []string `json:"user_ids,omitempty"`  // 接收通知的用户列表
    LoggedIn    bool   `json:"logged_in"`  // 是否已登录
}
```

#### 1.2 更新 NotificationChannel
```go
type NotificationChannel struct {
    // ... 现有字段 ...
    WechatILink WechatILinkConfig `gorm:"type:jsonb" json:"wechat_ilink"`
}
```

---

### Phase 2: 后端 API

#### 2.1 登录 API 改造
**文件：** `backend/handlers/notification.go`

```go
// GetWechatILinkQRCode 获取微信登录二维码
// GET /api/notifications/channels/wechat-ilink/qrcode
func GetWechatILinkQRCode(c *gin.Context) {
    qr, err := ilink.FetchQRCode(c.Request.Context())
    if err != nil {
        c.JSON(500, gin.H{"error": err.Error()})
        return
    }
    
    // 返回二维码图片 URL
    c.JSON(200, gin.H{
        "qrcode_url": qr.QRCodeImgContent,
        "qrcode":     qr.QRCode,
    })
}

// CheckWechatILinkStatus 检查登录状态
// GET /api/notifications/channels/wechat-ilink/status?qrcode=xxx
func CheckWechatILinkStatus(c *gin.Context) {
    qrcode := c.Query("qrcode")
    
    creds, err := ilink.PollQRStatus(c.Request.Context(), qrcode, nil)
    if err != nil {
        c.JSON(200, gin.H{"status": "waiting"})
        return
    }
    
    // 保存到通知渠道
    channel := getOrCreateWechatILinkChannel(creds)
    
    c.JSON(200, gin.H{
        "status": "success",
        "channel": channel,
    })
}
```

#### 2.2 限制只能创建一个微信 iLink 渠道
```go
// CreateChannel 中添加检查
func CreateChannel(c *gin.Context) {
    var req CreateChannelRequest
    // ...
    
    // 微信 iLink 只能创建一个
    if req.Type == models.NotificationTypeWechatILink {
        var count int64
        database.DB.Model(&models.NotificationChannel{}).
            Where("type = ?", models.NotificationTypeWechatILink).
            Count(&count)
        if count > 0 {
            c.JSON(400, gin.H{"error": "微信渠道已存在，请编辑现有渠道"})
            return
        }
    }
    
    // ...
}
```

---

### Phase 3: 通知发送

#### 3.1 更新 notify/service.go
```go
func (d *Dispatcher) dispatchWechatILink(config models.WechatILinkConfig, title, content string, event models.Event) error {
    if !config.LoggedIn || config.BotToken == "" {
        return fmt.Errorf("微信未登录")
    }
    
    client := wecom.GetClient()
    if client == nil {
        return wecom.ErrNotLoggedIn
    }
    
    // 发送给配置的用户列表
    for _, userID := range config.UserIDs {
        if err := messaging.SendTextReply(context.Background(), client, userID, content, "", ""); err != nil {
            log.Printf("发送微信消息失败: %v", err)
        }
    }
    
    return nil
}
```

#### 3.2 更新 factory.go
```go
type NotifierFactory struct {
    feishuCache      map[string]*FeishuNotifier
    wechatCache      map[string]*WechatWorkNotifier
    wechatILinkCache *WeComiLinkNotifier  // 单例
    mu               sync.RWMutex
}
```

---

### Phase 4: 前端集成

#### 4.1 添加微信 iLink 渠道表单
在通知渠道创建/编辑表单中添加微信 iLink 类型。

#### 4.2 显示登录二维码
当选择微信 iLink 类型时：
1. 显示"扫码登录"按钮
2. 点击后弹窗显示二维码图片
3. 轮询登录状态
4. 登录成功后自动填充配置

---

### Phase 5: 清理旧代码

删除独立的微信登录相关代码：
- `handlers/wecom_auth.go` 中的登录 API（迁移到 notification.go）
- `models/wecom_credentials.go`（使用 NotificationChannel 替代）

---

## 文件变更清单

### 修改文件
| 文件 | 变更 |
|------|------|
| `backend/models/notification.go` | 添加 WechatILinkConfig 和 NotificationTypeWechatILink |
| `backend/handlers/notification.go` | 添加微信登录 API、限制单例 |
| `backend/notify/service.go` | 添加 dispatchWechatILink |
| `backend/notify/factory.go` | 添加微信 iLink 通知器缓存 |
| `backend/main.go` | 添加新路由 |

### 删除文件
| 文件 | 原因 |
|------|------|
| `backend/models/wecom_credentials.go` | 使用 NotificationChannel 替代 |
| `backend/handlers/wecom_auth.go` | 迁移到 notification.go |

### 前端修改
| 文件 | 变更 |
|------|------|
| 通知渠道管理页面 | 添加微信 iLink 类型、显示二维码 |

---

## 验证方案

1. **创建微信渠道**
   - 在通知渠道管理界面选择"微信 iLink"
   - 点击扫码登录
   - 显示二维码图片
   - 扫码成功后自动保存

2. **发送通知**
   - 创建告警规则关联微信渠道
   - 触发告警后验证收到微信消息

3. **单例限制**
   - 尝试创建第二个微信渠道
   - 应该提示"微信渠道已存在"

---

## API 变更

### 新增 API
```
GET  /api/notifications/channels/wechat-ilink/qrcode   # 获取登录二维码
GET  /api/notifications/channels/wechat-ilink/status   # 检查登录状态
POST /api/notifications/channels/wechat-ilink/logout   # 登出
```

### 修改 API
```
POST /api/notifications/channels  # 添加 wechat_ilink 类型，限制单例
PUT  /api/notifications/channels/:id  # 支持 wechat_ilink 配置
```

### 删除 API
```
GET  /api/wecom/login/qrcode   # 迁移到通知渠道 API
GET  /api/wecom/login/status   # 迁移到通知渠道 API
GET  /api/wecom/status         # 迁移到通知渠道 API
DELETE /api/wecom/session      # 迁移到通知渠道 API
```

# FluxPanel 通知系统适配器架构设计

## 概述

设计一个统一的通知适配器系统，让各种信息源（告警、天气、提醒、AI消息等）不需要关心具体的通知渠道实现。信息源只需要将数据传递给适配器，由适配器负责转换和发送到不同的通知渠道。

## 核心架构

```
┌─────────────────────────────────────────────────────────────────┐
│                        信息源 (Sources)                          │
│  ┌──────────┐ ┌──────────┐ ┌──────────┐ ┌──────────┐           │
│  │  Alert   │ │ Weather  │ │ Reminder │ │   AI     │   ...     │
│  │ Service  │ │ Service  │ │ Service  │ │  Agent   │           │
│  └────┬─────┘ └────┬─────┘ └────┬─────┘ └────┬─────┘           │
└───────┼────────────┼────────────┼────────────┼──────────────────┘
        │            │            │            │
        ▼            ▼            ▼            ▼
┌─────────────────────────────────────────────────────────────────┐
│                     通知适配器 (Notifier)                        │
│  ┌─────────────────────────────────────────────────────────────┐│
│  │                    NotifyMessage                            ││
│  │  - Type: alert | weather | reminder | chat | system        ││
│  │  - Title: string                                            ││
│  │  - Content: string                                          ││
│  │  - Priority: low | normal | high | urgent                   ││
│  │  - Metadata: map[string]any (可选的额外数据)                ││
│  └─────────────────────────────────────────────────────────────┘│
│                                                                  │
│  ┌─────────────────────────────────────────────────────────────┐│
│  │                    Channel Router                           ││
│  │  根据消息类型/优先级/用户配置，路由到合适的渠道              ││
│  └─────────────────────────────────────────────────────────────┘│
└─────────────────────────────────────────────────────────────────┘
        │            │            │
        ▼            ▼            ▼
┌─────────────────────────────────────────────────────────────────┐
│                     渠道驱动 (Drivers)                           │
│  ┌──────────┐ ┌──────────┐ ┌──────────┐ ┌──────────┐           │
│  │  iLink   │ │  Feishu  │ │  Webhook │ │   ...    │           │
│  │  Driver  │ │  Driver  │ │  Driver  │ │          │           │
│  └──────────┘ └──────────┘ └──────────┘ └──────────┘           │
└─────────────────────────────────────────────────────────────────┘
```

## 文件结构

```
backend/
├── notify/
│   ├── notifier.go          # 核心接口定义
│   ├── message.go           # 消息结构定义
│   ├── router.go            # 渠道路由器
│   ├── drivers/
│   │   ├── driver.go        # Driver 接口
│   │   ├── ilink.go         # iLink 驱动实现
│   │   ├── feishu.go        # 飞书驱动实现
│   │   └── webhook.go       # 通用 Webhook 驱动
│   └── service.go           # 通知服务（对外暴露的统一入口）
```

## 接口定义

### 1. NotifyMessage - 通知消息

```go
// notify/message.go
package notify

import "time"

// MessageType 消息类型
type MessageType string

const (
    MessageTypeAlert    MessageType = "alert"     // 告警消息
    MessageTypeWeather  MessageType = "weather"   // 天气推送
    MessageTypeReminder MessageType = "reminder"  // 提醒消息
    MessageTypeChat     MessageType = "chat"      // AI 聊天
    MessageTypeSystem   MessageType = "system"    // 系统消息
)

// Priority 消息优先级
type Priority string

const (
    PriorityLow    Priority = "low"
    PriorityNormal Priority = "normal"
    PriorityHigh   Priority = "high"
    PriorityUrgent Priority = "urgent"
)

// NotifyMessage 统一的通知消息结构
type NotifyMessage struct {
    Type     MessageType         // 消息类型
    Title    string              // 标题
    Content  string              // 正文内容
    Priority Priority            // 优先级
    Metadata map[string]any      // 额外元数据
    
    // 来源信息
    SourceID   string            // 来源标识（客户端ID、用户ID等）
    SourceName string            // 来源名称
    
    // 时间信息
    Timestamp time.Time          // 消息时间
}

// NewNotifyMessage 创建通知消息的便捷方法
func NewNotifyMessage(msgType MessageType, title, content string) *NotifyMessage {
    return &NotifyMessage{
        Type:      msgType,
        Title:     title,
        Content:   content,
        Priority:  PriorityNormal,
        Metadata:  make(map[string]any),
        Timestamp: time.Now(),
    }
}

// WithPriority 设置优先级
func (m *NotifyMessage) WithPriority(p Priority) *NotifyMessage {
    m.Priority = p
    return m
}

// WithMetadata 添加元数据
func (m *NotifyMessage) WithMetadata(key string, value any) *NotifyMessage {
    if m.Metadata == nil {
        m.Metadata = make(map[string]any)
    }
    m.Metadata[key] = value
    return m
}

// WithSource 设置来源
func (m *NotifyMessage) WithSource(id, name string) *NotifyMessage {
    m.SourceID = id
    m.SourceName = name
    return m
}
```

### 2. Driver - 渠道驱动接口

```go
// notify/drivers/driver.go
package drivers

import "client-monitor/notify"

// Driver 通知渠道驱动接口
type Driver interface {
    // Name 返回驱动名称
    Name() string
    
    // Send 发送消息
    Send(msg *notify.NotifyMessage) error
    
    // IsAvailable 检查驱动是否可用（如是否已登录、配置是否正确）
    IsAvailable() bool
    
    // SupportedTypes 返回支持的消息类型（空表示支持所有类型）
    SupportedTypes() []notify.MessageType
}

// DriverInfo 驱动信息
type DriverInfo struct {
    Name        string
    Description string
    Available   bool
    Types       []notify.MessageType
}
```

### 3. iLink 驱动实现

```go
// notify/drivers/ilink.go
package drivers

import (
    "context"
    "log"
    
    "client-monitor/messaging"
    "client-monitor/notify"
    "client-monitor/wecom"
)

// ILinkDriver iLink 通知驱动
type ILinkDriver struct{}

// NewILinkDriver 创建 iLink 驱动
func NewILinkDriver() *ILinkDriver {
    return &ILinkDriver{}
}

// Name 返回驱动名称
func (d *ILinkDriver) Name() string {
    return "ilink"
}

// Send 发送消息
func (d *ILinkDriver) Send(msg *notify.NotifyMessage) error {
    if !d.IsAvailable() {
        return ErrNotAvailable
    }
    
    client := wecom.GetClient()
    if client == nil {
        return ErrClientNotReady
    }
    
    userID := wecom.GetILinkUserID()
    if userID == "" {
        return ErrUserNotFound
    }
    
    // 格式化消息
    content := d.formatMessage(msg)
    
    // 发送消息
    if err := messaging.SendTextReply(context.Background(), client, userID, content, "", ""); err != nil {
        log.Printf("[ilink] Send failed: %v", err)
        return err
    }
    
    log.Printf("[ilink] Sent %s message to %s", msg.Type, userID)
    return nil
}

// IsAvailable 检查驱动是否可用
func (d *ILinkDriver) IsAvailable() bool {
    return wecom.HasWechatILinkChannel()
}

// SupportedTypes 返回支持的消息类型（空表示支持所有）
func (d *ILinkDriver) SupportedTypes() []notify.MessageType {
    return nil // 支持所有类型
}

// formatMessage 格式化消息内容
func (d *ILinkDriver) formatMessage(msg *notify.NotifyMessage) string {
    var sb strings.Builder
    
    // 根据消息类型添加图标
    icon := d.getIcon(msg.Type)
    if icon != "" {
        sb.WriteString(icon)
        sb.WriteString(" ")
    }
    
    // 添加标题
    if msg.Title != "" {
        sb.WriteString(msg.Title)
        sb.WriteString("\n\n")
    }
    
    // 添加内容
    sb.WriteString(msg.Content)
    
    return sb.String()
}

// getIcon 获取消息类型对应的图标
func (d *ILinkDriver) getIcon(msgType notify.MessageType) string {
    switch msgType {
    case notify.MessageTypeAlert:
        return "🚨"
    case notify.MessageTypeWeather:
        return "🌤️"
    case notify.MessageTypeReminder:
        return "⏰"
    case notify.MessageTypeSystem:
        return "📢"
    default:
        return ""
    }
}
```

### 4. Router - 渠道路由器

```go
// notify/router.go
package notify

import (
    "log"
    "sync"
    
    "client-monitor/notify/drivers"
)

// Router 消息路由器
type Router struct {
    drivers map[string]drivers.Driver
    mu      sync.RWMutex
}

var (
    router     *Router
    routerOnce sync.Once
)

// GetRouter 获取路由器单例
func GetRouter() *Router {
    routerOnce.Do(func() {
        router = &Router{
            drivers: make(map[string]drivers.Driver),
        }
        // 注册默认驱动
        router.RegisterDriver(drivers.NewILinkDriver())
        router.RegisterDriver(drivers.NewFeishuDriver())
    })
    return router
}

// RegisterDriver 注册驱动
func (r *Router) RegisterDriver(driver drivers.Driver) {
    r.mu.Lock()
    defer r.mu.Unlock()
    r.drivers[driver.Name()] = driver
    log.Printf("[router] Registered driver: %s (available: %v)", driver.Name(), driver.IsAvailable())
}

// Route 路由消息到合适的渠道
func (r *Router) Route(msg *NotifyMessage) error {
    r.mu.RLock()
    defer r.mu.RUnlock()
    
    // 查找可用的驱动
    for name, driver := range r.drivers {
        if !driver.IsAvailable() {
            continue
        }
        
        // 检查消息类型是否支持
        supportedTypes := driver.SupportedTypes()
        if len(supportedTypes) > 0 {
            supported := false
            for _, t := range supportedTypes {
                if t == msg.Type {
                    supported = true
                    break
                }
            }
            if !supported {
                continue
            }
        }
        
        // 发送消息
        if err := driver.Send(msg); err != nil {
            log.Printf("[router] Driver %s failed: %v", name, err)
            continue
        }
        
        return nil
    }
    
    return ErrNoAvailableDriver
}

// RouteTo 指定渠道发送
func (r *Router) RouteTo(driverName string, msg *NotifyMessage) error {
    r.mu.RLock()
    driver, ok := r.drivers[driverName]
    r.mu.RUnlock()
    
    if !ok {
        return ErrDriverNotFound
    }
    
    return driver.Send(msg)
}

// GetAvailableDrivers 获取所有可用的驱动
func (r *Router) GetAvailableDrivers() []drivers.DriverInfo {
    r.mu.RLock()
    defer r.mu.RUnlock()
    
    var result []drivers.DriverInfo
    for _, driver := range r.drivers {
        result = append(result, drivers.DriverInfo{
            Name:        driver.Name(),
            Available:   driver.IsAvailable(),
            Types:       driver.SupportedTypes(),
        })
    }
    return result
}
```

### 5. Service - 统一通知服务

```go
// notify/service.go
package notify

import "log"

// Service 通知服务（对外暴露的统一入口）
type Service struct {
    router *Router
}

var service *Service

// GetNotifyService 获取通知服务
func GetNotifyService() *Service {
    if service == nil {
        service = &Service{
            router: GetRouter(),
        }
    }
    return service
}

// Send 发送通知（自动路由到可用渠道）
func (s *Service) Send(msg *NotifyMessage) error {
    log.Printf("[notify] Sending %s message: %s", msg.Type, msg.Title)
    return s.router.Route(msg)
}

// SendTo 指定渠道发送
func (s *Service) SendTo(driverName string, msg *NotifyMessage) error {
    return s.router.RouteTo(driverName, msg)
}

// 便捷方法

// SendAlert 发送告警
func (s *Service) SendAlert(title, content string, priority Priority) error {
    msg := NewNotifyMessage(MessageTypeAlert, title, content).
        WithPriority(priority)
    return s.Send(msg)
}

// SendWeather 发送天气
func (s *Service) SendWeather(location, content string) error {
    msg := NewNotifyMessage(MessageTypeWeather, "天气预报", content).
        WithSource(location, location)
    return s.Send(msg)
}

// SendReminder 发送提醒
func (s *Service) SendReminder(content string) error {
    msg := NewNotifyMessage(MessageTypeReminder, "提醒", content).
        WithPriority(PriorityHigh)
    return s.Send(msg)
}

// SendSystem 发送系统消息
func (s *Service) SendSystem(title, content string) error {
    msg := NewNotifyMessage(MessageTypeSystem, title, content)
    return s.Send(msg)
}
```

## 信息源使用示例

### 告警服务使用

```go
// notify/alert.go (修改后)
func (a *AlertService) sendAlertNotification(threshold models.AlertThreshold, event models.Event, value float64) {
    title := threshold.Name
    content := fmt.Sprintf("%s: %.1f%% (阈值: %.1f%%)", 
        a.getMetricName(threshold.MetricType), value, threshold.Threshold)
    
    // 使用统一的通知服务
    err := GetNotifyService().SendAlert(title, content, PriorityHigh)
    if err != nil {
        log.Printf("[alert] Failed to send: %v", err)
    }
}
```

### 天气服务使用

```go
// notify/weather.go (修改后)
func (w *WeatherService) sendWeatherNotification(location string, weather *models.WeatherAPIResponse) {
    content := w.buildWeatherContent(weather)
    
    // 使用统一的通知服务
    err := GetNotifyService().SendWeather(location, content)
    if err != nil {
        log.Printf("[weather] Failed to send: %v", err)
    }
}
```

### 提醒服务使用

```go
// notify/reminder.go (修改后)
func (r *ReminderService) sendReminder(reminder *models.Reminder) error {
    content := reminder.Content
    
    // 使用统一的通知服务
    return GetNotifyService().SendReminder(content)
}
```

## 扩展性

### 添加新的通知渠道

只需要实现 `Driver` 接口并注册：

```go
// notify/drivers/email.go
package drivers

type EmailDriver struct {
    config EmailConfig
}

func NewEmailDriver(config EmailConfig) *EmailDriver {
    return &EmailDriver{config: config}
}

func (d *EmailDriver) Name() string { return "email" }
func (d *EmailDriver) Send(msg *notify.NotifyMessage) error {
    // 实现邮件发送逻辑
}
func (d *EmailDriver) IsAvailable() bool {
    return d.config.SMTPHost != ""
}
func (d *EmailDriver) SupportedTypes() []notify.MessageType {
    return nil // 支持所有类型
}

// 在 main.go 中注册
notify.GetRouter().RegisterDriver(drivers.NewEmailDriver(emailConfig))
```

## 实现步骤

1. **创建接口和消息结构** (`message.go`, `drivers/driver.go`)
2. **实现 iLink 驱动** (`drivers/ilink.go`) - 将现有 iLink 发送逻辑迁移
3. **实现飞书驱动** (`drivers/feishu.go`) - 将现有飞书发送逻辑迁移
4. **创建路由器** (`router.go`)
5. **创建统一服务** (`service.go`)
6. **修改现有信息源** - 改为使用统一的通知服务
7. **清理旧代码** - 移除 `factory.go`, `dispatcher` 等旧实现

## 优势

1. **解耦** - 信息源和通知渠道完全解耦
2. **可扩展** - 添加新渠道只需实现 Driver 接口
3. **可测试** - 可以轻松 mock Driver 进行测试
4. **统一管理** - 所有通知逻辑集中在 Router
5. **灵活路由** - 可根据消息类型、优先级等选择合适渠道

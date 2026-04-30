package types

import "time"

// MessageType 消息类型
type MessageType string

const (
	MessageTypeAlert    MessageType = "alert"    // 告警消息
	MessageTypeWeather  MessageType = "weather"  // 天气推送
	MessageTypeReminder MessageType = "reminder" // 提醒消息
	MessageTypeChat     MessageType = "chat"     // AI 聊天
	MessageTypeSystem   MessageType = "system"   // 系统消息
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
	Type     MessageType    // 消息类型
	Title    string         // 标题
	Content  string         // 正文内容
	Priority Priority       // 优先级
	Metadata map[string]any // 额外元数据

	// 来源信息
	SourceID   string // 来源标识（客户端ID、用户ID等）
	SourceName string // 来源名称

	// 时间信息
	Timestamp time.Time // 消息时间
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

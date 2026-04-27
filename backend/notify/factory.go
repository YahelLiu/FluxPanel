package notify

import (
	"log"
	"sync"

	"client-monitor/models"
)

// NotifierFactory 通知器工厂
type NotifierFactory struct {
	feishuCache map[string]*FeishuNotifier
	wechatCache map[string]*WechatWorkNotifier
	mu          sync.RWMutex
}

var factory *NotifierFactory
var factoryOnce sync.Once

// GetFactory 获取工厂单例
func GetFactory() *NotifierFactory {
	factoryOnce.Do(func() {
		factory = &NotifierFactory{
			feishuCache: make(map[string]*FeishuNotifier),
			wechatCache: make(map[string]*WechatWorkNotifier),
		}
	})
	return factory
}

// GetFeishu 获取飞书通知器
func (f *NotifierFactory) GetFeishu(config models.FeishuConfig) *FeishuNotifier {
	key := config.WebhookURL + config.AppID

	f.mu.RLock()
	notifier, ok := f.feishuCache[key]
	f.mu.RUnlock()

	if ok {
		return notifier
	}

	notifier = NewFeishuNotifier(config)
	f.mu.Lock()
	f.feishuCache[key] = notifier
	f.mu.Unlock()

	return notifier
}

// GetWechatWork 获取企业微信通知器
func (f *NotifierFactory) GetWechatWork(config models.WechatWorkConfig) *WechatWorkNotifier {
	key := config.WebhookURL + config.CorpID

	f.mu.RLock()
	notifier, ok := f.wechatCache[key]
	f.mu.RUnlock()

	if ok {
		return notifier
	}

	notifier = NewWechatWorkNotifier(config)
	f.mu.Lock()
	f.wechatCache[key] = notifier
	f.mu.Unlock()

	return notifier
}

// ClearCache 清除缓存
func (f *NotifierFactory) ClearCache() {
	f.mu.Lock()
	f.feishuCache = make(map[string]*FeishuNotifier)
	f.wechatCache = make(map[string]*WechatWorkNotifier)
	f.mu.Unlock()
	log.Println("Notifier cache cleared")
}

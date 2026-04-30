package wecom

import (
	"context"
	"log"
	"sync"
	"time"

	"client-monitor/ilink"
)

// Monitor 微信消息监听器
type Monitor struct {
	client  *ilink.Client
	handler *MessageHandler
	mu      sync.Mutex
	running bool
}

// NewMonitor 创建消息监听器
func NewMonitor() (*Monitor, error) {
	client := GetClient()
	if client == nil {
		return nil, ErrNotLoggedIn
	}

	return &Monitor{
		client:  client,
		handler: NewMessageHandler(),
	}, nil
}

// Start 启动消息监听（阻塞）
func (m *Monitor) Start(ctx context.Context) error {
	m.mu.Lock()
	if m.running {
		m.mu.Unlock()
		return nil
	}
	m.running = true
	m.mu.Unlock()
	SetMonitorRunning(true)

	defer func() {
		m.mu.Lock()
		m.running = false
		m.mu.Unlock()
		SetMonitorRunning(false)
	}()

	monitor, err := ilink.NewMonitor(m.client, m.handler.HandleMessage)
	if err != nil {
		return err
	}

	log.Println("[wecom] Starting iLink monitor...")
	return monitor.Run(ctx)
}

// RunWithRestart 带自动重连的运行
func (m *Monitor) RunWithRestart(ctx context.Context) {
	const maxRestartDelay = 30 * time.Second
	restartDelay := 3 * time.Second

	for {
		if err := m.Start(ctx); err != nil {
			// 检查 context 是否取消
			if ctx.Err() != nil {
				log.Println("[wecom] Monitor stopped (context cancelled)")
				return
			}

			log.Printf("[wecom] Monitor error: %v, reconnecting in %s...", err, restartDelay)

			select {
			case <-time.After(restartDelay):
			case <-ctx.Done():
				return
			}

			// 指数退避
			restartDelay *= 2
			if restartDelay > maxRestartDelay {
				restartDelay = maxRestartDelay
			}
			continue
		}

		// 正常退出
		return
	}
}

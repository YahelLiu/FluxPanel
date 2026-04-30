package wecom

import (
	"sync"
	"time"
)

var (
	monitorRunning   bool
	monitorStartTime time.Time
	monitorMu        sync.RWMutex
)

// SetMonitorRunning 设置 Monitor 运行状态
func SetMonitorRunning(running bool) {
	monitorMu.Lock()
	defer monitorMu.Unlock()
	monitorRunning = running
	if running {
		monitorStartTime = time.Now()
	}
}

// GetMonitorStatus 获取 Monitor 运行状态
func GetMonitorStatus() (running bool, startTime time.Time) {
	monitorMu.RLock()
	defer monitorMu.RUnlock()
	return monitorRunning, monitorStartTime
}

package notify

import (
	"errors"
	"fmt"
	"log"
	"sync"

	"client-monitor/notify/drivers"
	"client-monitor/notify/types"
)

// Errors
var (
	ErrNoAvailableDriver = errors.New("no available driver")
	ErrDriverNotFound    = errors.New("driver not found")
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
		log.Printf("[router] Creating new router instance")
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
func (r *Router) Route(msg *types.NotifyMessage) error {
	r.mu.RLock()
	defer r.mu.RUnlock()

	log.Printf("[router] Routing message type=%s, title=%s, drivers count=%d", msg.Type, msg.Title, len(r.drivers))

	// 按优先级顺序检查驱动：ilink > feishu
	driverOrder := []string{"ilink", "feishu"}

	for _, name := range driverOrder {
		driver, ok := r.drivers[name]
		if !ok {
			continue
		}

		log.Printf("[router] Checking driver: %s", name)
		available := driver.IsAvailable()
		log.Printf("[router] Driver %s: available=%v", name, available)

		if !available {
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

		log.Printf("[router] Driver %s sent successfully", name)
		return nil
	}

	return ErrNoAvailableDriver
}

// RouteAll 路由消息到所有可用渠道
func (r *Router) RouteAll(msg *types.NotifyMessage) []error {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var errs []error

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
			errs = append(errs, fmt.Errorf("%s: %w", name, err))
		}
	}

	if len(errs) == 0 {
		return nil
	}
	return errs
}

// RouteTo 指定渠道发送
func (r *Router) RouteTo(driverName string, msg *types.NotifyMessage) error {
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
			Name:      driver.Name(),
			Available: driver.IsAvailable(),
			Types:     driver.SupportedTypes(),
		})
	}
	return result
}

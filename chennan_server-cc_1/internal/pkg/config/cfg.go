package config

import "sync"

// cfg 持有全局配置，读写均通过 RWMutex 保护。
type cfg struct {
	mu      sync.RWMutex
	options Options
}

// Get 返回当前配置的值拷贝，线程安全。
func (c *cfg) Get() Options {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.options
}

func (c *cfg) set(o *Options) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.options = *o
}

// Cfg 是包级全局配置实例，通过 Setup() 初始化后使用。
var Cfg = &cfg{}

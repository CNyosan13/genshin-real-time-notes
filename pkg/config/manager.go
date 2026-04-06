package config

import (
	"sync"
)

// Manager handles thread-safe access to the application configuration.
type Manager struct {
	mu  sync.RWMutex
	cfg *Config
}

// NewManager creates a new configuration manager.
func NewManager(cfg *Config) *Manager {
	return &Manager{
		cfg: cfg,
	}
}

// Get returns the current configuration.
func (m *Manager) Get() *Config {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.cfg
}

// Set updates the current configuration.
func (m *Manager) Set(cfg *Config) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.cfg = cfg
}

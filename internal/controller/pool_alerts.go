package controller

import (
	"sync"
)

// SearchRuleAlertPool
var SearchRuleAlertPool = &AlertsStore{
	store: make(map[string]*Alert),
}

// Alert
type Alert struct {
}

// AlertsStore
type AlertsStore struct {
	mu    sync.RWMutex
	store map[string]*Alert
}

func (c *AlertsStore) Set(key string, alert *Alert) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.store[key] = alert
}

func (c *AlertsStore) Get(key string) (*Alert, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	alert, exists := c.store[key]
	return alert, exists
}

func (c *AlertsStore) GetAll() map[string]*Alert {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.store
}

func (c *AlertsStore) Delete(key string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.store, key)
}

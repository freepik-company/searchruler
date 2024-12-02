package pools

import (
	"regexp"
	"sync"

	"prosimcorp.com/SearchRuler/api/v1alpha1"
)

// Alert
type Alert struct {
	SearchRule v1alpha1.SearchRule
	Value      float64
}

// AlertsStore
type AlertsStore struct {
	mu    sync.RWMutex
	Store map[string]*Alert
}

func (c *AlertsStore) Set(key string, alert *Alert) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.Store[key] = alert
}

func (c *AlertsStore) Get(key string) (*Alert, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	alert, exists := c.Store[key]
	return alert, exists
}

func (c *AlertsStore) GetAll() map[string]*Alert {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.Store
}

func (c *AlertsStore) Delete(key string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.Store, key)
}

func (c *AlertsStore) GetByRegex(pattern string) map[string]*Alert {
	c.mu.RLock()
	defer c.mu.RUnlock()

	regex, err := regexp.Compile(pattern)
	if err != nil {
		return nil
	}

	matchedAlerts := make(map[string]*Alert)

	for key, alert := range c.Store {
		if regex.MatchString(key) {
			matchedAlerts[key] = alert
		}
	}
	return matchedAlerts
}

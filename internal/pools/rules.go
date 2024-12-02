package pools

import (
	"sync"
	"time"
)

// Rule
type Rule struct {
	FiringTime    time.Time
	ResolvingTime time.Time
	State         string
}

// RulesStore
type RulesStore struct {
	mu    sync.RWMutex
	Store map[string]*Rule
}

func (c *RulesStore) Set(key string, rule *Rule) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.Store[key] = rule
}

func (c *RulesStore) Get(key string) (*Rule, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	rule, exists := c.Store[key]
	return rule, exists
}

func (c *RulesStore) GetAll() map[string]*Rule {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.Store
}

func (c *RulesStore) Delete(key string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.Store, key)
}

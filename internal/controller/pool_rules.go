package controller

import (
	"sync"
	"time"
)

// SearchRulePool
var SearchRulePool = &RulesStore{
	store: make(map[string]*Rule),
}

// Rule
type Rule struct {
	firingTime    time.Time
	resolvingTime time.Time
	state         string
}

// RulesStore
type RulesStore struct {
	mu    sync.RWMutex
	store map[string]*Rule
}

func (c *RulesStore) Set(key string, rule *Rule) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.store[key] = rule
}

func (c *RulesStore) Get(key string) (*Rule, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	rule, exists := c.store[key]
	return rule, exists
}

func (c *RulesStore) GetAll() map[string]*Rule {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.store
}

func (c *RulesStore) Delete(key string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.store, key)
}

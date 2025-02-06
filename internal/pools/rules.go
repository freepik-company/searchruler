/*
Copyright 2024.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package pools

import (
	"sync"
	"time"

	"prosimcorp.com/SearchRuler/api/v1alpha1"
)

// Rule
type Rule struct {
	SearchRule    v1alpha1.SearchRule
	FiringTime    time.Time
	ResolvingTime time.Time
	State         string
	Value         float64
	Aggregations  interface{}
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

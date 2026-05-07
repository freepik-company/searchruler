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

	"freepik.com/searchruler/api/v1alpha1"
)

// Rule
type Rule struct {
	SearchRule    v1alpha1.SearchRule
	FiringTime    time.Time
	ResolvingTime time.Time
	State         string
	Value         float64

	// Aggregations holds the last `aggregations` block parsed from the
	// Elasticsearch response. The metrics goroutine reads it to fan out
	// spec.customMetrics into per-bucket Prometheus samples; nil when the
	// query did not return any aggregations.
	Aggregations interface{}
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

// Snapshot returns a value-copied view of every Rule in the pool, captured
// under the read lock. Mutations on the returned entries do not affect the
// pool. Use this from goroutines that race with the reconciler — notably
// the metrics ticker, which would otherwise read `rule.Aggregations` (an
// interface{} so two machine words wide) while the reconciler reassigns it
// in-place between Get and Set, producing a torn read undetectable by go's
// race detector unless both goroutines exercise the same key concurrently.
func (c *RulesStore) Snapshot() map[string]Rule {
	c.mu.RLock()
	defer c.mu.RUnlock()
	out := make(map[string]Rule, len(c.Store))
	for k, v := range c.Store {
		if v == nil {
			continue
		}
		out[k] = *v
	}
	return out
}

func (c *RulesStore) Delete(key string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.Store, key)
}

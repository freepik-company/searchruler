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

// Set stores a value-copy of rule under key. The caller's pointer is
// never aliased by the pool, so subsequent in-place mutations on it have
// no effect until another Set runs. This is the only safe way to keep
// callers from racing with goroutines that read via Get/Snapshot.
func (c *RulesStore) Set(key string, rule *Rule) {
	if rule == nil {
		return
	}
	cp := *rule
	c.mu.Lock()
	defer c.mu.Unlock()
	c.Store[key] = &cp
}

// Get returns a value-copy of the Rule stored under key. Callers mutate
// their own local copy and must call Set to publish changes back. Returning
// a copy (not the pointer) means no caller can mutate pool-internal state
// without the pool's mutex — which closes the torn-read race on the
// `Aggregations interface{}` field.
func (c *RulesStore) Get(key string) (Rule, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	v, ok := c.Store[key]
	if !ok || v == nil {
		return Rule{}, false
	}
	return *v, true
}

// GetAll returns the live pointer map. Kept for backwards compatibility
// with callers that only read; new readers should use Snapshot which
// returns a value-copied view safe to iterate concurrently with writers.
//
// Deprecated: prefer Snapshot for any reader that races with writers.
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

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

// Set stores a struct-level copy of rule under key. The copy is SHALLOW:
// scalar and array fields are copied verbatim, but the slice/map/interface
// fields inside Rule (notably Aggregations, SearchRule.Spec.CustomMetrics
// and Status.Conditions) keep the same backing storage as the caller's
// argument. This is enough to close the torn-read race on Aggregations
// (interface{} is two machine words and the field is always reassigned,
// never mutated in-place), but only because every caller follows the
// "Get → mutate the local Rule → Set" convention. Mutating slices or maps
// in-place on a value returned by Get is not safe; allocate a new
// slice/map and assign it instead.
func (c *RulesStore) Set(key string, rule *Rule) {
	if rule == nil {
		return
	}
	cp := *rule
	c.mu.Lock()
	defer c.mu.Unlock()
	c.Store[key] = &cp
}

// Get returns a struct-level (shallow) copy of the Rule stored under key.
// See Set for the convention readers and writers must follow.
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

// Snapshot returns a map of struct-level (shallow) copies of every Rule
// in the pool, captured under the read lock. The same caveats as Set/Get
// apply: slice/map/interface fields share storage with the pool entry, so
// readers must treat them as read-only. Use this from goroutines that
// race with the reconciler — notably the metrics ticker, which reads
// `rule.Aggregations` (interface{} so two machine words wide) while the
// reconciler keeps mutating its own local Rule between Get and Set.
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

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

	"prosimcorp.com/SearchRuler/api/v1alpha1"
)

// Alert
type Alert struct {
	RulerActionName string
	SearchRule      v1alpha1.SearchRule
	Value           float64
	Aggregations    interface{}
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

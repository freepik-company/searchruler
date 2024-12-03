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

import "sync"

// Credentials
type Credentials struct {
	Username string
	Password string
}

// CredentialsStore
type CredentialsStore struct {
	mu    sync.RWMutex
	Store map[string]*Credentials
}

func (c *CredentialsStore) Set(key string, creds *Credentials) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.Store[key] = creds
}

func (c *CredentialsStore) Get(key string) (*Credentials, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	creds, exists := c.Store[key]
	return creds, exists
}

func (c *CredentialsStore) GetAll() map[string]*Credentials {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.Store
}

func (c *CredentialsStore) Delete(key string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	_, exists := c.Store[key]
	if exists {
		delete(c.Store, key)
	}
}

package pools

import "sync"

// PlainCredentials
type Credentials struct {
	Username string
	Password string
}

// CredentialsStore
type CredentialsStore struct {
	mu    sync.RWMutex
	Store map[string]*Credentials
}

// Set agrega o actualiza las credenciales
func (c *CredentialsStore) Set(key string, creds *Credentials) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.Store[key] = creds
}

// Get obtiene las credenciales para una clave
func (c *CredentialsStore) Get(key string) (*Credentials, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	creds, exists := c.Store[key]
	return creds, exists
}

// Delete elimina las credenciales de la clave
func (c *CredentialsStore) Delete(key string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	_, exists := c.Store[key]
	if exists {
		delete(c.Store, key)
	}
}

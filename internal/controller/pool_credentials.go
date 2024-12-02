package controller

import "sync"

// GlobalCredentials almacena las credenciales globales
var QueryConnectorCredentialsPool = &CredentialsStore{
	store: make(map[string]*Credentials),
}

// GlobalCredentials almacena las credenciales globales
var RulerActionCredentialsPool = &CredentialsStore{
	store: make(map[string]*Credentials),
}

// PlainCredentials
type Credentials struct {
	Username string
	Password string
}

// CredentialsStore
type CredentialsStore struct {
	mu    sync.RWMutex
	store map[string]*Credentials
}

// Set agrega o actualiza las credenciales
func (c *CredentialsStore) Set(key string, creds *Credentials) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.store[key] = creds
}

// Get obtiene las credenciales para una clave
func (c *CredentialsStore) Get(key string) (*Credentials, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	creds, exists := c.store[key]
	return creds, exists
}

// Delete elimina las credenciales de la clave
func (c *CredentialsStore) Delete(key string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.store, key)
}

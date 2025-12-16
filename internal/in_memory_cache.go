package internal

import "sync"

type InMemoryCache struct {
	store map[string]*CacheEntity
	mu    sync.RWMutex
}

func NewInMemoryCache(size int) *InMemoryCache {
	c := new(InMemoryCache)
	c.store = make(map[string]*CacheEntity, size)
	return c
}

func (c *InMemoryCache) Get(key string) (*CacheEntity, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	val := c.store[key]
	return val, nil
}

func (c *InMemoryCache) Set(key string, value *CacheEntity) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.store[key] = value
	return nil
}

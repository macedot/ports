package docker

import (
	"context"
	"sync"
	"time"

	"golang.org/x/sync/singleflight"
)

// Cache wraps a Docker Collector with TTL-based caching.
type Cache struct {
	collector  *Collector
	ttl        time.Duration
	mu         sync.Mutex
	lastFetch  time.Time
	containers []ContainerInfo
	sfGroup    singleflight.Group
}

// NewCache creates a Docker cache with the given TTL.
// If collector is nil (Docker unavailable), returns a cache that always returns empty results.
func NewCache(collector *Collector, ttl time.Duration) *Cache {
	return &Cache{
		collector: collector,
		ttl:       ttl,
	}
}

// Get returns cached containers or fetches fresh data if TTL expired.
func (c *Cache) Get(ctx context.Context) ([]ContainerInfo, error) {
	// If no collector (Docker disabled), return empty
	if c.collector == nil {
		return nil, nil
	}

	c.mu.Lock()
	if time.Since(c.lastFetch) < c.ttl && c.containers != nil {
		result := c.containers
		c.mu.Unlock()
		return result, nil
	}
	c.mu.Unlock()

	// Use singleflight to coalesce concurrent requests
	result, err, _ := c.sfGroup.Do("docker-containers", func() (interface{}, error) {
		containers, err := c.collector.Collect(ctx)
		if err != nil {
			return nil, err
		}

		c.mu.Lock()
		c.containers = containers
		c.lastFetch = time.Now()
		c.mu.Unlock()

		return containers, nil
	})

	if err != nil {
		// Return stale data on error if available
		c.mu.Lock()
		if c.containers != nil {
			stale := c.containers
			c.mu.Unlock()
			return stale, nil
		}
		c.mu.Unlock()
		return nil, err
	}

	return result.([]ContainerInfo), nil
}

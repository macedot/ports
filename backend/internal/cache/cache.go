package cache

import (
	"errors"
	"os"
	"sync"
	"time"

	"golang.org/x/sync/singleflight"

	"listen-ports/internal/parser"
)

var ErrNoData = errors.New("cache: no data available")

type Cache struct {
	fetchFunc func() ([]parser.SocketEntry, error)
	ttl       time.Duration

	mu        sync.RWMutex
	data      []parser.SocketEntry
	updatedAt time.Time

	sfGroup singleflight.Group
}

func NewCache(fetchFunc func() ([]parser.SocketEntry, error)) *Cache {
	ttlStr := os.Getenv("CACHE_TTL")
	ttl := 2 * time.Second
	if ttlStr != "" {
		if parsed, err := time.ParseDuration(ttlStr); err == nil && parsed > 0 {
			ttl = parsed
		}
	}

	return &Cache{
		fetchFunc: fetchFunc,
		ttl:       ttl,
	}
}

func (c *Cache) Get() ([]parser.SocketEntry, time.Time, error) {
	c.mu.RLock()
	data := c.data
	updatedAt := c.updatedAt
	valid := !updatedAt.IsZero() && time.Since(updatedAt) < c.ttl
	c.mu.RUnlock()

	if valid {
		return data, updatedAt, nil
	}

	result, err, _ := c.sfGroup.Do("refresh", func() (interface{}, error) {
		c.mu.Lock()
		defer c.mu.Unlock()

		// Re-check under write lock (another goroutine may have refreshed)
		if !c.updatedAt.IsZero() && time.Since(c.updatedAt) < c.ttl {
			return cachedResult{data: c.data, updatedAt: c.updatedAt}, nil
		}

		newData, fetchErr := c.fetchFunc()
		if fetchErr != nil {
			// Fetch failed; return stale data if available
			if c.data != nil {
				return cachedResult{data: c.data, updatedAt: c.updatedAt, stale: true}, nil
			}
			return nil, fetchErr
		}

		c.data = newData
		c.updatedAt = time.Now()
		return cachedResult{data: c.data, updatedAt: c.updatedAt}, nil
	})

	if err != nil {
		// Fetch failed and no stale data
		return nil, time.Time{}, err
	}

	res := result.(cachedResult)
	return res.data, res.updatedAt, nil
}

type cachedResult struct {
	data      []parser.SocketEntry
	updatedAt time.Time
	stale     bool
}

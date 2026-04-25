package cache

import (
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"listen-ports/internal/parser"
)

func TestNewCache(t *testing.T) {
	// Test default TTL (2 seconds)
	c := NewCache(func() ([]parser.SocketEntry, error) {
		return nil, nil
	})
	if c.ttl != 2*time.Second {
		t.Errorf("expected default TTL of 2s, got %v", c.ttl)
	}
}

func TestNewCacheCustomTTL(t *testing.T) {
	t.Setenv("CACHE_TTL", "5s")
	c := NewCache(func() ([]parser.SocketEntry, error) {
		return nil, nil
	})
	if c.ttl != 5*time.Second {
		t.Errorf("expected TTL of 5s, got %v", c.ttl)
	}
}

func TestGetCallsFetchOnFirstCall(t *testing.T) {
	called := atomic.Bool{}
	fetch := func() ([]parser.SocketEntry, error) {
		called.Store(true)
		return []parser.SocketEntry{
			{Protocol: "tcp", LocalAddr: "127.0.0.1:8080"},
		}, nil
	}
	c := NewCache(fetch)

	data, _, err := c.Get()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !called.Load() {
		t.Error("fetch was not called on first Get")
	}
	if len(data) != 1 || data[0].Protocol != "tcp" {
		t.Errorf("unexpected data: %v", data)
	}
}

func TestGetReturnsCachedDataWithinTTL(t *testing.T) {
	callCount := atomic.Int32{}
	fetch := func() ([]parser.SocketEntry, error) {
		callCount.Add(1)
		return []parser.SocketEntry{
			{Protocol: "udp", LocalAddr: "127.0.0.1:3000"},
		}, nil
	}
	c := NewCache(fetch)

	// First call
	_, _, err := c.Get()
	if err != nil {
		t.Fatalf("first call error: %v", err)
	}
	if callCount.Load() != 1 {
		t.Errorf("expected 1 fetch call, got %d", callCount.Load())
	}

	// Second call within TTL (should not fetch again)
	_, _, err = c.Get()
	if err != nil {
		t.Fatalf("second call error: %v", err)
	}
	if callCount.Load() != 1 {
		t.Errorf("expected 1 fetch call (cached), got %d", callCount.Load())
	}
}

func TestGetTriggersRefreshAfterTTLExpires(t *testing.T) {
	callCount := atomic.Int32{}
	fetch := func() ([]parser.SocketEntry, error) {
		callCount.Add(1)
		return []parser.SocketEntry{
			{Protocol: "tcp", LocalAddr: "127.0.0.1:9000"},
		}, nil
	}
	// Use 50ms TTL for testing
	t.Setenv("CACHE_TTL", "50ms")
	c := NewCache(fetch)

	// First call
	_, _, err := c.Get()
	if err != nil {
		t.Fatalf("first call error: %v", err)
	}
	if callCount.Load() != 1 {
		t.Errorf("expected 1 fetch call, got %d", callCount.Load())
	}

	// Wait for TTL to expire
	time.Sleep(60 * time.Millisecond)

	// Second call should trigger refresh
	_, _, err = c.Get()
	if err != nil {
		t.Fatalf("refresh call error: %v", err)
	}
	if callCount.Load() != 2 {
		t.Errorf("expected 2 fetch calls after TTL expiry, got %d", callCount.Load())
	}
}

func TestStaleDataFallbackOnFetchFailure(t *testing.T) {
	callCount := atomic.Int32{}
	staleData := []parser.SocketEntry{
		{Protocol: "tcp", LocalAddr: "127.0.0.1:8080"},
	}

	fetch := func() ([]parser.SocketEntry, error) {
		callCount.Add(1)
		if callCount.Load() == 1 {
			return staleData, nil
		}
		return nil, errors.New("fetch failed")
	}
	t.Setenv("CACHE_TTL", "50ms")
	c := NewCache(fetch)

	// First call succeeds, populates cache
	_, _, err := c.Get()
	if err != nil {
		t.Fatalf("first call error: %v", err)
	}

	// Wait for TTL to expire
	time.Sleep(60 * time.Millisecond)

	// Second call fails, should return stale data
	data, _, err := c.Get()
	if err != nil {
		t.Fatalf("expected stale data fallback, got error: %v", err)
	}
	if len(data) != 1 || data[0].Protocol != "tcp" {
		t.Errorf("expected stale data, got: %v", data)
	}
}

func TestFirstCallFailureReturnsError(t *testing.T) {
	expectedErr := errors.New("fetch failed")
	fetch := func() ([]parser.SocketEntry, error) {
		return nil, expectedErr
	}
	c := NewCache(fetch)

	_, _, err := c.Get()
	if err == nil {
		t.Error("expected error on first call failure")
	}
	if !errors.Is(err, expectedErr) {
		t.Errorf("expected wrapped error, got: %v", err)
	}
}

func TestConcurrentCallsSingleflight(t *testing.T) {
	callCount := atomic.Int32{}
	startChan := make(chan struct{})
	blockChan := make(chan struct{})

	fetch := func() ([]parser.SocketEntry, error) {
		callCount.Add(1)
		// Signal that fetch has started
		close(startChan)
		// Block until unblocked (simulating slow fetch)
		<-blockChan
		return []parser.SocketEntry{
			{Protocol: "tcp", LocalAddr: "127.0.0.1:9090"},
		}, nil
	}
	// Short TTL so all concurrent calls trigger fetch
	t.Setenv("CACHE_TTL", "1ms")
	c := NewCache(fetch)

	var wg sync.WaitGroup
	goroutineCount := 5
	for i := 0; i < goroutineCount; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, _, _ = c.Get()
		}()
	}

	// Wait for fetch to start
	<-startChan

	// Give all goroutines time to call Get
	time.Sleep(10 * time.Millisecond)

	// Unblock fetch
	close(blockChan)

	wg.Wait()

	// Should only have called fetch once due to singleflight
	if callCount.Load() != 1 {
		t.Errorf("expected singleflight to call fetch once, got %d calls", callCount.Load())
	}
}

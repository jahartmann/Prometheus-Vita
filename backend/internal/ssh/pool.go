package ssh

import (
	"fmt"
	"sync"
	"time"
)

type poolEntry struct {
	client   *Client
	lastUsed time.Time
}

// Pool manages a pool of SSH connections keyed by node ID.
type Pool struct {
	mu          sync.Mutex
	connections map[string]*poolEntry
	cfg         PoolConfig
}

// NewPool creates a new SSH connection pool with the given configuration.
func NewPool(cfg PoolConfig) *Pool {
	if cfg.IdleTimeout == 0 {
		cfg.IdleTimeout = 5 * time.Minute
	}
	if cfg.MaxConnections == 0 {
		cfg.MaxConnections = 10
	}
	return &Pool{
		connections: make(map[string]*poolEntry),
		cfg:         cfg,
	}
}

// Get returns a cached connection for the given node ID, or creates a new one.
// Stale connections that have exceeded the idle timeout are closed and replaced.
func (p *Pool) Get(nodeID string, sshCfg SSHConfig) (*Client, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if entry, ok := p.connections[nodeID]; ok {
		if time.Since(entry.lastUsed) > p.cfg.IdleTimeout {
			_ = entry.client.Close()
			delete(p.connections, nodeID)
		} else {
			// Verify connection is still alive
			_, _, err := entry.client.client.SendRequest("keepalive@prometheus", true, nil)
			if err != nil {
				_ = entry.client.Close()
				delete(p.connections, nodeID)
				// Fall through to create new connection
			} else {
				entry.lastUsed = time.Now()
				return entry.client, nil
			}
		}
	}

	client, err := NewClient(sshCfg)
	if err != nil {
		return nil, fmt.Errorf("create ssh client for node %s: %w", nodeID, err)
	}

	p.connections[nodeID] = &poolEntry{
		client:   client,
		lastUsed: time.Now(),
	}

	p.evictLRU()

	return client, nil
}

// Return returns a client back to the pool, updating the last used timestamp.
// A liveness check is performed under the lock to avoid race conditions;
// dead connections are discarded atomically.
func (p *Pool) Return(nodeID string, client *Client) {
	if client == nil {
		return
	}

	p.mu.Lock()
	defer p.mu.Unlock()

	// Liveness check under lock to prevent race between check and pool update
	_, _, err := client.client.SendRequest("keepalive@prometheus", true, nil)
	if err != nil {
		_ = client.Close()
		delete(p.connections, nodeID)
		return
	}

	p.connections[nodeID] = &poolEntry{
		client:   client,
		lastUsed: time.Now(),
	}

	p.evictLRU()
}

// NewDirect creates a fresh SSH connection bypassing the pool.
// The caller is responsible for closing it. Use for long-running operations
// like file transfers where pooled connections may be stale.
func (p *Pool) NewDirect(sshCfg SSHConfig) (*Client, error) {
	return NewClient(sshCfg)
}

// evictLRU removes the least-recently-used connections when the pool exceeds
// MaxConnections. Must be called with p.mu held.
func (p *Pool) evictLRU() {
	for len(p.connections) > p.cfg.MaxConnections {
		var oldestID string
		var oldestTime time.Time
		first := true
		for id, entry := range p.connections {
			if first || entry.lastUsed.Before(oldestTime) {
				oldestID = id
				oldestTime = entry.lastUsed
				first = false
			}
		}
		if oldestID != "" {
			_ = p.connections[oldestID].client.Close()
			delete(p.connections, oldestID)
		}
	}
}

// CloseAll closes all connections in the pool.
func (p *Pool) CloseAll() {
	p.mu.Lock()
	defer p.mu.Unlock()

	for nodeID, entry := range p.connections {
		_ = entry.client.Close()
		delete(p.connections, nodeID)
	}
}

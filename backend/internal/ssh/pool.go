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
			entry.lastUsed = time.Now()
			return entry.client, nil
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

	return client, nil
}

// Return returns a client back to the pool, updating the last used timestamp.
func (p *Pool) Return(nodeID string, client *Client) {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.connections[nodeID] = &poolEntry{
		client:   client,
		lastUsed: time.Now(),
	}
}

// NewDirect creates a fresh SSH connection bypassing the pool.
// The caller is responsible for closing it. Use for long-running operations
// like file transfers where pooled connections may be stale.
func (p *Pool) NewDirect(sshCfg SSHConfig) (*Client, error) {
	return NewClient(sshCfg)
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

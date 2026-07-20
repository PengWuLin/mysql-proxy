package proxy

import (
	"context"
	"log/slog"
	"sync/atomic"

	"github.com/go-mysql-org/go-mysql/client"
)

// BackendPool manages a pool of backend MySQL connections across multiple
// backend servers, distributing connections with round-robin scheduling.
type BackendPool struct {
	pools   []*client.Pool
	counter atomic.Uint64
}

// BackendConfig is the configuration for a single backend MySQL server.
type BackendConfig struct {
	Addr     string
	User     string
	Password string
}

// PoolConfig holds connection pool sizing parameters.
type PoolConfig struct {
	MinAlive int
	MaxAlive int
	MaxIdle  int
}

// NewBackendPool creates a BackendPool with one client.Pool per backend.
func NewBackendPool(backends []BackendConfig, cfg PoolConfig) (*BackendPool, error) {
	pools := make([]*client.Pool, 0, len(backends))
	for _, b := range backends {
		pool, err := client.NewPoolWithOptions(
			b.Addr, b.User, b.Password, "",
			client.WithPoolLimits(cfg.MinAlive, cfg.MaxAlive, cfg.MaxIdle),
			client.WithLogger(slog.Default()),
		)
		if err != nil {
			// Close any pools already created before returning the error.
			for _, p := range pools {
				p.Close()
			}
			return nil, err
		}
		pools = append(pools, pool)
	}
	return &BackendPool{pools: pools}, nil
}

// GetConn picks a backend using round-robin and returns a connection from its pool.
// The caller must call the returned release function when done with the connection.
func (bp *BackendPool) GetConn(ctx context.Context) (*client.Conn, func(), error) {
	idx := bp.counter.Add(1) % uint64(len(bp.pools))
	conn, err := bp.pools[idx].GetConn(ctx)
	if err != nil {
		return nil, nil, err
	}
	release := func() {
		bp.pools[idx].PutConn(conn)
	}
	return conn, release, nil
}

// Close shuts down all backend pools.
func (bp *BackendPool) Close() {
	for _, p := range bp.pools {
		p.Close()
	}
}

// Len returns the number of backend pools.
func (bp *BackendPool) Len() int {
	return len(bp.pools)
}

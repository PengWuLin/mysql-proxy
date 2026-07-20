package proxy

import (
	"context"
	"testing"
)

func testBackendConfig() BackendConfig {
	return BackendConfig{
		Addr:     "127.0.0.1:3306",
		User:     "root",
		Password: "",
	}
}

func testPoolConfig() PoolConfig {
	return PoolConfig{MinAlive: 1, MaxAlive: 2, MaxIdle: 1}
}

func TestBackendPoolSingle(t *testing.T) {
	mysqlAvailable(t)

	bp, err := NewBackendPool([]BackendConfig{testBackendConfig()}, testPoolConfig())
	if err != nil {
		t.Fatalf("NewBackendPool: %v", err)
	}
	defer bp.Close()

	if bp.Len() != 1 {
		t.Fatalf("pool len = %d, want 1", bp.Len())
	}

	conn1, release1, err := bp.GetConn(context.Background())
	if err != nil {
		t.Fatalf("GetConn: %v", err)
	}
	release1()

	conn2, release2, err := bp.GetConn(context.Background())
	if err != nil {
		t.Fatalf("GetConn: %v", err)
	}
	release2()

	if conn1 == conn2 {
		t.Error("expected different connections from pool")
	}
}

func TestBackendPoolRoundRobin(t *testing.T) {
	mysqlAvailable(t)

	backends := []BackendConfig{
		{Addr: "127.0.0.1:3306", User: "root", Password: ""},
		{Addr: "127.0.0.1:3306", User: "root", Password: ""},
	}
	bp, err := NewBackendPool(backends, testPoolConfig())
	if err != nil {
		t.Fatalf("NewBackendPool: %v", err)
	}
	defer bp.Close()

	if bp.Len() != 2 {
		t.Fatalf("pool len = %d, want 2", bp.Len())
	}

	for i := range 3 {
		_, release, err := bp.GetConn(context.Background())
		if err != nil {
			t.Fatalf("GetConn %d: %v", i, err)
		}
		release()
	}
}

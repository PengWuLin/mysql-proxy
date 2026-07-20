package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadConfig(t *testing.T) {
	yaml := `
listen:
  addr: "127.0.0.1:4000"
proxy:
  user: "proxy_admin"
  password: "secret"
backends:
  - addr: "127.0.0.1:3306"
    user: "root"
    password: "pass1"
  - addr: "127.0.0.1:3307"
    user: "admin"
    password: "pass2"
pool:
  min_alive: 3
  max_alive: 20
  max_idle: 8
`
	cfg, err := loadFromString(t, yaml)
	if err != nil {
		t.Fatal(err)
	}

	if cfg.Listen.Addr != "127.0.0.1:4000" {
		t.Errorf("listen addr = %q, want 127.0.0.1:4000", cfg.Listen.Addr)
	}
	if len(cfg.Backends) != 2 {
		t.Fatalf("got %d backends, want 2", len(cfg.Backends))
	}
	if cfg.Backends[0].Addr != "127.0.0.1:3306" {
		t.Errorf("backend[0].addr = %q", cfg.Backends[0].Addr)
	}
	if cfg.Backends[0].Password != "pass1" {
		t.Errorf("backend[0].password = %q", cfg.Backends[0].Password)
	}
	if cfg.Proxy.User != "proxy_admin" || cfg.Proxy.Password != "secret" {
		t.Errorf("proxy auth = %+v, want proxy_admin/secret", cfg.Proxy)
	}
	if cfg.Pool.MinAlive != 3 || cfg.Pool.MaxAlive != 20 || cfg.Pool.MaxIdle != 8 {
		t.Errorf("pool config = %+v, want 3/20/8", cfg.Pool)
	}
}

func TestLoadConfigDefaults(t *testing.T) {
	yaml := `
backends:
  - addr: "127.0.0.1:3306"
    user: "root"
    password: ""
`
	cfg, err := loadFromString(t, yaml)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Listen.Addr != "0.0.0.0:4000" {
		t.Errorf("default listen addr = %q", cfg.Listen.Addr)
	}
	if cfg.Pool.MinAlive != 2 || cfg.Pool.MaxAlive != 10 || cfg.Pool.MaxIdle != 5 {
		t.Errorf("default pool = %+v", cfg.Pool)
	}
}

func TestLoadConfigNoBackends(t *testing.T) {
	yaml := `
listen:
  addr: "0.0.0.0:4000"
backends:
`
	_, err := loadFromString(t, yaml)
	if err == nil {
		t.Fatal("expected error for empty backends")
	}
}

func loadFromString(t *testing.T, content string) (*Config, error) {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
	return LoadConfig(path)
}

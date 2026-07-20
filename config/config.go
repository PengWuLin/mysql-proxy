package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

type AddrConfig struct {
	Addr string `yaml:"addr"`
}

type BackendConfig struct {
	Addr     string `yaml:"addr"`
	User     string `yaml:"user"`
	Password string `yaml:"password"`
}

type PoolConfig struct {
	MinAlive int `yaml:"min_alive"`
	MaxAlive int `yaml:"max_alive"`
	MaxIdle  int `yaml:"max_idle"`
}

type ProxyAuth struct {
	User     string `yaml:"user"`
	Password string `yaml:"password"`
}

type AdminConfig struct {
	Addr string `yaml:"addr"`
}

type Config struct {
	Listen   AddrConfig      `yaml:"listen"`
	Admin    AdminConfig     `yaml:"admin"`
	Proxy    ProxyAuth       `yaml:"proxy"`
	Backends []BackendConfig `yaml:"backends"`
	Pool     PoolConfig      `yaml:"pool"`
}

func LoadConfig(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read config file: %w", err)
	}

	cfg := &Config{
		Listen: AddrConfig{Addr: "0.0.0.0:4000"},
		Admin:  AdminConfig{Addr: "0.0.0.0:8080"},
		Proxy:  ProxyAuth{User: "root", Password: ""},
		Pool:   PoolConfig{MinAlive: 2, MaxAlive: 10, MaxIdle: 5},
	}

	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}

	if len(cfg.Backends) == 0 {
		return nil, fmt.Errorf("at least one backend is required")
	}

	if cfg.Listen.Addr == "" {
		cfg.Listen.Addr = "0.0.0.0:4000"
	}
	if cfg.Pool.MinAlive <= 0 {
		cfg.Pool.MinAlive = 2
	}
	if cfg.Pool.MaxAlive <= 0 {
		cfg.Pool.MaxAlive = 10
	}
	if cfg.Pool.MaxIdle <= 0 {
		cfg.Pool.MaxIdle = 5
	}

	return cfg, nil
}

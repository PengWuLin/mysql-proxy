package main

import (
	"context"
	"flag"
	"log"
	"net"
	"net/http"
	_ "net/http/pprof"

	"mysql-proxy/config"
	"mysql-proxy/proxy"

	"github.com/go-mysql-org/go-mysql/server"
)

func main() {
	configPath := flag.String("config", "config.yaml", "path to config file")
	flag.Parse()

	cfg, err := config.LoadConfig(*configPath)
	if err != nil {
		log.Fatalf("load config: %v", err)
	}

	backends := make([]proxy.BackendConfig, len(cfg.Backends))
	for i, b := range cfg.Backends {
		backends[i] = proxy.BackendConfig{
			Addr:     b.Addr,
			User:     b.User,
			Password: b.Password,
		}
	}

	pool, err := proxy.NewBackendPool(backends, proxy.PoolConfig{
		MinAlive: cfg.Pool.MinAlive,
		MaxAlive: cfg.Pool.MaxAlive,
		MaxIdle:  cfg.Pool.MaxIdle,
	})
	if err != nil {
		log.Fatalf("create backend pool: %v", err)
	}
	defer pool.Close()

	listener, err := net.Listen("tcp", cfg.Listen.Addr)
	if err != nil {
		log.Fatalf("listen %s: %v", cfg.Listen.Addr, err)
	}
	log.Printf("mysql-proxy listening on %s, %d backend(s)", cfg.Listen.Addr, pool.Len())

	// Start admin HTTP server for pprof.
	if cfg.Admin.Addr != "" {
		go func() {
			log.Printf("admin server on %s", cfg.Admin.Addr)
			if err := http.ListenAndServe(cfg.Admin.Addr, nil); err != nil {
				log.Printf("admin server: %v", err)
			}
		}()
	}

	srv := server.NewDefaultServer()
	for {
		clientConn, err := listener.Accept()
		if err != nil {
			log.Printf("accept: %v", err)
			continue
		}
		go handleClient(srv, clientConn, pool, cfg.Proxy.User, cfg.Proxy.Password)
	}
}

func handleClient(srv *server.Server, clientConn net.Conn, pool *proxy.BackendPool, proxyUser, proxyPassword string) {
	defer clientConn.Close()

	backendConn, release, err := pool.GetConn(context.Background())
	if err != nil {
		log.Printf("get backend conn: %v", err)
		return
	}
	defer release()

	h := proxy.NewHandler(backendConn)
	conn, err := srv.NewConn(clientConn, proxyUser, proxyPassword, h)
	if err != nil {
		log.Printf("new server conn: %v", err)
		return
	}
	defer conn.Close()

	for {
		if err := conn.HandleCommand(); err != nil {
			log.Printf("handle command: %v", err)
			return
		}
	}
}

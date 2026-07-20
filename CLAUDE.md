# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Build & Run

```bash
go build -o mysql-proxy.exe .
go run .
```

## Architecture

A MySQL protocol proxy server using the [`go-mysql-org/go-mysql`](https://github.com/go-mysql-org/go-mysql) library.

- **`main.go`** — Entry point. Listens on TCP port 4000, accepts a single MySQL client connection, and delegates command handling to the `go-mysql` server layer using an `EmptyHandler` (a no-op handler provided by the library).
- The proxy speaks the MySQL wire protocol — clients connect with a MySQL client (e.g., `mysql -h 127.0.0.1 -P 4000 -u root`) and the server handles the protocol handshake and command loop.

**Key dependency:** `github.com/go-mysql-org/go-mysql` provides the MySQL server framework. Custom behavior is added by replacing `server.EmptyHandler{}` with a handler implementing the `server.Handler` interface.

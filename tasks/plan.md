# Implementation Plan: MySQL 透明代理

## Dependency Graph

```
config.yaml ──► config.go ──► backend.go (Pool)
                                  │
                                  ▼
                             handler.go ──► main.go
```

**Vertical slices** — each task delivers a testable unit:

| # | Task | Depends on | Verifiable by |
|---|------|-----------|---------------|
| 1 | Config 解析 | — | `go test ./config/` |
| 2 | 后端池 (Round Robin + Pool) | Config | 单元测试：取连接轮询 |
| 3 | Handler 转发 | 后端池 | Mock 后端，验证每个 Handler 方法 |
| 4 | main 入口串联 | Handler + Config | `go build` + 启动代理 |
| 5 | 集成测试 | 全部 | 真实 MySQL 测试 |

## Task 1: Config 解析

**文件**: `config/config.go`, `config/config_test.go`, `config.yaml`

实现：
- `type Config struct` — 包含 Listen, Backends, Pool 三个 section
- `func LoadConfig(path string) (*Config, error)` — 读取并解析 YAML
- 默认值：pool 参数有合理的 zero-value fallback

**API 要点**：
```go
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

type Config struct {
    Listen   AddrConfig       `yaml:"listen"`
    Backends []BackendConfig  `yaml:"backends"`
    Pool     PoolConfig       `yaml:"pool"`
}
```

**验证**: `go test ./config/` 测试解析示例 YAML、空 backends 报错、默认值填充。

---

## Task 2: 后端池管理

**文件**: `proxy/backend.go`, `proxy/backend_test.go`

实现：
- `type BackendPool struct` — 封装 `[]*client.Pool` + 原子计数器
- `func NewBackendPool(backends []BackendConfig, poolCfg PoolConfig) (*BackendPool, error)`
  - 为每个 backend 调用 `client.NewPoolWithOptions(addr, user, password, "", poolOpts...)`
- `func (bp *BackendPool) GetConn(ctx context.Context) (*client.Conn, func(), error)`
  - Round Robin: `atomic.AddUint64(&bp.counter, 1) % len(pools)` 选池
  - 从池中 `GetConn(ctx)` 获取连接
  - 返回连接 + release 函数（调用者 defer release()）
- `func (bp *BackendPool) Close()` — 关闭所有池

**关键设计决策**：返回 `release func()` 而非直接暴露 `PutConn`，调用者无需知道如何归还。

**验证**: 单元测试 Mock 多个池验证轮询顺序。使用真实 `client.Pool` 连接本地 MySQL 验证获取/归还。

---

## Task 3: Handler 实现

**文件**: `proxy/handler.go`, `proxy/handler_test.go`

实现 `server.Handler` 接口：

```go
type handler struct {
    backend *client.Conn
}
```

每个方法的转发策略：

| 方法 | 转发到 |
|------|--------|
| `UseDB(dbName)` | `backend.UseDB(dbName)` |
| `HandleQuery(query)` | `backend.Execute(query)` |
| `HandleFieldList(table, wildcard)` | `backend.FieldList(table, wildcard)` |
| `HandleStmtPrepare(query)` | `backend.Prepare(query)` → 返回 `*client.Stmt` 作为 context |
| `HandleStmtExecute(ctx, query, args)` | `ctx.(*client.Stmt).Execute(args...)` |
| `HandleStmtClose(ctx)` | `ctx.(*client.Stmt).Close()` |
| `HandleOtherCommand(cmd, data)` | 返回 nil（静默忽略未知命令） |

**关键设计决策**：
- `HandleStmtPrepare` 返回的 context 是 `*client.Stmt`，在 `HandleStmtExecute`/`HandleStmtClose` 中类型断言取回
- 所有错误从后端直接返回给客户端，不做包装
- Handler 不持有 reference 到 BackendPool，只持有当前分配的 `*client.Conn`

**验证**: 用 mock `*client.Conn`（或真实连接）测试每个 Handler 方法。

---

## Task 4: main 入口串联

**文件**: `main.go` (改写现有文件)

```go
func main() {
    // 1. 解析命令行参数 -config
    // 2. 加载配置 LoadConfig()
    // 3. 初始化后端池 NewBackendPool()
    // 4. 监听端口 net.Listen()
    // 5. accept 循环：
    //    for each conn:
    //      go handleClient(conn)
}

func handleClient(clientConn net.Conn) {
    // 1. backend.GetConn() → backend conn
    // 2. server.NewConn(clientConn, user, password, handler{backend})
    // 3. defer release()
    // 4. for conn.HandleCommand() { ... }
}
```

**关键设计决策**：
- `server.NewConn` 已 deprecated，改用 `server.NewDefaultServer().NewConn(...)`
- 每客户端连接一个 goroutine，异常不影响其他客户端（log + return）
- 客户端断开时（HandleCommand 返回 error），退出循环并释放后端连接

**验证**: `go build` 成功，`go run .` 启动无报错。

---

## Task 5: 集成测试

**文件**: `proxy/handler_test.go`（集成测试部分）

- 启动一个真实的后端 MySQL（或使用测试容器）
- 创建 BackendPool 指向该 MySQL
- 用 `client.Connect` 模拟客户端连接代理 → 执行 SELECT 1 → 验证返回
- 验证 USE db、INSERT、SELECT 等基本流程

**验证**: `go test -v ./proxy/` 全部通过。

---

## Checkpoints

| 检查点 | 位置 | 条件 |
|--------|------|------|
| CP1 | Task 1 完成后 | 配置解析正确，`go test ./config/` PASS |
| CP2 | Task 2 完成后 | 后端池可用，单元测试 PASS |
| CP3 | Task 3 完成后 | Handler 所有方法可用，`go test ./proxy/` PASS |
| CP4 | Task 4 完成后 | `go build` 成功，代理可启动 |
| CP5 | Task 5 完成后 | 集成测试 PASS，手动 `mysql` CLI 验证通过 |

# SPEC: MySQL 透明代理

## 1. 目标

实现一个基于 Go 语言的 MySQL 透明代理，客户端连接代理后，代理将请求透明转发到后端 MySQL 服务器池。

- **目标用户**：需要将 MySQL 客户端流量分发到多个后端实例的场景（负载均衡、高可用）
- **核心价值**：客户端无感知的多后端 MySQL 接入；连接池复用后端连接以降低开销

## 2. 核心功能

| 功能 | 描述 |
|------|------|
| 透明转发 | 代理接收 MySQL 客户端连接，将全部 MySQL 命令（QUERY、INIT_DB、STMT_PREPARE/EXECUTE/CLOSE、FIELD_LIST 等）透明转发到后端 MySQL |
| 多后端轮询 | 维护多个后端 MySQL 地址，按 Round Robin 策略分发客户端连接 |
| 后端连接池 | 每个后端独立维护 `client.Pool`，复用连接避免频繁握手 |
| 配置文件 | YAML 配置文件指定代理监听地址、后端列表、认证信息、连接池参数 |
| 认证透传 | 客户端认证用户名/密码透传到后端 MySQL，不在代理层做验证（可选：代理层做认证代理） |

## 3. 命令（开发）

```bash
go build -o mysql-proxy.exe .          # 构建
go run .                                # 运行（使用默认配置）
go run . -config config.yaml           # 运行（指定配置文件）
go test ./...                           # 运行全部测试
go test -v -run TestHandler ./proxy/   # 运行单个包的测试
```

## 4. 项目结构

```
mysql-proxy/
├── main.go              # 入口：读配置 → 初始化后端池 → 监听 → accept 循环
├── config/
│   └── config.go        # YAML 配置解析（监听地址、后端列表、认证、连接池参数）
├── proxy/
│   ├── handler.go       # 实现 server.Handler：所有方法转发到后端 client.Conn
│   └── backend.go       # 后端池管理：多 client.Pool、Round Robin 选取
├── config.yaml          # 默认配置文件
├── go.mod / go.sum
└── SPEC.md
```

## 5. 技术方案

### 5.1 依赖

- **`github.com/go-mysql-org/go-mysql`** v1.16.0：提供 MySQL 协议实现
  - `server` 包：处理客户端连接协议（`server.Conn`, `server.Handler` 接口）
  - `client` 包：后端 MySQL 连接和连接池（`client.Conn`, `client.Pool`）

### 5.2 连接生命周期

```
MySQL Client ───TCP───► server.Conn ──Handler方法──► client.Conn ───TCP───► Backend MySQL
```

1. `main` accept 一个 TCP 连接（客户端）
2. 从后端池 Round Robin 选取一个后端，通过 `client.Pool.GetConn()` 获取后端连接
3. 创建一个 `server.Conn`，传入自定义 Handler（持有后端 `client.Conn`）
4. 循环 `server.Conn.HandleCommand()` 直到客户端断开
5. 归还后端连接到池 `client.Pool.PutConn()`

### 5.3 Handler 实现（`server.Handler` 接口）

每个方法直接委托给后端 `client.Conn`：

- `UseDB(dbName)` → `backend.UseDB(dbName)`
- `HandleQuery(query)` → `backend.Execute(query)`，返回 `*mysql.Result`
- `HandleFieldList(table, wildcard)` → `backend.FieldList(table, wildcard, "")`
- `HandleStmtPrepare(query)` → `backend.Prepare(query)`
- `HandleStmtExecute(ctx, query, args)` → `backend.ExecuteSelectStreaming()` 或用 prepared stmt
- `HandleStmtClose(ctx)` → 无需操作（或关闭 prepared statement）
- `HandleOtherCommand(cmd, data)` → 直接透传，返回 nil（或 `ER_UNKNOWN_ERROR`）

### 5.4 配置文件格式

```yaml
# config.yaml
listen:
  addr: "0.0.0.0:4000"

backends:
  - addr: "127.0.0.1:3306"
    user: "root"
    password: ""
  - addr: "127.0.0.1:3307"
    user: "root"
    password: ""

pool:
  min_alive: 2
  max_alive: 10
  max_idle: 5
```

### 5.5 轮询算法

后端池维护一份 `[]*client.Pool` 列表和一个原子计数器。每次有新的客户端连接，`atomic.AddUint64(&counter, 1)` 后模以下标给出行号，从对应的 `Pool` 取连接。

## 6. 测试策略

| 层级 | 测试内容 | 工具 |
|------|---------|------|
| 单元测试 | Handler 转发逻辑 → mock `client.Conn` | `testing` 标准库 |
| 集成测试 | 启动代理，用 `go-mysql/client.Conn` 连接 → 执行 SQL → 验证 | `testing` + 本地测试 MySQL 实例 |
| 手动验证 | `mysql -h 127.0.0.1 -P 4000 -u root` 连接代理执行 SQL | MySQL CLI |

## 7. 边界

### 始终做
- 使用 `server.Handler` 接口转发，不自行实现 MySQL 协议解析
- 使用 `client.Pool` 管理后端连接池，避免裸连接
- 配置变更通过修改 `config.yaml`，不硬编码
- 代理错误要记录日志（`log` / `slog`）但不中断其他连接

### 先问再做
- 增加认证逻辑（代理层验证用户/密码而非透传）
- 支持 TLS/SSL
- 增加 SQL 审计、日志记录等中间件功能
- 支持读写分离

### 永远不做
- 不修改 `go-mysql` 库的内部逻辑（依赖最终开源版本）
- 不在代理层缓存查询结果

# Task List

> **Execution order**: Tasks must be completed in ID order (1→5), each task depends on the previous.

## Task 1: Config 解析

**Status**: pending

**Files to create**:
- `config/config.go`
- `config/config_test.go`
- `config.yaml`

**Acceptance Criteria**:
- [ ] `config.yaml` 可被正确解析为 `Config` 结构体
- [ ] backends 为空时报错
- [ ] pool 参数未配置时使用默认值（min=2, max=10, idle=5）
- [ ] Listen addr 未配置时默认 "0.0.0.0:4000"
- [ ] `go test ./config/` PASS

**Verification**:
```bash
go test -v ./config/
```

---

## Task 2: 后端池管理

**Status**: pending (blocked by Task 1)

**Files to create**:
- `proxy/backend.go`
- `proxy/backend_test.go`

**Acceptance Criteria**:
- [ ] `NewBackendPool` 为每个配置的后端创建独立的 `client.Pool`
- [ ] `GetConn` 按 Round Robin 顺序从不同后端池取连接
- [ ] `GetConn` 返回的 release 函数正确归还连接到池
- [ ] `Close` 关闭所有后端池
- [ ] 只有一个后端时，每次取连接都来自同一个池
- [ ] `go test -v ./proxy/ -run TestBackend` PASS

**Verification**:
```bash
go test -v -run TestBackend ./proxy/
```

---

## Task 3: Handler 转发

**Status**: pending (blocked by Task 2)

**Files to create**:
- `proxy/handler.go`
- `proxy/handler_test.go`

**Acceptance Criteria**:
- [ ] 实现 `server.Handler` 接口的所有 7 个方法
- [ ] `UseDB` 转发正确
- [ ] `HandleQuery` 转发 SELECT/INSERT/UPDATE 并返回结果
- [ ] `HandleFieldList` 转发
- [ ] `HandleStmtPrepare` 返回 `*client.Stmt` 作为 context
- [ ] `HandleStmtExecute` 从 context 恢复 `*client.Stmt` 并执行
- [ ] `HandleStmtClose` 关闭 prepared statement
- [ ] `HandleOtherCommand` 返回 nil（静默处理）
- [ ] `go test -v ./proxy/` PASS

**Verification**:
```bash
go test -v ./proxy/
```

---

## Task 4: main 入口串联

**Status**: pending (blocked by Task 3)

**Files to modify**:
- `main.go`
- `go.mod` (可能需要 `gopkg.in/yaml.v3` 依赖)

**Acceptance Criteria**:
- [ ] 支持 `-config` 命令行参数指定配置文件路径
- [ ] 启动时加载配置，初始化后端池
- [ ] Accept 循环：每个客户端连接一个 goroutine
- [ ] 客户端断开时正确释放后端连接
- [ ] 单个客户端异常不影响其他客户端
- [ ] `go build` 成功
- [ ] `go run . -config config.yaml` 启动无报错

**Verification**:
```bash
go build -o mysql-proxy.exe .
.\mysql-proxy.exe -config config.yaml
# 另一个终端: mysql -h 127.0.0.1 -P 4000 -u root
```

---

## Task 5: 集成测试

**Status**: pending (blocked by Task 4)

**Files to modify**:
- `proxy/handler_test.go` (添加集成测试)
- 或新建 `integration_test.go`

**Acceptance Criteria**:
- [ ] 集成测试覆盖：SELECT 1, USE db, CREATE TABLE, INSERT, SELECT, DROP TABLE
- [ ] 测试使用真实后端 MySQL（本地或 Docker）
- [ ] `go test -v ./...` 全部 PASS

**Verification**:
```bash
go test -v ./...
```

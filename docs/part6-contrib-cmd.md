# 第六部分：扩展模块与 CLI 工具（contrib/*、cmd/gf）

## 目录

1. [架构概览](#1-架构概览)
2. [数据库驱动（contrib/drivers/）](#2-数据库驱动contribdrivers)
3. [NoSQL 适配器（contrib/nosql/）](#3-nosql-适配器contribnosql)
4. [服务注册中心（contrib/registry/）](#4-服务注册中心contribregistry)
5. [配置适配器（contrib/config/）](#5-配置适配器contribconfig)
6. [链路追踪（contrib/trace/）](#6-链路追踪contribtrace)
7. [指标监控（contrib/metric/otelmetric）](#7-指标监控contribmetricotelmetric)
8. [gRPC 增强（contrib/rpc/grpcx/）](#8-grpc-增强contribrpcgrpcx)
9. [SDK 适配器（contrib/sdk/httpclient/）](#9-sdk-适配器contribsdkhttpclient)
10. [CLI 工具（cmd/gf/）](#10-cli-工具cmdgf)

---

## 1. 架构概览

GoFrame 采用**核心 + 插件**的模块化架构。`contrib/` 下的每个扩展模块都是独立的 Go Module（拥有自己的 `go.mod`），遵循统一的设计模式：

- **实现核心接口**：每个 contrib 模块实现核心框架定义的接口（如 `gdb.Driver`、`gsvc.Registry`、`gcfg.Adapter`）
- **通过 `init()` 自注册**：利用 Go 的 `init()` 机制在导入时自动注册到全局工厂
- **用户按需导入**：只需 `_ "github.com/gogf/gf/contrib/drivers/mysql/v2"` 空导入即可激活

### 核心接口映射表

| contrib 模块 | 实现的核心接口 | 注册函数 |
|---|---|---|
| `contrib/drivers/*` | `gdb.Driver` | `gdb.Register()` |
| `contrib/nosql/redis` | `gredis.Adapter` | `gredis.RegisterAdapterFunc()` |
| `contrib/registry/*` | `gsvc.Registry` | 手动创建实例 |
| `contrib/config/*` | `gcfg.Adapter` | 手动创建实例 |
| `contrib/trace/*` | `gtrace.TracerProvider` | `otel.SetTracerProvider()` |
| `contrib/metric/*` | `gmetric.Provider` | `gmetric.SetGlobalProvider()` |
| `contrib/rpc/grpcx` | `gsvc.Registry`（可选） | 自动/手动 |

---

## 2. 数据库驱动（contrib/drivers/）

### 2.1 统一架构

所有数据库驱动共享相同的设计模式：

1. 定义 `Driver` 结构体，内嵌 `*gdb.Core`
2. 实现 `gdb.Driver` 接口（`New()`、`Open()`、`GetChars()` 等）
3. 在 `init()` 中调用 `gdb.Register()` 注册驱动名
4. 覆写差异方法处理 SQL 方言

`Driver` 结构体定义（以 MySQL 为例，`contrib/drivers/mysql/mysql.go:18-20`）：

```go
type Driver struct {
    *gdb.Core
}
```

### 2.2 MySQL 驱动深度分析

**注册机制**（`contrib/drivers/mysql/mysql.go:26-37`）：

```go
func init() {
    var (
        driverObj   = New()
        driverNames = g.SliceStr{"mysql", "mariadb", "tidb"}
    )
    for _, driverName := range driverNames {
        if err = gdb.Register(driverName, driverObj); err != nil {
            panic(err)
        }
    }
}
```

一个驱动实例同时注册为 `mysql`、`mariadb`、`tidb` 三个名称（兼容旧版用法）。

**连接字符串构建**（`contrib/drivers/mysql/mysql_open.go:38-59`）：

DSN 格式：`user:pass@protocol(host:port)/dbname?charset=utf8&loc=Asia%2FShanghai`

**引用字符**：反引号 `` ` ``

**表列表查询**（`contrib/drivers/mysql/mysql_tables.go:17-32`）：直接使用 `SHOW TABLES`

**字段信息查询**（`contrib/drivers/mysql/mysql_table_fields.go:63-103`）：使用 `SHOW FULL COLUMNS FROM table`，MariaDB 模式下使用 `information_schema.COLUMNS` 联查

**使用示例**：

```go
import (
    _ "github.com/gogf/gf/contrib/drivers/mysql/v2"
    "github.com/gogf/gf/v2/frame/g"
)

// 在配置文件中配置 type 为 "mysql"
db := g.DB()
result, err := db.Model("user").Where("id", 1).One()
```

### 2.3 PostgreSQL 驱动深度分析

**注册机制**（`contrib/drivers/pgsql/pgsql.go:28-32`）：

```go
func init() {
    if err := gdb.Register(`pgsql`, New()); err != nil {
        panic(err)
    }
}
```

#### 关键差异处理

##### (1) 占位符转换（`contrib/drivers/pgsql/pgsql_do_filter.go:19-57`）

PostgreSQL 使用 `$1, $2, $3...` 而非 MySQL 的 `?`：

```go
func (d *Driver) DoFilter(ctx context.Context, link gdb.Link, sql string, args []any) (newSql string, newArgs []any, err error) {
    var index int
    // 将 ? 转为 $x
    newSql, err = gregex.ReplaceStringFunc(`\?`, sql, func(s string) string {
        index++
        return fmt.Sprintf(`$%d`, index)
    })
    // 处理 jsonb 中的 '?' 操作符（不应被替换）
    // LIMIT x,y → LIMIT y OFFSET x
    // INSERT OR IGNORE → INSERT ... ON CONFLICT DO NOTHING
    return d.Core.DoFilter(ctx, link, newSql, newArgs)
}
```

##### (2) Upsert 语法（`contrib/drivers/pgsql/pgsql_format_upsert.go:21-93`）

PostgreSQL 使用 `ON CONFLICT ... DO UPDATE SET ... = EXCLUDED.*`

##### (3) 类型系统（`contrib/drivers/pgsql/pgsql_convert.go:72-124`）

| PostgreSQL 类型 | Go 类型 |
|---|---|
| `int2`, `int4` | `int` |
| `int8` | `int64` |
| `uuid` | `uuid.UUID` |
| `_int4` | `[]int32` |
| `_varchar`, `_text` | `[]string` |
| `bytea` | `[]byte` |

##### (4) LastInsertId 兼容

PostgreSQL 的 `lib/pq` 驱动不原生支持 `LastInsertId`，GoFrame 通过自定义 `Result` 包装处理。

**连接字符串格式**：

```
user=postgres password='xxx' host=127.0.0.1 port=5432 dbname=test sslmode=disable
```

### 2.4 ClickHouse 驱动深度分析

**不支持的功能**：

- `InsertIgnore`
- `InsertGetId`
- `Replace`
- `Begin`（事务）

**SQL 方言转换**（`contrib/drivers/clickhouse/clickhouse_do_filter.go:19-79`）：

```go
// MySQL: UPDATE table SET field=val WHERE ...
// ClickHouse: ALTER TABLE table UPDATE field=val WHERE ...
case "UPDATE":
    newSql = fmt.Sprintf("ALTER TABLE %s UPDATE", tableName)

// MySQL: DELETE FROM table WHERE ...
// ClickHouse: ALTER TABLE table DELETE WHERE ...
case "DELETE":
    newSql = fmt.Sprintf("ALTER TABLE %s DELETE", tableName)
```

**连接字符串**：

```
clickhouse://user:password@host:port/dbname?debug=false
```

### 2.5 驱动差异对比总结

| 维度 | MySQL | PostgreSQL | ClickHouse | DM |
|---|---|---|---|---|
| 底层驱动 | `go-sql-driver/mysql` | `lib/pq` | `clickhouse-go` | `gitee.com/chunanyong/dm` |
| 引用字符 | `` ` `` | `"` | 无特殊 | `"` |
| 占位符 | `?` | `$1,$2...` | `$1,$2...` | `?` |
| Upsert | `ON DUPLICATE KEY UPDATE` | `ON CONFLICT DO UPDATE` | 不支持 | 兼容 MySQL |
| 事务 | 支持 | 支持 | **不支持** | 支持 |
| LIMIT 语法 | `LIMIT offset,count` | `LIMIT count OFFSET offset` | 同 PG | 兼容 MySQL |

### 2.6 驱动使用规范

**配置文件方式**：

```yaml
database:
  default:
    type: "mysql"
    link: "mysql:root:123456@tcp(127.0.0.1:3306)/test"
    debug: true
  pgsql:
    type: "pgsql"
    link: "user=postgres password=123456 host=127.0.0.1 port=5432 dbname=test sslmode=disable"
```

**代码方式**：

```go
import (
    _ "github.com/gogf/gf/contrib/drivers/mysql/v2"
    _ "github.com/gogf/gf/contrib/drivers/pgsql/v2"
    "github.com/gogf/gf/v2/frame/g"
)

db := g.DB()          // 使用默认数据库
dbPgsql := g.DB("pgsql")  // 使用命名数据库组
```

---

## 3. NoSQL 适配器（contrib/nosql/）

### 3.1 Redis 适配器

基于 `github.com/redis/go-redis/v9` 实现的 `gredis.Adapter` 接口，支持单机、哨兵、集群三种模式。

**注册机制**（`contrib/nosql/redis/redis.go:37-41`）：

```go
func init() {
    gredis.RegisterAdapterFunc(func(config *gredis.Config) gredis.Adapter {
        return New(config)
    })
}
```

**客户端创建**（`contrib/nosql/redis/redis.go:44-85`）：

自动根据配置选择模式：
- **哨兵模式**：`MasterName` 非空时使用 `redis.NewFailoverClient`
- **集群模式**：多个地址或 `config.Cluster` 为 true 时使用 `redis.NewClusterClient`
- **单机模式**：默认使用 `redis.NewClient`

**连接池默认值**：

```go
const (
    defaultPoolMaxIdle     = 10
    defaultPoolMaxActive   = 100
    defaultPoolIdleTimeout = 10 * time.Second
    defaultMaxRetries      = -1  // 禁用重试
)
```

**OpenTelemetry 集成**：每次 `Do()` 操作自动创建 Trace Span。

**数据类型分组**：

| 文件 | 职责 |
|---|---|
| `redis_group_string.go` | String 操作 |
| `redis_group_hash.go` | Hash 操作 |
| `redis_group_list.go` | List 操作 |
| `redis_group_set.go` | Set 操作 |
| `redis_group_sorted_set.go` | Sorted Set 操作 |
| `redis_group_pubsub.go` | Pub/Sub 操作 |
| `redis_group_script.go` | Lua 脚本 |
| `redis_group_generic.go` | 通用操作 |
| `redis_pipeline.go` | Pipeline 批量操作 |

**使用示例**：

```go
import (
    _ "github.com/gogf/gf/contrib/nosql/redis/v2"
    "github.com/gogf/gf/v2/frame/g"
)

redis := g.Redis()
val, err := redis.Do(ctx, "SET", "key", "value")
val, err = redis.Do(ctx, "GET", "key")
```

---

## 4. 服务注册中心（contrib/registry/）

所有注册中心实现统一的 `gsvc.Registry` 接口：

```go
type Registry interface {
    Register(ctx context.Context, service Service) (Service, error)
    Deregister(ctx context.Context, service Service) error
    Search(ctx context.Context, in SearchInput) ([]Service, error)
    Watch(ctx context.Context, key string) (Watcher, error)
}
```

### 4.1 Etcd 注册中心深度分析

**客户端创建**（`contrib/registry/etcd/etcd.go:63-117`）：

支持 `ip:port@username:password` 格式的认证。

**服务注册**（`contrib/registry/etcd/etcd_registrar.go:22-63`）：

基于 Lease + KeepAlive 机制：

```go
func (r *Registry) doRegisterLease(ctx context.Context, service gsvc.Service) error {
    grant, err := r.lease.Grant(ctx, int64(r.keepaliveTTL.Seconds()))
    _, err = r.client.Put(ctx, key, value, etcd3.WithLease(grant.ID))
    keepAliceCh, err := r.client.KeepAlive(context.Background(), grant.ID)
    go r.doKeepAlive(service, grant.ID, keepAliceCh)
    return nil
}
```

**KeepAlive 重连机制**（`contrib/registry/etcd/etcd_registrar.go:75-117`）：

当 KeepAlive 通道关闭时，自动重试注册（间隔 1-3 秒随机）。

### 4.2 Consul 注册中心深度分析

**服务注册**（`contrib/registry/consul/consul.go:96-157`）：

Consul 使用 HTTP API + TTL 健康检查：

```go
reg := &api.AgentServiceRegistration{
    ID:      serviceID,
    Name:    service.GetName(),
    Tags:    []string{service.GetVersion()},
    Check: &api.AgentServiceCheck{
        TTL:                            DefaultTTL.String(),    // 20s
        DeregisterCriticalServiceAfter: "1m",
    },
}
```

### 4.3 注册中心对比

| 维度 | etcd | Consul |
|---|---|---|
| 心跳机制 | Lease + KeepAlive（gRPC 流） | TTL Health Check（HTTP 轮询） |
| 元数据存储 | KV value（直接存储） | Service Meta（JSON 序列化） |
| 重连机制 | 自动重试（1-3s 随机间隔） | 无自动重连 |

### 4.4 使用规范

```go
import (
    "github.com/gogf/gf/contrib/registry/etcd/v2"
    "github.com/gogf/gf/v2/net/gsvc"
)

// etcd 注册中心
gsvc.SetRegistry(etcd.New("127.0.0.1:2379"))

// HTTP Server 自动注册
s := g.Server()
s.SetRegistry(etcd.New("127.0.0.1:2379"))
s.Run()
```

其他注册中心（nacos、polaris、zookeeper、file）遵循相同的模式。

---

## 5. 配置适配器（contrib/config/）

所有配置适配器实现统一的 `gcfg.Adapter` 接口：

```go
type Adapter interface {
    Available(ctx context.Context, resource ...string) bool
    Get(ctx context.Context, pattern string) (value any, err error)
    Data(ctx context.Context) (data map[string]any, err error)
}
```

同时实现 `gcfg.WatcherAdapter` 接口以支持配置热更新。

### 5.1 Apollo 配置适配器深度分析

**核心结构**（`contrib/config/apollo/apollo.go:51-56`）：

```go
type Client struct {
    config   Config
    client   agollo.Client
    value    *g.Var                // 缓存的配置数据（内部为 *gjson.Json）
    watchers *gcfg.WatcherRegistry
}
```

**配置获取**：采用懒加载 + 内存缓存策略。

**配置更新**：Apollo 支持多 Namespace，配置更新时遍历所有 Namespace 的 cache。

### 5.2 Nacos 配置适配器深度分析

**Watch 机制**（`contrib/config/nacos/nacos.go:133-152`）：

```go
func (c *Client) addWatcher() error {
    c.config.ConfigParam.OnChange = func(namespace, group, dataId, data string) {
        _ = c.doUpdate(data)      // 更新内存缓存
        c.notifyWatchers(adapterCtx.Ctx)
    }
    return c.client.ListenConfig(c.config.ConfigParam)
}
```

### 5.3 配置适配器对比

| 维度 | Apollo | Nacos |
|---|---|---|
| 客户端库 | `agollo/v4` | `nacos-sdk-go/v2` |
| 配置组织 | Namespace（逗号分隔多命名空间） | Group + DataId |
| Watch 机制 | `AddChangeListener` | `ListenConfig` 回调 |
| 缓存策略 | 遍历 Namespace cache → gjson | 直接解析 → gjson |

### 5.4 使用规范

```go
// Apollo 配置
adapter, err := apollo.New(ctx, apollo.Config{
    AppID:         "myapp",
    IP:            "http://127.0.0.1:8080",
    Cluster:       "default",
    NamespaceName: "application",
    Watch:         true,
})
g.Cfg().SetAdapter(adapter)

// Nacos 配置
adapter, err := nacos.New(ctx, nacos.Config{
    ServerConfigs: []constant.ServerConfig{
        {IpAddr: "127.0.0.1", Port: 8848},
    },
    ConfigParam: vo.ConfigParam{
        DataId: "config.json",
        Group:  "DEFAULT_GROUP",
    },
    Watch: true,
})
g.Cfg().SetAdapter(adapter)
```

---

## 6. 链路追踪（contrib/trace/）

### 6.1 架构

两个追踪模块（`otlpgrpc` 和 `otlphttp`）共享相同的设计，区别仅在导出协议（gRPC vs HTTP）。

### 6.2 gRPC 导出器（contrib/trace/otlpgrpc/）

**初始化**（`contrib/trace/otlpgrpc/otlpgrpc.go:36-109`）：

```go
func Init(serviceName, endpoint, traceToken string) (func(ctx context.Context), error) {
    traceExp, err := otlptrace.New(ctx, otlptracegrpc.NewClient(
        otlptracegrpc.WithInsecure(),
        otlptracegrpc.WithEndpoint(endpoint),
        otlptracegrpc.WithHeaders(map[string]string{"Authentication": traceToken}),
        otlptracegrpc.WithCompressor(gzip.Name),
    ))
    tracerProvider := trace.NewTracerProvider(
        trace.WithSampler(trace.AlwaysSample()),
        trace.WithResource(res),
        trace.WithSpanProcessor(trace.NewBatchSpanProcessor(traceExp)),
    )
    otel.SetTracerProvider(tracerProvider)
    return shutdownFunc, nil
}
```

### 6.3 使用示例

```go
import "github.com/gogf/gf/contrib/trace/otlpgrpc/v2"

func main() {
    shutdown, err := otlpgrpc.Init("my-service", "localhost:4317", "my-token")
    if err != nil {
        panic(err)
    }
    defer shutdown(context.Background())
    // 后续所有 HTTP/gRPC/Redis 请求自动链路追踪
}
```

---

## 7. 指标监控（contrib/metric/otelmetric）

### 7.1 架构

基于 OpenTelemetry Metric SDK 实现 `gmetric.Provider` 接口。

### 7.2 Provider 创建

```go
func newProvider(options ...Option) (gmetric.Provider, error) {
    provider := &localProvider{
        MeterProvider: metric.NewMeterProvider(config.MetricOptions()...),
    }
    provider.initializeMetrics(gmetric.GetAllMetrics())
    provider.initializeCallback(gmetric.GetRegisteredCallbacks())
    // 内置 Go 运行时指标
    if config.IsBuiltInMetricsEnabled() {
        runtime.Start(runtime.WithMeterProvider(provider))
    }
}
```

### 7.3 支持的指标类型

| 文件 | 指标类型 |
|---|---|
| `otelmetric_meter_counter_performer.go` | Counter（计数器） |
| `otelmetric_meter_updown_counter_performer.go` | UpDownCounter（可增减计数器） |
| `otelmetric_meter_histogram_performer.go` | Histogram（直方图） |
| `otelmetric_meter_observable_counter_performer.go` | Observable Counter（异步计数器） |
| `otelmetric_meter_observable_gauge_performer.go` | Observable Gauge（异步仪表） |
| `otelmetric_meter_observable_updown_counter_performer.go` | Observable UpDownCounter |

### 7.4 使用示例

```go
import "github.com/gogf/gf/contrib/metric/otelmetric/v2"

provider := otelmetric.MustProvider(
    otelmetric.WithBuiltInMetrics(),
    otelmetric.WithReader(prometheus.New()),
)
provider.SetAsGlobal()
```

---

## 8. gRPC 增强（contrib/rpc/grpcx/）

### 8.1 模块概述

`grpcx` 提供了对标准 gRPC 的增强封装，包括服务端自动注册/注销、内置拦截器链、客户端服务发现和负载均衡。

### 8.2 服务端

**创建服务器**（`contrib/rpc/grpcx/grpcx_grpc_server.go:49-88`）：

默认拦截器链（按顺序）：

1. `UnaryTracing` — 链路追踪
2. `UnaryLogger` — 日志
3. `UnaryRecover` — Panic 恢复
4. `UnaryAllowNilRes` — 允许 nil 响应
5. `UnaryError` — 错误码转换

**服务注册**：启动时自动注册到配置的注册中心，关闭时自动注销。

**优雅关闭**：监听系统信号，先注销服务再停止 gRPC 服务器。

### 8.3 客户端

**创建连接**（`contrib/rpc/grpcx/grpcx_grpc_client.go:31-78`）：

支持两种模式：
- **服务名模式**：传入服务名，自动通过服务发现解析地址
- **地址模式**：传入 `host:port`，直接连接

### 8.4 拦截器

| 拦截器 | 位置 | 功能 |
|---|---|---|
| `UnaryTracing` / `StreamTracing` | 服务端/客户端 | OpenTelemetry 链路追踪 |
| `UnaryLogger` | 服务端 | 请求日志记录 |
| `UnaryRecover` | 服务端 | Panic 恢复 |
| `UnaryError` | 服务端/客户端 | GoFrame 错误码 → gRPC 状态码 |
| `UnaryAllowNilRes` | 服务端 | 允许 nil 响应 |
| `UnaryValidate` | 服务端（可选） | 请求参数校验 |

### 8.5 使用示例

```go
import "github.com/gogf/gf/contrib/rpc/grpcx/v2"

// 服务端
server := grpcx.Server.New()
pb.RegisterGreeterServer(server.Server, &Greeter{})
server.Run()

// 客户端
conn := grpcx.Client.MustNewGrpcClientConn("my-grpc-service")
client := pb.NewGreeterClient(conn)
resp, err := client.SayHello(ctx, &pb.HelloRequest{Name: "world"})
```

---

## 9. SDK 适配器（contrib/sdk/httpclient/）

### 9.1 模块概述

提供基于 `gclient` 的 SDK HTTP 客户端，支持结构化请求/响应映射、TLS 指纹模拟。

**核心结构**（`contrib/sdk/httpclient/httpclient.go:27-30`）：

```go
type Client struct {
    *gclient.Client
    Handler
}
```

**请求方法**：

```go
func (c *Client) Request(ctx context.Context, req, res any) error {
    method := gmeta.Get(req, gtag.Method).String()
    path := gmeta.Get(req, gtag.Path).String()
    // 自动从结构体元数据读取 HTTP 方法和路径
}
```

**路径参数处理**：将 `/users/{id}` 中的 `{id}` 替换为结构体字段值。

**Engine 类型**：

| Engine | 用途 |
|---|---|
| `native` | 标准 Go HTTP Transport |
| `chrome` | Chrome TLS 指纹模拟 |
| `utls` | uTLS 指纹模拟 |

---

## 10. CLI 工具（cmd/gf/）

### 10.1 架构

CLI 工具基于 GoFrame 的 `gcmd` 包构建，采用对象化命令定义方式。

### 10.2 完整子命令列表

| 命令 | 用途 | 关键选项 |
|---|---|---|
| `gf init` | 项目初始化 | `-m`(mono), `-a`(mono-app), `-u`(更新), `-r`(远程模板), `-i`(交互式) |
| `gf run` | 热编译运行 | `-p`(输出路径), `-w`(监听路径), `-i`(忽略模式) |
| `gf build` | 交叉编译 | `--os`, `--arch`, `--ps`(打包目录), `--cgo` |
| `gf gen dao` | 生成 DAO/DO/Entity | 数据库表结构映射 |
| `gf gen ctrl` | 生成控制器 | API 接口 → 控制器 |
| `gf gen service` | 生成服务接口 | 模块服务接口提取 |
| `gf gen pb` | 生成 Protobuf | proto 文件生成 |
| `gf gen pbentity` | 生成 PB Entity | 数据库表 → proto entity |
| `gf gen enums` | 生成枚举 | Go 枚举定义 |
| `gf pack` | 资源打包 | 将目录打包为 Go 文件 |
| `gf docker` | Docker 构建 | 构建容器镜像 |
| `gf up` | 启动依赖服务 | docker-compose |
| `gf env` | 环境信息 | Go/GF 版本 |
| `gf fix` | 修复兼容性 | 版本迁移 |
| `gf version` | 版本信息 | |
| `gf install` | 安装 CLI | |

### 10.3 热编译（gf run）

**核心机制**（`cmd/gf/internal/cmd/cmd_run.go:102-179`）：

1. 首次编译并运行 `go build -o output main.go`
2. 使用 `gfsnotify` 监听 `.go` 文件变更
3. 检测到变更后重新编译，杀死旧进程，启动新进程
4. 使用 `dirty` 标志位防止短时间内多次重编译（防抖 1500ms）

**默认忽略目录**：`node_modules`、`vendor`、`.*`（隐藏目录）、`_*`（下划线开头目录）。

### 10.4 代码生成（gf gen）

**`gen dao`** 生成的代码结构：

```
internal/
├── model/
│   ├── entity/     # 数据库表对应的 Entity 结构体
│   └── do/         # 数据操作对象（包含所有字段）
├── dao/            # 数据访问对象（包含 CRUD 方法）
└── logic/          # 业务逻辑层
```

**`gen ctrl`**：根据 API 定义文件自动生成控制器骨架。

**`gen service`**：从 logic 层自动提取服务接口。

### 10.5 项目初始化（gf init）

**三种模板**：

- `template-single`：单体项目
- `template-mono`：多模块项目
- `template-mono-app`：多模块子应用

支持从 GitHub 仓库下载远程模板：

```bash
gf init my-project -r github.com/gogf/template-single
gf init -i  # 交互式选择模板
```

### 10.6 使用示例

```bash
# 初始化项目
gf init myapp
gf init myapp -m          # mono-repo
gf init myapp -i          # 交互式

# 热编译运行
gf run main.go
gf run main.go -w internal,api

# 生成 DAO
gf gen dao

# 交叉编译
gf build main.go --os linux --arch amd64

# 打包资源
gf pack public,template packed/packed.go

# Docker 构建
gf docker
```

---

## 附录：扩展模块开发规范

### 新增数据库驱动

1. 创建 `contrib/drivers/newdb/` 目录，添加 `go.mod`
2. 定义 `Driver` 结构体内嵌 `*gdb.Core`
3. 实现 `gdb.Driver` 接口（必须方法：`New()`、`Open()`、`GetChars()`）
4. 在 `init()` 中调用 `gdb.Register("newdb", New())`
5. 根据需要覆写差异方法（`DoFilter`、`Tables`、`TableFields`、`FormatUpsert` 等）
6. 添加测试文件和 README

### 新增注册中心

1. 实现 `gsvc.Registry` 接口（`Register`、`Deregister`、`Search`、`Watch`）
2. 实现 `gsvc.Watcher` 接口
3. 使用编译时校验：`var _ gsvc.Registry = (*Registry)(nil)`
4. 提供 `New()` 构造函数

### 新增配置适配器

1. 实现 `gcfg.Adapter` 接口（`Available`、`Get`、`Data`）
2. 实现 `gcfg.WatcherAdapter` 接口（配置热更新）
3. 使用 `gcfg.WatcherRegistry` 管理监听器

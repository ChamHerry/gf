# 第四部分：操作系统与服务层（os/）

## 目录

- [1. os/gcfg — 配置管理（适配器模式）](#1-osgcfg--配置管理适配器模式)
- [2. os/glog — 结构化日志（处理器链 + 日志轮转）](#2-osglog--结构化日志处理器链--日志轮转)
- [3. os/gcache — 通用缓存（适配器 + 内存 LRU）](#3-osgcache--通用缓存适配器--内存-lru)
- [4. os/gsession — 会话管理（Manager / Session / Storage 三层）](#4-osgsession--会话管理manager--session--storage-三层)
- [5. os/gview — 模板引擎（Go template 增强）](#5-osgview--模板引擎go-template-增强)
- [6. os/gcmd — 命令行框架（命令树 + 参数解析）](#6-osgcmd--命令行框架命令树--参数解析)
- [7. i18n/gi18n — 国际化管理](#7-i18ngi18n--国际化管理)
- [8. os/gfile — 文件操作工具集](#8-osgfile--文件操作工具集)
- [9. os/gctx — 上下文管理（OTel 链路传播）](#9-osgctx--上下文管理otel-链路传播)
- [10. os/gproc — 进程管理与进程间通信](#10-osgproc--进程管理与进程间通信)
- [11. os/gtimer — 高性能定时器（堆优先队列）](#11-osgtimer--高性能定时器堆优先队列)
- [12. os/gcron — Cron 定时任务（基于 gtimer）](#12-osgcron--cron-定时任务基于-gtimer)
- [13. os/gfsnotify — 文件系统监听（fsnotify 封装）](#13-osgfsnotify--文件系统监听fsnotify-封装)
- [14. os/gmetric — 指标度量（OTel Metrics 抽象）](#14-osgmetric--指标度量otel-metrics-抽象)
- [15. os/gres — 资源嵌入（打包文件系统）](#15-osgres--资源嵌入打包文件系统)
- [16. os/gstructs — 结构体反射工具](#16-osgstructs--结构体反射工具)
- [17. os/gtime — 时间扩展（Time 包装器）](#17-osgtime--时间扩展time-包装器)

---

## 1. os/gcfg — 配置管理（适配器模式）

### 1.1 包概述

`gcfg` 提供统一的配置管理能力，采用**适配器模式**（Adapter Pattern）实现配置源的抽象与解耦。框架内置文件适配器（`AdapterFile`），第三方可通过实现 `Adapter` 接口扩展 Apollo、Nacos、Polaris 等远程配置中心。

源码位置：`os/gcfg/`

包注释见 `os/gcfg/gcfg.go:7`：
> Package gcfg provides flexible and hierarchical configuration managing functionalities.

### 1.2 核心设计

```
gcfg/
├── gcfg.go                  — 公共 API + 全局实例管理
├── gcfg_adapter.go          — Adapter 接口定义
├── gcfg_adapter_file.go     — 文件适配器（内置默认）
├── gcfg_loader.go           — 泛型 Loader[T]（配置绑定到结构体）
└── gcfg_z_unit_test.go      — 单元测试
```

**三层架构**：

```
┌─────────────────────────────────────────┐
│         公共 API（gcfg.go）              │
│   Instance() / SetAdapter() / Get()     │
├─────────────────────────────────────────┤
│         Adapter 接口（adapter.go）        │
│   Get / Set / Available / GetContent    │
├──────────────┬──────────────────────────┤
│ AdapterFile  │  AdapterApollo (contrib) │
│  (内置默认)   │  AdapterNacos (contrib)   │
└──────────────┴──────────────────────────┘
```

### 1.3 核心类型和接口

#### Adapter 接口（`os/gcfg/gcfg_adapter.go`）

```go
type Adapter interface {
    // Available checks and returns whether configuration is available.
    Available(ctx context.Context, resource string) bool

    // Get retrieves and returns value by specified `pattern`.
    Get(ctx context.Context, pattern string) (Value, error)

    // Set sets value with specified `pattern`.
    Set(ctx context.Context, pattern string, value any) error

    // Search returns the searched result for specified `pattern`.
    Search(ctx context.Context, pattern string) (string, error)

    // GetContent returns the content for specified resource.
    GetContent(ctx context.Context, resource string) (string, error)

    // SetContent sets the content for specified resource.
    SetContent(ctx context.Context, resource string, content string) error

    // Clear removes all caches.
    Clear(ctx context.Context) error
}
```

#### Value 接口（`os/gcfg/gcfg.go`）

```go
type Value interface {
    Val() any
    IsNil() bool
    Bool() bool
    Int() int
    Int64() int64
    Uint() uint
    Uint64() uint64
    Float32() float32
    Float64() float64
    String() string
    Strings() []string
    Map() map[string]any
    // ... 更多 gconv 委托方法
}
```

`Value` 接口内部委托 `gvar.Var`，所有类型转换由 `gconv` 完成。

#### 全局实例（`os/gcfg/gcfg.go`）

```go
var (
    localInstance  *Instance      // 全局默认实例
    localInstances = gmap.New()   // 按名称管理的实例映射
)
```

### 1.4 文件适配器三级搜索

`AdapterFile` 支持三级配置搜索策略（`os/gcfg/gcfg_adapter_file.go`）：

```
1. 默认目录       → 当前工作目录 / config 子目录
2. 自定义搜索路径 → 通过 SetPath / AddPath 设置
3. 资源系统       → gres 打包的资源文件
```

配置文件支持格式由 `gjson` 决定：`json` / `yaml` / `toml` / `ini` / `xml` / `properties`，通过文件扩展名自动识别。

### 1.5 泛型 Loader[T]（配置绑定）

`os/gcfg/gcfg_loader.go` 提供泛型配置绑定功能，类似 Spring Boot 的 `@ConfigurationProperties`：

```go
type Loader[T any] struct {
    Options LoaderOptions
}

type LoaderOptions struct {
    Prefix   string         // 配置前缀，如 "database.default"
    Paths    []string       // 搜索路径
    Tags     []string       // 优先标签顺序，如 ["json", "yaml"]
    Panic    bool           // 配置解析失败时是否 panic
}
```

#### 关键方法

| 方法 | 说明 |
|------|------|
| `NewLoader[T](opts)` | 创建 Loader 实例 |
| `Load(ctx)` | 加载配置并绑定到 `*T` |
| `LoadWithParent(ctx, parent)` | 带父对象加载 |

### 1.6 关键公共方法

| 方法 | 说明 |
|------|------|
| `Instance(name...)` | 获取/创建配置实例 |
| `SetAdapter(adapter)` | 设置全局适配器 |
| `GetAdapter()` | 获取当前适配器 |
| `Get(ctx, pattern)` | 按点分路径获取配置值 |
| `Set(ctx, pattern, value)` | 运行时设置配置值 |
| `GetWithEnv(ctx, pattern)` | 获取配置，优先读取环境变量 |
| `GetContent(ctx, file)` | 获取原始配置文件内容 |
| `MustGet(ctx, pattern)` | Get 的无错误版本 |
| `Available(ctx, file)` | 检查配置文件是否可用 |

### 1.7 使用示例

```go
package main

import (
    "context"
    "fmt"
    "github.com/gogf/gf/v2/frame/g"
)

func main() {
    ctx := context.Background()

    // 读取配置（config/config.yaml 中 database.host）
    host := g.Cfg().MustGet(ctx, "database.host").String()
    port := g.Cfg().MustGet(ctx, "database.port").Int()
    fmt.Printf("DB: %s:%d\n", host, port)

    // 支持环境变量覆盖
    // GF_DATABASE_HOST=127.0.0.1
    hostWithEnv := g.Cfg().GetWithEnv(ctx, "database.host").String()

    // 泛型 Loader 绑定
    // type DatabaseConfig struct {
    //     Host string `json:"host"`
    //     Port int    `json:"port"`
    // }
    // loader := gcfg.NewLoader[gcfg.DatabaseConfig](gcfg.LoaderOptions{
    //     Prefix: "database.default",
    // })
    // cfg, err := loader.Load(ctx)
}
```

### 1.8 模块间依赖

```
gcfg → gjson (配置解析)
     → gfile (文件路径)
     → gres  (资源文件搜索)
     → gvar  (Value 类型委托)
     → gconv (类型转换)
     → gmap  (实例管理)
```

---

## 2. os/glog — 结构化日志（处理器链 + 日志轮转）

### 2.1 包概述

`glog` 是 GoFrame 的高性能结构化日志组件，支持**处理器链**（Handler Chain）、**多级别日志**、**异步写入**、**文件轮转**（按大小/时间）、以及 OpenTelemetry 链路追踪集成。

源码位置：`os/glog/`

包注释见 `os/glog/glog.go:7`：
> Package glog implements powerful and easy logging APIs.

### 2.2 核心设计

```
glog/
├── glog.go                      — ILogger 接口 + 公共 API + 全局实例
├── glog_logger.go               — Logger 核心实现（日志写入引擎）
├── glog_logger_handler.go       — Handler 链机制
├── glog_logger_config.go        — Config 配置结构体
├── glog_logger_rotate.go        — 日志文件轮转
├── glog_logger_level.go         — 日志级别常量与转换
├── glog_logger_api.go           — 级别快捷方法（Debug/Info/Warn/Error/Fatal/Panic）
├── glog_logger_output.go        — 输出控制（stdout/file/both）
├── glog_logger_color.go         — 终端颜色输出
├── glog_level.go                — 级别常量定义
├── glog_handler.go              — Handler 接口定义
└── glog_print.go                — Stack/Notice/Deprecated 等特殊输出
```

### 2.3 核心类型和接口

#### ILogger 接口（`os/glog/glog.go`）

```go
type ILogger interface {
    // 日志配置
    SetConfig(config Config) error
    SetLevel(level int) *Logger
    SetLevelStr(levelStr string) error
    SetLevelPrefix(level int, prefix string)
    GetLevel() int
    GetLevelPrefix(level int) string

    // Handler 链
    SetHandlers(handlers ...Handler)
    AddHandlers(handlers ...Handler)
    GetHandlers() []Handler

    // 文件与输出
    SetPath(path string) error
    GetPath() string
    SetStdoutPrint(enabled bool)
    SetWriter(writer io.Writer)
    GetWriter() io.Writer

    // 日志级别快捷方法
    Debug(ctx context.Context, v ...any)
    Info(ctx context.Context, v ...any)
    Notice(ctx context.Context, v ...any) // Notice 级别，介于 Info 和 Warn 之间
    Warning(ctx context.Context, v ...any)
    Error(ctx context.Context, v ...any)
    Critical(ctx context.Context, v ...any)
    Panic(ctx context.Context, v ...any)
    Fatal(ctx context.Context, v ...any)

    // 格式化
    Debugf(ctx context.Context, format string, v ...any)
    Errorf(ctx context.Context, format string, v ...any)
    // ... 其他级别同理

    // 分类输出
    Cat(category string) *Logger

    // 日志轮转
    SetRotateExpire(expire time.Duration) *Logger
    SetRotateSize(size int64) *Logger
    SetRotateBackupLimit(limit int) *Logger
    SetRotateBackupExpire(expire time.Duration) *Logger
    SetRotateBackupCompress(compress int) *Logger
    SetRotateCheckInterval(interval time.Duration) *Logger
}
```

#### Handler 函数类型（`os/glog/glog_handler.go`）

```go
type Handler func(r *HandlerParam)
```

Handler 采用**中间件模式**，通过 `HandlerParam.Next()` 传递控制权：

```go
type HandlerParam struct {
    Ctx      context.Context  // 上下文
    Time     time.Time        // 日志时间
    Level    int              // 日志级别
    LevelStr string           // 级别字符串
    Content  any              // 日志内容（格式化前）
    Buffer   *bytes.Buffer    // 格式化后的输出缓冲区
    Logger   *Logger          // 关联的 Logger
    Handler  Handler          // 当前 handler 链
    index    int              // 当前 handler 索引（用于 Next）
}

// Next executes the next handler in the chain.
func (p *HandlerParam) Next() {
    if p.index < len(p.handlers) {
        p.index++
        p.handlers[p.index-1](p)
    }
}
```

#### 日志级别（`os/glog/glog_level.go`）

```go
const (
    LEVEL_ALL  = LEVEL_PANI | LEVEL_FATA | LEVEL_ERRO | LEVEL_WARN | LEVEL_INFO | LEVEL_NOTI | LEVEL_DEBU
    LEVEL_DEBU = 1 << 0  // Debug
    LEVEL_INFO = 1 << 1  // Info
    LEVEL_NOTI = 1 << 2  // Notice
    LEVEL_WARN = 1 << 3  // Warning
    LEVEL_ERRO = 1 << 4  // Error
    LEVEL_FATA = 1 << 5  // Fatal
    LEVEL_PANI = 1 << 6  // Panic
)
```

级别通过**位掩码**管理，支持按位组合：`glog.LEVEL_INFO | glog.LEVEL_ERRO`。

#### Config 配置（`os/glog/glog_logger_config.go`）

```go
type Config struct {
    Level            int           // 日志级别（位掩码）
    Levels           string        // 级别字符串，如 "all"
    Path             string        // 日志文件目录
    File             string        // 日志文件名
    TimeFormat       string        // 时间格式
    StdoutPrint      bool          // 是否同时打印到 stdout
    RotateExpire     time.Duration // 按时间轮转
    RotateSize       int64         // 按大小轮转（字节）
    RotateBackupLimit int          // 轮转备份数量
    RotateBackupExpire time.Duration // 备份过期时间
    RotateBackupCompress int       // 备份压缩级别
    RotateCheckInterval time.Duration // 轮转检查间隔
    Writer           io.Writer     // 自定义 Writer
    StStatus         int           // StackTrace 状态
    StSkip           int           // StackTrace 跳过帧数
    StFilter         *gregex.Regex // StackTrace 过滤
    Flags            int           // 额外标志
    Prefix           string        // 日志前缀
}
```

### 2.4 异步日志写入

`glog_logger.go` 内部维护一个**单工作协程**的异步写入池：

```go
// 异步日志池（保持日志顺序的单 worker）
type asyncLoggerPool struct {
    // 使用 channel 缓冲日志任务
    // 单 worker 协程按顺序处理，保证日志的时间顺序性
}
```

设计决策：使用**单个 worker**而非 worker pool，目的是**保证日志写入顺序**。异步池通过 `channel` 接收日志任务，worker 按接收顺序依次写入文件。

### 2.5 日志文件轮转（`os/glog/glog_logger_rotate.go`）

| 轮转维度 | 配置字段 | 说明 |
|---------|---------|------|
| 按大小 | `RotateSize` | 文件超过指定字节数时轮转 |
| 按时间 | `RotateExpire` | 文件创建超过指定时长后轮转 |
| 备份数量 | `RotateBackupLimit` | 保留的备份文件数量 |
| 备份过期 | `RotateBackupExpire` | 超过此时间的备份自动删除 |
| 备份压缩 | `RotateBackupCompress` | gzip 压缩级别（0=不压缩） |
| 检查间隔 | `RotateCheckInterval` | 轮转检查频率 |

轮转通过 `gtimer` 定时检查触发。

### 2.6 OTel 链路追踪集成

日志写入时自动从 `context.Context` 提取 OTel Span 信息：

```go
// 如果 ctx 中存在 trace.SpanContext，自动注入：
// - trace_id
// - span_id
```

这使日志能够与分布式追踪系统关联。

### 2.7 使用示例

```go
package main

import (
    "context"
    "github.com/gogf/gf/v2/os/glog"
)

func main() {
    ctx := context.Background()

    // 基础用法
    glog.Info(ctx, "server started")
    glog.Errorf(ctx, "failed to connect: %s", "timeout")

    // 分类日志（输出到独立文件）
    glog.Cat("sql").Info(ctx, "SELECT * FROM users")

    // 自定义 Handler
    glog.SetHandlers(func(p *glog.HandlerParam) {
        // 前置处理：添加 request_id
        p.Buffer.WriteString("[REQ-123] ")
        p.Next() // 继续链上的下一个 handler
    })

    // 日志轮转配置
    logger := glog.New()
    logger.SetPath("/var/log/myapp")
    logger.SetRotateSize(100 * 1024 * 1024) // 100MB
    logger.SetRotateBackupLimit(7)
    logger.SetLevel(glog.LEVEL_INFO | glog.LEVEL_ERRO)
}
```

### 2.8 模块间依赖

```
glog → gfile      (文件路径操作)
     → gtime      (时间格式化)
     → gctx       (OTel trace 提取)
     → gerror     (堆栈信息)
     → gjson      (JSON 格式输出)
     → gtimer     (轮转定时检查)
     → container/gtype (并发安全配置)
     → container/garray (handler 链管理)
```

---

## 3. os/gcache — 通用缓存（适配器 + 内存 LRU）

### 3.1 包概述

`gcache` 提供统一的缓存抽象，内置内存缓存适配器（支持 LRU 淘汰策略和 TTL 过期），并可通过 `Adapter` 接口扩展 Redis 等远程缓存。

源码位置：`os/gcache/`

包注释见 `os/gcache/gcache.go:7`：
> Package gcache provides high-performance concurrent-safe cache facilities.

### 3.2 核心设计

```
gcache/
├── gcache.go                       — Cache 结构体 + 公共 API
├── gcache_adapter.go               — Adapter 接口定义
├── gcache_adapter_memory.go        — 内存适配器核心
├── gcache_adapter_memory_lru.go    — LRU 淘汰策略实现
├── gcache_adapter_redis.go         — Redis 适配器（接口定义，实现在 contrib/nosql/gredis）
└── gcache_z_unit_test.go           — 单元测试
```

**适配器分层**：

```
┌─────────────────────────────────────┐
│         Cache（gcache.go）           │
│  委托给底层 Adapter 接口              │
├─────────────────────────────────────┤
│         Adapter 接口                  │
│  Set/Get/Remove/Contains/Clear/...  │
├──────────────┬──────────────────────┤
│ AdapterMemory│  AdapterRedis        │
│ (LRU + TTL)  │  (委托给 gredis)      │
└──────────────┴──────────────────────┘
```

### 3.3 核心类型和接口

#### Adapter 接口（`os/gcache/gcache_adapter.go`）

```go
type Adapter interface {
    // Set sets cache with key-value pair and TTL.
    Set(ctx context.Context, key any, value any, duration time.Duration) error

    // SetMap batch sets cache with key-value pairs.
    SetMap(ctx context.Context, data map[any]any, duration time.Duration) error

    // Get retrieves and returns the associated value of given `key`.
    Get(ctx context.Context, key any) (*gvar.Var, error)

    // GetOrSet returns the value if exists, or sets and returns it.
    GetOrSet(ctx context.Context, key any, value any, duration time.Duration) (*gvar.Var, error)

    // Contains checks whether `key` exists in the cache.
    Contains(ctx context.Context, key any) (bool, error)

    // Remove deletes one or multiple keys.
    Remove(ctx context.Context, keys ...any) (*gvar.Var, error)

    // Data returns all cache items as map.
    Data(ctx context.Context) (map[any]*gvar.Var, error)

    // Keys returns all keys in the cache.
    Keys(ctx context.Context) ([]any, error)

    // Values returns all values in the cache.
    Values(ctx context.Context) ([]*gvar.Var, error)

    // Size returns the number of items in the cache.
    Size(ctx context.Context) (int, error)

    // Clear clears all data of the cache.
    Clear(ctx context.Context) error

    // Close closes the cache.
    Close(ctx context.Context) error
}
```

#### Cache 结构体（`os/gcache/gcache.go`）

```go
type Cache struct {
    localAdapter Adapter // 底层适配器实例
}
```

`Cache` 是 `Adapter` 的薄封装，所有方法委托给 `localAdapter`。

### 3.4 AdapterMemory 内存适配器

#### 核心结构（`os/gcache/gcache_adapter_memory.go`）

```go
type AdapterMemory struct {
    mu            sync.RWMutex                  // 读写锁
    data          map[any]*AdapterMemoryItem    // 缓存数据
    cap           int                           // 容量上限（0=无限制）
    lru           bool                          // 是否启用 LRU
    eventList     *glist.List[*eventListElem]   // 事件列表（过期事件）
    eventListMu   sync.Mutex                    // 事件列表锁
    clearedCursor int                           // 清理游标
}

type AdapterMemoryItem struct {
    v     any           // 值
    exp   int64         // 过期时间戳（UnixNano），0=永不过期
    lruId int64         // LRU 访问 ID（仅 LRU 模式使用）
}
```

#### LRU 淘汰策略（`os/gcache/gcache_adapter_memory_lru.go`）

LRU 在**容量有限**（`cap > 0`）时自动启用：

```go
type adapterMemoryLru struct {
    mu     sync.Mutex
    cap    int                              // 容量
    data   map[any]*list.Element            // key → 链表节点
    list   *list.List                       // 双向链表（最近访问在前）
    nextId int64                            // 下一个 LRU ID
}
```

- 新增/访问时将元素移到链表头部
- 容量超限时淘汰链表尾部元素

#### 异步过期清理

过期清理采用**事件驱动 + 定时扫描**混合策略：

1. **事件驱动**：写入带 TTL 的 key 时，将过期事件添加到 `eventList`
2. **定时扫描**：通过 `gtimer` 定期检查 `eventList`，批量清理已过期的 key

```go
type eventListElem struct {
    key any
    exp int64
}
```

### 3.5 关键方法

| 方法 | 说明 |
|------|------|
| `New()` / `NewWithAdapter(adapter)` | 创建缓存实例 |
| `SetAdapter(adapter)` | 设置底层适配器 |
| `GetAdapter()` | 获取当前适配器 |
| `Set(ctx, key, value, duration)` | 设置缓存 |
| `Get(ctx, key)` | 获取缓存 |
| `GetOrSet(ctx, key, value, duration)` | 获取或设置 |
| `GetVar(ctx, key)` | 获取并返回 `*gvar.Var` |
| `Contains(ctx, key)` | 检查 key 是否存在 |
| `Remove(ctx, keys...)` | 删除缓存 |
| `Size(ctx)` | 获取缓存大小 |
| `Data(ctx)` | 获取所有数据 |
| `Keys(ctx)` / `Values(ctx)` | 获取所有 key/value |
| `Clear(ctx)` | 清空缓存 |
| `Close(ctx)` | 关闭缓存 |

### 3.6 使用示例

```go
package main

import (
    "context"
    "time"
    "github.com/gogf/gf/v2/os/gcache"
)

func main() {
    ctx := context.Background()

    // 默认实例（内存缓存）
    gcache.Set(ctx, "key1", "value1", time.Minute)
    val, _ := gcache.Get(ctx, "key1")

    // 自定义内存缓存（带 LRU 容量限制）
    cache := gcache.New()
    cache.SetCap(1000) // 最多缓存 1000 条，超出时 LRU 淘汰

    // GetOrSet 模式
    cache.GetOrSet(ctx, "key2", "value2", 10*time.Second)

    // 使用 Redis 适配器（需要导入 contrib/nosql/gredis）
    // redisCache := gcache.New()
    // redisCache.SetAdapter(gcache.NewAdapterRedis(gredis.New()))
}
```

### 3.7 模块间依赖

```
gcache → container/gvar   (Value 返回类型)
       → container/glist  (事件列表)
       → container/gtype  (并发安全计数)
       → gtimer           (过期清理定时器)
       → gconv            (类型转换)
```

---

## 4. os/gsession — 会话管理（Manager / Session / Storage 三层）

### 4.1 包概述

`gsession` 提供统一的 Web 会话管理能力，采用 **Manager / Session / Storage 三层架构**，支持文件存储、Redis 存储和内存存储。Session 采用懒加载设计，仅在首次访问时才从 Storage 读取。

源码位置：`os/gsession/`

包注释见 `os/gsession/gsession.go:7`：
> Package gsession provides high-performance session management.

### 4.2 核心设计

```
gsession/
├── gsession.go                    — SessionId 常量 + 公共函数
├── gsession_manager.go            — Manager（会话管理器）
├── gsession_session.go            — Session（会话实例，懒加载）
├── gsession_storage.go            — Storage 接口定义
├── gsession_storage_file.go       — 文件存储实现
├── gsession_storage_memory.go     — 内存存储实现
├── gsession_storage_redis.go      — Redis 存储实现（委托 contrib/nosql/gredis）
└── gsession_z_unit_test.go        — 单元测试
```

**三层架构**：

```
┌──────────────────────────────────┐
│   Manager（管理器）                │
│   存储策略 + TTL 配置 + SessionId  │
├──────────────────────────────────┤
│   Session（会话实例）              │
│   懒加载 + dirty 标记 + 批量操作   │
├──────────────────────────────────┤
│   Storage（存储接口）              │
│   File / Memory / Redis          │
└──────────────────────────────────┘
```

### 4.3 核心类型和接口

#### Storage 接口（`os/gsession/gsession_storage.go`）

```go
type Storage interface {
    // New creates a session id and stores to storage.
    New(ctx context.Context, sessionId string, data map[string]any, ttl time.Duration) error

    // Get retrieves session data by key.
    Get(ctx context.Context, sessionId string, key string) (any, error)

    // Set sets session data by key-value pair.
    Set(ctx context.Context, sessionId string, key string, value any, ttl time.Duration) error

    // Remove removes key from session.
    Remove(ctx context.Context, sessionId string, key string) error

    // Data retrieves all session data.
    Data(ctx context.Context, sessionId string) (map[string]any, error)

    // GetSize returns the number of keys in session.
    GetSize(ctx context.Context, sessionId string) (int, error)

    // RemoveAll removes all session data.
    RemoveAll(ctx context.Context, sessionId string) error

    // UpdateTTL updates the TTL of session.
    UpdateTTL(ctx context.Context, sessionId string, ttl time.Duration) error
}
```

#### Manager 结构体（`os/gsession/gsession_manager.go`）

```go
type Manager struct {
    storage        Storage       // 底层存储
    ttl            time.Duration // 默认会话 TTL
    sessionIdName  string        // Cookie 中的 SessionId 字段名
}
```

#### Session 结构体（`os/gsession/gsession_session.go`）

```go
type Session struct {
    id        string         // Session ID
    manager   *Manager       // 关联的管理器
    request   *ghttp.Request // HTTP 请求
    dirty     bool           // 是否有未保存的修改
    data      map[string]any // 会话数据（懒加载缓存）
    inited    bool           // 是否已初始化
    mu        sync.Mutex     // 互斥锁
}
```

### 4.4 懒加载机制

`Session` 采用**懒加载**设计：

1. `Session` 创建时不读取数据
2. 首次调用 `Get/Set/All` 等方法时触发 `init()`，从 Storage 加载数据
3. 使用 `dirty` 标记记录是否有未保存的修改
4. 在 HTTP 响应写入前（通过中间件钩子），将 `dirty` 的数据写回 Storage

### 4.5 关键方法

**Manager**：

| 方法 | 说明 |
|------|------|
| `New(storage, ttl, sessionIdName)` | 创建管理器 |
| `NewSession(request)` | 为 HTTP 请求创建会话 |
| `SetStorage(storage)` | 更换存储策略 |

**Session**：

| 方法 | 说明 |
|------|------|
| `Id()` | 获取 Session ID |
| `Set(key, value)` | 设置会话数据 |
| `Get(key)` | 获取会话数据 |
| `Remove(key)` | 删除会话数据 |
| `All()` / `Data()` | 获取所有会话数据 |
| `Size()` | 获取数据条数 |
| `Clear()` | 清空会话 |
| `Destroy()` | 销毁会话 |

### 4.6 使用示例

```go
package main

import (
    "github.com/gogf/gf/v2/frame/g"
    "github.com/gogf/gf/v2/net/ghttp"
    "github.com/gogf/gf/v2/os/gsession"
    "time"
)

func main() {
    s := g.Server()

    // 使用 Redis 作为会话存储
    // s.SetSessionStorage(gsession.NewStorageRedis(gredis.New()))
    // s.SetSessionMaxAge(30 * time.Minute)

    s.BindHandler("/login", func(r *ghttp.Request) {
        r.Session.Set("user_id", 123)
        r.Session.Set("username", "gf")
        r.Response.Writeln("logged in")
    })

    s.BindHandler("/profile", func(r *ghttp.Request) {
        uid := r.Session.Get("user_id").Int()
        name := r.Session.Get("username").String()
        r.Response.Writelnf("User: %d, Name: %s", uid, name)
    })

    s.Run()
}
```

### 4.7 模块间依赖

```
gsession → os/gcache       (内存存储适配器)
         → os/gtime        (TTL 计算)
         → os/gfile        (文件存储路径)
         → encoding/gjson  (序列化)
         → container/gvar  (Get 返回值)
         → container/gtype (并发安全)
         → util/gconv      (类型转换)
```

---

## 5. os/gview — 模板引擎（Go template 增强）

### 5.1 包概述

`gview` 基于 Go 标准库 `text/template` 和 `html/template` 构建，扩展了 40+ 内置模板函数、支持资源嵌入（gres）、文件热更新（gfsnotify）、以及多路径搜索。

源码位置：`os/gview/`

包注释见 `os/gview/gview.go:7`：
> Package gview provides template engine functionalities.

### 5.2 核心设计

```
gview/
├── gview.go              — View 结构体 + 全局实例 + 公共 API
├── gview_config.go       — Config 配置 + 路径管理
├── gview_buildin.go      — 40+ 内置模板函数
├── gview_parse.go        — 模板解析引擎
└── gview_z_unit_test.go  — 单元测试
```

### 5.3 核心类型

#### View 结构体（`os/gview/gview.go`）

```go
type View struct {
    mu            sync.Mutex                    // 互斥锁
    config        Config                        // 配置
    searchPaths   *garray.StrArray              // 模板搜索路径列表
    funcMap       map[string]any                // 自定义模板函数
    fileCacheMap  *gmap.StrAnyMap               // 文件内容缓存
}
```

#### Config 配置（`os/gview/gview_config.go`）

```go
type Config struct {
    Paths       []string       `json:"paths"`       // 搜索路径列表
    Data        map[string]any `json:"data"`        // 全局模板变量
    DefaultFile string         `json:"defaultFile"` // 默认模板文件
    Delimiters  []string       `json:"delimiters"`  // 自定义定界符
    AutoEncode  bool           `json:"autoEncode"`  // 自动 HTML 编码（防 XSS）
    I18nManager *gi18n.Manager `json:"-"`           // 国际化管理器
}
```

### 5.4 内置模板函数（`os/gview/gview_buildin.go`）

| 函数 | 说明 |
|------|------|
| `eq` / `ne` / `lt` / `gt` / `lte` / `gte` | 比较运算 |
| `text` / `html` / `htmlencode` / `htmldecode` | 文本处理 |
| `date` / `dateformat` | 时间格式化 |
| `compare` / `substr` / `strlimit` | 字符串操作 |
| `concat` | 字符串拼接 |
| `replace` / `trim` | 字符串替换/修剪 |
| `add` / `sub` / `mul` / `div` / `mod` | 算术运算 |
| `implode` / `explode` | 字符串与数组互转 |
| `json` / `json_encode` / `json_decode` | JSON 编解码 |
| `dump` | 调试输出 |
| `map` | 创建 map |
| `plus` / `minus` | 加减 |
| `icontains` / `contain` | 包含检查 |
| `config` | 读取配置值 |
| `i18n` / `t` | 国际化翻译 |
| `iq` | 安全输出（防 XSS） |

### 5.5 模板解析流程

`os/gview/gview_parse.go` 中的解析流程：

1. **搜索模板文件**：依次检查 gres 资源、绝对路径、搜索路径
2. **文件缓存**：使用 `fileCacheMap` 缓存文件内容，通过 `gfsnotify` 自动失效
3. **模板编译**：使用 `text/template` 或 `html/template`（根据 `AutoEncode` 配置）
4. **函数注册**：合并内置函数和自定义函数
5. **变量合并**：合并全局变量（`config.Data`）和传入变量
6. **渲染输出**：执行模板并返回结果

### 5.6 热更新

通过 `gfsnotify` 监听模板文件目录：

```go
// 文件变更时自动清除缓存
gfsnotify.AddPath(path, func(event *gfsnotify.Event) {
    view.fileCacheMap.Clear()
    templates.Clear() // 清除编译后的模板缓存
})
```

### 5.7 关键方法

| 方法 | 说明 |
|------|------|
| `Instance()` | 获取全局默认 View 实例 |
| `New()` | 创建新 View |
| `Parse(ctx, file, params)` | 解析模板文件 |
| `ParseContent(ctx, content, params)` | 解析模板字符串 |
| `SetPath(path)` | 设置模板目录（替换） |
| `AddPath(path)` | 添加搜索路径 |
| `SetDelimiters(left, right)` | 设置定界符 |
| `BindFunc(name, function)` | 注册自定义函数 |
| `BindFuncMap(funcMap)` | 批量注册函数 |
| `Assign(key, value)` | 设置全局变量 |
| `Assigns(params)` | 批量设置全局变量 |
| `SetDefaultFile(file)` | 设置默认模板文件 |
| `SetAutoEncode(enable)` | 启用/禁用 XSS 防护 |
| `SetI18n(manager)` | 绑定国际化管理器 |

### 5.8 使用示例

```go
package main

import (
    "context"
    "github.com/gogf/gf/v2/os/gview"
)

func main() {
    ctx := context.Background()
    view := gview.New()

    // 注册自定义函数
    view.BindFunc("truncate", func(s string, n int) string {
        if len(s) > n {
            return s[:n] + "..."
        }
        return s
    })

    // 解析模板内容
    content := `Hello {{.name}}, today is {{date .timestamp "Y-m-d"}}`
    result, _ := view.ParseContent(ctx, content, gview.Params{
        "name":      "GoFrame",
        "timestamp": gtime.Now().Unix(),
    })
    fmt.Println(result) // Hello GoFrame, today is 2026-06-13
}
```

### 5.9 模块间依赖

```
gview → text/template, html/template (标准库模板引擎)
      → os/gfile       (文件搜索)
      → os/gres        (资源文件读取)
      → os/gfsnotify   (模板热更新)
      → os/gspath      (路径搜索)
      → i18n/gi18n     (国际化支持)
      → encoding/gjson (JSON 模板函数)
      → util/gconv     (类型转换)
      → container/gmap (文件缓存)
```

---

## 6. os/gcmd — 命令行框架（命令树 + 参数解析）

### 6.1 包概述

`gcmd` 提供完整的 CLI 应用框架，支持**命令树**（父子命令）、参数解析（短选项/长选项）、OTel 链路追踪、以及子命令自动发现。

源码位置：`os/gcmd/`

包注释见 `os/gcmd/gcmd.go:7`：
> Package gcmd provides console operations, like options/arguments parsing and commands running.

### 6.2 核心设计

```
gcmd/
├── gcmd.go                    — 公共 API + 常量定义
├── gcmd_command.go            — Command 结构体（命令树节点）
├── gcmd_command_run.go        — 命令执行引擎
├── gcmd_parser.go             — 参数解析器
└── gcmd_z_unit_test.go        — 单元测试
```

### 6.3 核心类型

#### Command 结构体（`os/gcmd/gcmd_command.go`）

```go
type Command struct {
    Name          string        // 命令名称
    Brief         string        // 简短描述
    Usage         string        // 使用说明
    UsageFunc     func()        // 自定义 usage 输出
    Description   string        // 详细描述
    Func          Function      // 处理函数（无额外返回值）
    FuncWithValue FuncWithValue // 处理函数（带返回值）
    Arguments     []Argument    // 参数定义
    Options       []Option      // 选项定义
    HelpFunc      func()        // 自定义帮助函数
    Examples      string        // 示例说明
    Additional    []string      // 额外信息
    parent        *Command      // 父命令
    commands      []*Command    // 子命令列表
    once          sync.Once     // 初始化保护
}
```

#### Function 和 FuncWithValue

```go
// Function 是基本的命令处理函数
type Function func(ctx context.Context, parser *Parser) (err error)

// FuncWithValue 带返回值的命令处理函数
type FuncWithValue func(ctx context.Context, parser *Parser) (out any, err error)
```

#### Argument 和 Option

```go
type Argument struct {
    Name   string // 参数名称
    Brief  string // 简短描述
    IsArg  bool   // 是否为位置参数
}

type Option struct {
    Name      string // 选项名称（支持逗号分隔别名，如 "name,n"）
    Brief     string // 简短描述
    Type      string // 类型（"bool" / "string" / "int" 等）
    Default   string // 默认值
    IsArg     bool   // 是否为位置参数
    Required  bool   // 是否必须
    BriefOnly bool   // 是否仅在帮助中显示
}
```

#### Parser 结构体（`os/gcmd/gcmd_parser.go`）

```go
type Parser struct {
    option           ParserOption      // 解析选项
    parsedArgs       []string          // 已解析的位置参数
    parsedOptions    map[string]string // 已解析的选项值
    passedOptions    map[string]bool   // 用户传入的选项
    supportedOptions map[string]bool   // 支持的选项（name:needArg）
    commandFuncMap   map[string]func() // 命令函数映射
}

type ParserOption struct {
    CaseSensitive bool // 选项是否区分大小写
    Strict        bool // 是否严格模式（非法选项时报错）
}
```

### 6.4 命令执行流程

`os/gcmd/gcmd_command_run.go` 中的执行流程：

1. **参数解析**：调用 `Parser.Parse` 解析 `os.Args`
2. **命令匹配**：在命令树中找到匹配的子命令
3. **OTel Span 创建**：为命令执行创建 trace span
4. **中间件执行**：依次执行注册的中间件
5. **处理函数调用**：调用 `Command.Func` 或 `Command.FuncWithValue`
6. **错误处理**：将返回的 error 转换为 gerror 并输出
7. **帮助信息**：必要时输出帮助信息

### 6.5 关键方法

| 方法 | 说明 |
|------|------|
| `New(name)` | 创建根命令 |
| `AddCommand(commands...)` | 添加子命令 |
| `Run(ctx)` | 执行命令树 |
| `GetArg(name)` | 获取位置参数 |
| `GetArgAll()` | 获取所有位置参数 |
| `GetOpt(name)` | 获取选项值 |
| `GetOptAll()` | 获取所有选项 |
| `Parse(supportedOptions, option...)` | 解析命令行参数 |
| `ParseArgs(args, supportedOptions, option...)` | 解析指定参数 |
| `ParserFromCtx(ctx)` | 从 context 获取 Parser |

### 6.6 使用示例

```go
package main

import (
    "context"
    "github.com/gogf/gf/v2/os/gcmd"
)

func main() {
    root := gcmd.Command{
        Name:  "myapp",
        Brief: "My CLI application",
    }

    // 添加子命令
    root.AddCommand(&gcmd.Command{
        Name:  "serve",
        Brief: "Start the server",
        Func: func(ctx context.Context, parser *gcmd.Parser) error {
            port := parser.GetOpt("port", "8080").String()
            println("Server running on :" + port)
            return nil
        },
    })

    // 执行
    root.Run(context.Background())
}

// $ myapp serve --port 9090
// Server running on :9090
```

### 6.7 模块间依赖

```
gcmd → internal/command (全局参数初始化)
      → gerror          (错误处理)
      → gctx            (OTel span 创建)
      → gstr/gregex     (参数名称解析)
      → gconv           (类型转换)
      → container/gvar  (参数值返回)
```

---

## 7. i18n/gi18n — 国际化管理

### 7.1 包概述

`gi18n` 提供国际化（i18n）和本地化（l10n）支持。基于 JSON 配置文件管理多语言翻译，支持变量插值、热更新和资源嵌入。

> **注意**：`gi18n` 位于 `i18n/gi18n/` 而非 `os/` 下，但它与 `os/` 模块（gview、gres、gfsnotify）紧密集成。

源码位置：`i18n/gi18n/`

包注释见 `i18n/gi18n/gi18n.go:7`：
> Package gi18n implements internationalization and localization.

### 7.2 核心设计

```
gi18n/
├── gi18n.go              — 公共 API（委托给默认 Manager 实例）
├── gi18n_manager.go      — Manager 核心实现
└── gi18n_z_unit_test.go  — 单元测试
```

### 7.3 核心类型

#### Manager 结构体（`i18n/gi18n/gi18n_manager.go`）

```go
type Manager struct {
    mu       sync.RWMutex                   // 读写锁
    data     map[string]map[string]string   // 翻译数据 [language][key]value
    pattern  string                         // 变量插值的正则模式
    pathType pathType                       // 路径类型（none/normal/gres）
    options  Options                        // 配置选项
}

type Options struct {
    Path       string          // i18n 文件目录
    Language   string          // 默认语言
    Delimiters []string        // 变量定界符（默认 {# #}）
    Resource   *gres.Resource  // 资源嵌入
}
```

**路径类型枚举**：

```go
const (
    pathTypeNone   pathType = "none"    // 无配置路径
    pathTypeNormal pathType = "normal"  // 文件系统
    pathTypeGres   pathType = "gres"    // 打包资源
)
```

### 7.4 多语言文件搜索

默认搜索以下目录（`i18n/gi18n/gi18n_manager.go:60`）：

```go
var searchFolders = []string{
    "manifest/i18n",
    "manifest/config/i18n",
    "i18n",
}
```

每个语言对应一个 JSON 文件（如 `en.json`、`zh-CN.json`），格式为：

```json
{
    "hello": "Hello, {#name}!",
    "welcome": "Welcome to {#app}"
}
```

### 7.5 变量插值

使用正则表达式（基于配置的 Delimiters）进行变量插值：

```go
// 默认模式：{#variable_name}
pattern = `{#(.+?)}`
```

### 7.6 关键方法

| 方法 | 说明 |
|------|------|
| `New(options...)` | 创建 Manager |
| `Instance()` | 获取全局默认实例 |
| `Translate(ctx, content)` | 翻译文本 |
| `T(ctx, content)` | Translate 的简写 |
| `Tf(ctx, format, values...)` | 翻译并格式化 |
| `GetContent(ctx, key)` | 获取指定 key 的原始翻译内容 |
| `SetLanguage(language)` | 设置语言 |
| `SetPath(path)` | 设置 i18n 文件目录 |
| `SetDelimiters(left, right)` | 设置变量定界符 |

### 7.7 使用示例

```go
package main

import (
    "context"
    "github.com/gogf/gf/v2/i18n/gi18n"
)

func main() {
    ctx := context.Background()

    // 设置语言
    gi18n.SetLanguage("zh-CN")

    // 翻译
    result := gi18n.T(ctx, "hello") // 从 i18n/zh-CN.json 读取 "hello" 对应的翻译
}
```

### 7.8 模块间依赖

```
gi18n → encoding/gjson (翻译文件解析)
       → os/gfile      (文件路径)
       → os/gres        (资源搜索)
       → os/gfsnotify   (热更新)
       → text/gregex    (变量插值)
       → util/gconv     (类型转换)
```

---

## 8. os/gfile — 文件操作工具集

### 8.1 包概述

`gfile` 是 GoFrame 最基础的文件操作工具包，提供涵盖文件/目录创建、读取、写入、搜索、权限管理、路径处理等全面功能。

源码位置：`os/gfile/gfile.go`

包注释见 `os/gfile/gfile.go:7`：
> Package gfile provides easy-to-use APIs for file operations.

### 8.2 核心常量

```go
const (
    Separator      = string(filepath.Separator) // 路径分隔符（/或\）
    DefaultPerm    = 0666                        // 默认文件权限
    DefaultPermDir = 0755                        // 默认目录权限
    DefaultPermCopy = 0755                       // 复制时的默认权限
)
```

### 8.3 功能分类与方法列表

#### 路径操作

| 方法 | 说明 |
|------|------|
| `Abs(path)` | 获取绝对路径 |
| `RealPath(path)` | 获取真实路径（解析符号链接） |
| `SelfPath()` | 获取当前可执行文件路径 |
| `SelfDir()` | 获取当前可执行文件目录 |
| `Home()` | 获取用户 Home 目录 |
| `TempDir()` | 获取系统临时目录 |
| `Dir(path)` | 获取父目录 |
| `Basename(path)` | 获取文件名（含扩展名） |
| `Name(path)` | 获取文件名（不含扩展名） |
| `Ext(path)` | 获取扩展名（含 `.`） |
| `ExtName(path)` | 获取扩展名（不含 `.`） |
| `Join(paths...)` | 拼接路径 |
| `ReplaceSeparator(path)` | 替换路径分隔符 |

#### 文件/目录创建与删除

| 方法 | 说明 |
|------|------|
| `Mkdir(path)` | 递归创建目录 |
| `MkdirPerm(path, perm)` | 创建目录（指定权限） |
| `Create(path)` | 创建空文件 |
| `Remove(path)` | 删除文件/目录 |
| `RemoveAll(path)` | 递归删除（类似 `rm -rf`） |
| `Copy(src, dst)` | 复制文件/目录 |
| `CopyFile(src, dst)` | 复制单个文件 |
| `CopyDir(src, dst)` | 复制目录 |

#### 文件读取

| 方法 | 说明 |
|------|------|
| `GetContents(path)` | 读取全部内容（`[]byte`） |
| `GetContentsWithCache(path, duration)` | 带缓存读取 |
| `GetContentsByStr(path)` | 读取为字符串 |
| `GetBytes(path, start, end)` | 读取指定范围字节 |
| `GetNextCharOffset(path, char, start)` | 查找字符位置 |
| `ReadByte(path)` | 读取单字节 |
| `ReadLines(path)` | 按行读取（`[]string`） |
| `ReadLinesBytes(path)` | 按行读取（`[][]byte`） |
| `Truncate(path, size)` | 截断文件 |

#### 文件写入

| 方法 | 说明 |
|------|------|
| `PutContents(path, content)` | 写入内容 |
| `PutContentsAppend(path, content)` | 追加写入 |
| `PutBytes(path, content)` | 写入字节 |
| `PutBytesAppend(path, content)` | 追加字节 |
| `Truncate(path, size)` | 截断/扩展文件 |

#### 文件信息

| 方法 | 说明 |
|------|------|
| `Stat(path)` | 获取文件信息 |
| `Exists(path)` | 检查文件是否存在 |
| `IsDir(path)` | 是否为目录 |
| `IsFile(path)` | 是否为文件 |
| `Size(path)` | 获取文件大小（字节） |
| `ModTime(path)` | 获取修改时间 |
| `IsAbsPath(path)` | 是否为绝对路径 |

#### 目录扫描

| 方法 | 说明 |
|------|------|
| `ScanDir(path)` | 扫描目录（不递归） |
| `ScanDirFile(path)` | 扫描文件（不递归，不含目录） |
| `ScanDirFunc(path, recursive, handler)` | 带回调扫描 |
| `ScanDirFuncFile(path, recursive, handler)` | 带回调扫描（仅文件） |

#### 权限操作

| 方法 | 说明 |
|------|------|
| `Chmod(path, mode)` | 修改权限 |
| `Chown(path, uid, gid)` | 修改所有者 |

### 8.4 使用示例

```go
package main

import (
    "fmt"
    "github.com/gogf/gf/v2/os/gfile"
)

func main() {
    // 写入文件
    gfile.PutContents("/tmp/test.txt", "Hello GoFrame")

    // 读取文件
    content := gfile.GetContents("/tmp/test.txt")
    fmt.Println(string(content))

    // 递归扫描目录
    files := gfile.ScanDir("/var/log", "*.log", true)
    fmt.Println("Log files:", files)

    // 文件信息
    if gfile.Exists("/tmp/test.txt") {
        fmt.Println("Size:", gfile.Size("/tmp/test.txt"))
        fmt.Println("ModTime:", gfile.ModTime("/tmp/test.txt"))
    }
}
```

### 8.5 模块间依赖

```
gfile → (标准库 os, io, path/filepath, bufio)
       → gtime    (时间相关)
       → gerror   (错误处理)
```

> **注意**：`gfile` 保持极少的依赖，是框架最底层的工具包之一。

---

## 9. os/gctx — 上下文管理（OTel 链路传播）

### 9.1 包概述

`gctx` 封装了 Go 标准 `context` 包，扩展了 OpenTelemetry 链路追踪的自动传播能力。所有 GoFrame 的异步操作（日志、缓存、ORM 等）均使用 `gctx` 作为上下文传递标准。

源码位置：`os/gctx/gctx.go`

包注释见 `os/gctx/gctx.go:7`：
> Package gctx wraps and provides context features.

### 9.2 核心设计

```
gctx/
├── gctx.go              — 上下文创建 + OTel 集成
└── gctx_never_done.go   — NeverDone 包装（防异步泄漏）
```

### 9.3 核心类型和常量

#### StrKey 类型

```go
// StrKey is a custom type for context key.
type StrKey string
```

使用自定义类型避免 context key 冲突。

#### CtxKey 常量

```go
const (
    CtxKeyParser StrKey = "gctx_parser" // gcmd 解析器
)
```

### 9.4 OTel 链路传播

`gctx.New()` 创建的 context 自动携带从环境变量读取的 trace 上下文：

```go
// New creates and returns a context.
// The new context automatically inherits the trace context from
// environment variables like TRACEPARENT and TRACESTATE.
func New() context.Context {
    return NewWithSpan("")
}

// NewWithSpan creates a context with a span.
func NewWithSpan(spanName string) context.Context {
    ctx := context.Background()

    // 从环境变量提取 trace 上下文（如果存在）
    ctx = trace.ContextWithRemoteSpanContext(ctx, trace.SpanContextFromContext(ctx))

    // 如果指定了 span 名称，创建新 span
    if spanName != "" {
        ctx, span := tracer.GetProvider().Tracer("gctx").Start(ctx, spanName)
        defer span.End()
        return ctx
    }
    return ctx
}
```

### 9.5 NeverDone 上下文（`os/gctx/gctx_never_done.go`）

`NeverDone` 包装使得 context **永远不会被取消**，用于异步 goroutine 中防止 context 过早取消：

```go
type neverDoneCtx struct {
    context.Context
}

func (*neverDoneCtx) Done() <-chan struct{} { return nil }
func (*neverDoneCtx) Deadline() (time.Time, bool) { return time.Time{}, false }
func (*neverDoneCtx) Err() error { return nil }

func NeverDone(ctx context.Context) context.Context {
    return &neverDoneCtx{ctx}
}
```

**使用场景**：当需要在 HTTP 响应后继续执行异步任务时，父 context 可能已被取消。使用 `NeverDone` 包装确保异步任务的 context 不会被意外取消。

### 9.6 关键方法

| 方法 | 说明 |
|------|------|
| `New()` | 创建带 OTel trace 的 context |
| `NewWithSpan(spanName)` | 创建带 span 的 context |
| `NeverDone(ctx)` | 包装为永不可取消的 context |

### 9.7 使用示例

```go
package main

import (
    "github.com/gogf/gf/v2/os/gctx"
    "github.com/gogf/gf/v2/os/glog"
)

func main() {
    // 创建带 trace 的 context
    ctx := gctx.New()
    glog.Info(ctx, "trace context created")

    // 异步任务使用 NeverDone
    asyncCtx := gctx.NeverDone(ctx)
    go func() {
        glog.Info(asyncCtx, "async task running")
    }()
}
```

### 9.8 模块间依赖

```
gctx → go.opentelemetry/otel/trace (OTel trace API)
      → go.opentelemetry/otel/baggage (OTel baggage)
```

---

## 10. os/gproc — 进程管理与进程间通信

### 10.1 包概述

`gproc` 提供进程管理和进程间通信（IPC）功能，支持进程信息查询、信号处理、子进程管理、以及基于 TCP 的跨进程通信。

源码位置：`os/gproc/`

包注释见 `os/gproc/gproc.go:7`：
> Package gproc makes the process managing and the inter process communication easily.

### 10.2 核心设计

```
gproc/
├── gproc.go            — 进程信息查询 + 公共 API
├── gproc_comm.go       — 进程间通信（IPC）
├── gproc_signal.go     — 信号处理
├── gproc_shell.go      — Shell 命令执行
├── gproc_manager.go    — 进程管理器
└── gproc_z_unit_test.go
```

### 10.3 进程信息

```go
var (
    Pid     = os.Getpid()  // 当前进程 ID
    PPid    = os.Getppid() // 父进程 ID
)
```

| 方法 | 说明 |
|------|------|
| `Pid` | 当前进程 ID（全局变量） |
| `PPid` | 父进程 ID（全局变量） |
| `IsChild()` | 当前进程是否为子进程（通过环境变量判断） |
| `Listen()` | 启动 IPC 监听 |

### 10.4 进程间通信（IPC）

#### 核心设计（`os/gproc/gproc_comm.go`）

IPC 基于 **TCP** 实现，每个进程通过 PID 到端口映射文件关联 TCP 端口：

```
1. 进程启动时分配一个空闲 TCP 端口
2. 将 {pid:port} 映射写入临时文件（~/.gf/pid_port_map/）
3. 发送消息时，通过目标 PID 查找端口并建立 TCP 连接
4. 使用 MsgRequest/MsgResponse 结构体进行通信
```

#### 消息结构

```go
type MsgRequest struct {
    SenderPid int    // 发送方 PID
    RecvPid   int    // 接收方 PID
    Group     string // 消息分组
    Data      []byte // 消息数据
}

type MsgResponse struct {
    Code int    // 响应码
    Msg  string // 消息
    Data []byte // 数据
}
```

#### 关键方法

| 方法 | 说明 |
|------|------|
| `Send(pid, data, group)` | 向指定进程发送消息 |
| `SendWithRetry(pid, data, group, retry)` | 带重试发送 |
| `Receive(group, callback)` | 注册消息接收回调 |
| `Listen()` | 启动 IPC 监听服务 |

### 10.5 Shell 执行

| 方法 | 说明 |
|------|------|
| `ShellRun(ctx, cmd)` | 执行 Shell 命令（阻塞） |
| `ShellRunAsync(ctx, cmd)` | 异步执行 Shell 命令 |
| `ShellExec(ctx, cmd)` | 执行并返回输出 |

### 10.6 信号处理

| 方法 | 说明 |
|------|------|
| `Listen()` | 启动进程信号监听 |
| `AddSigHandler(handler, signals...)` | 注册信号处理函数 |

### 10.7 使用示例

```go
package main

import (
    "github.com/gogf/gf/v2/os/gproc"
)

func main() {
    // 获取进程信息
    println("PID:", gproc.Pid)
    println("PPID:", gproc.PPid)

    // 注册信号处理
    gproc.AddSigHandler(func(sig os.Signal) {
        println("Received signal:", sig.String())
    }, syscall.SIGINT, syscall.SIGTERM)

    // 启动信号监听
    gproc.Listen()

    // 执行 Shell 命令
    gproc.ShellRun(context.Background(), "echo hello")

    // IPC：向其他进程发送消息
    // gproc.Send(targetPid, []byte("hello"), "default")
}
```

### 10.8 模块间依赖

```
gproc → os/gfile      (pid-port 映射文件)
      → os/glog       (日志记录)
      → container/gtype (并发安全)
      → container/glist (信号处理器列表)
```

---

## 11. os/gtimer — 高性能定时器（堆优先队列）

### 11.1 包概述

`gtimer` 是 GoFrame 的核心定时器组件，基于**堆优先队列**实现高性能调度。是 `gcron`（Cron 定时任务）的底层基础。支持单次触发、周期触发、单例模式等。

源码位置：`os/gtimer/`

包注释见 `os/gtimer/gtimer.go:7`：
> Package gtimer provides high-performance timer features.

### 11.2 核心设计

```
gtimer/
├── gtimer.go              — 全局实例 + Entry + 常量
├── gtimer_timer.go        — Timer 核心实现（堆调度引擎）
├── gtimer_timer_queue.go  — 优先队列（堆实现）
├── gtimer_timer_entry.go  — Entry 定时任务条目
├── gtimer_wheel.go        — 时间轮辅助
└── gtimer_z_unit_test.go
```

### 11.3 核心类型

#### Entry 定时任务条目（`os/gtimer/gtimer.go`）

```go
type Entry struct {
    timer         *Timer        // 所属 Timer
    job           JobFunc       // 任务函数
    ctx           context.Context // 上下文
    nextTime      int64         // 下次执行时间（UnixNano）
    interval      int64         // 执行间隔（纳秒）
    times         int           // 剩余执行次数（-1=无限）
    status        atomic.Int32  // 状态（Created/Running/Stopped/Closed）
    singleton     bool          // 是否单例模式
    singletonMu   sync.Mutex    // 单例互斥锁
    isSingleton   atomic.Bool   // 单例状态标志
}

type JobFunc func(ctx context.Context)
```

#### Timer 调度器（`os/gtimer/gtimer_timer.go`）

```go
type Timer struct {
    mu         sync.Mutex                    // 互斥锁
    queue      *timerPriorityQueue           // 优先队列（堆）
    options    TimerOptions                  // 配置选项
    status     atomic.Int32                  // Timer 状态
}

type TimerOptions struct {
    Interval time.Duration // 调度间隔（默认 100ms）
}
```

#### EntryStatus 常量

```go
const (
    StatusCreated  EntryStatus = 0 // 已创建
    StatusRunning  EntryStatus = 1 // 运行中
    StatusStopped  EntryStatus = 2 // 已暂停
    StatusClosed   EntryStatus = 3 // 已关闭
)
```

### 11.4 优先队列调度

`timerPriorityQueue` 基于 Go 标准库的 `container/heap` 实现：

- 队列按 `nextTime`（下次执行时间）排序
- 堆顶元素是最快到期的任务
- 调度循环每 `Interval`（默认 100ms）检查堆顶任务是否到期
- 到期任务执行后，更新 `nextTime` 并重新堆化

### 11.5 单例模式

当 `Entry.singleton` 为 `true` 时：

```go
func (entry *Entry) do(ctx context.Context) {
    if entry.isSingleton.Load() {
        return // 上一次执行还未完成，跳过
    }
    entry.isSingleton.Store(true)
    defer entry.isSingleton.Store(false)
    entry.job(ctx)
}
```

单例模式确保同一任务在上一次执行完成前不会被重复触发。

### 11.6 关键方法

**全局 API**（操作默认 Timer）：

| 方法 | 说明 |
|------|------|
| `Add(ctx, interval, job)` | 添加无限循环任务 |
| `AddEntry(ctx, interval, job, singleton, times, ...)` | 添加任务（全参数） |
| `AddSingleton(ctx, interval, job)` | 添加单例任务 |
| `AddOnce(ctx, interval, job)` | 添加单次任务 |
| `AddTimes(ctx, interval, times, job)` | 添加指定次数任务 |
| `DelayAdd(ctx, delay, interval, job)` | 延迟添加 |
| `Search(timer)` | 查找任务 |
| `Remove(entry)` | 移除任务 |

**Entry 方法**：

| 方法 | 说明 |
|------|------|
| `Start()` | 启动/恢复任务 |
| `Stop()` | 暂停任务 |
| `Close()` | 关闭并移除任务 |
| `Status()` | 获取状态 |
| `SetTimes(times)` | 设置剩余次数 |
| `Job()` | 获取任务函数 |

### 11.7 使用示例

```go
package main

import (
    "context"
    "time"
    "github.com/gogf/gf/v2/os/gtimer"
)

func main() {
    ctx := context.Background()

    // 每 2 秒执行
    entry := gtimer.Add(ctx, 2*time.Second, func(ctx context.Context) {
        println("tick")
    })

    // 单例模式（防止重叠执行）
    gtimer.AddSingleton(ctx, 1*time.Second, func(ctx context.Context) {
        time.Sleep(2 * time.Second) // 模拟耗时操作
        println("singleton tick")
    })

    // 单次执行
    gtimer.AddOnce(ctx, 5*time.Second, func(ctx context.Context) {
        println("once")
    })

    time.Sleep(10 * time.Second)
    entry.Stop()
}
```

### 11.8 模块间依赖

```
gtimer → container/gtype  (并发安全状态)
       → container/glist  (队列管理)
```

> **注意**：`gtimer` 刻意保持极少的依赖，因为它是框架的基础设施，被 `gcron`、`glog`（轮转检查）、`gcache`（过期清理）等模块依赖。

---

## 12. os/gcron — Cron 定时任务（基于 gtimer）

### 12.1 包概述

`gcron` 基于 `gtimer` 构建定时任务调度器，支持标准 **crontab 表达式**（5 段式或 6 段式），提供单例模式、单次执行等高级特性。

源码位置：`os/gcron/gcron.go`

包注释见 `os/gcron/gcron.go:7`：
> Package gcron provides a cron pattern job scheduler.

### 12.2 核心设计

`gcron` 在 `gtimer` 之上增加了一层 **crontab 表达式解析器**，将 cron pattern 转换为 gtimer 的 `interval` 和 `nextTime`：

```
┌──────────────────────────┐
│      gcron（调度入口）     │
│  Add("*/5 * * * *", fn)  │
│         ↓ 解析 cron       │
│   Schedule{Next, Expr}   │
│         ↓ 注册到          │
├──────────────────────────┤
│      gtimer（底层引擎）    │
│  Entry{nextTime, job}    │
└──────────────────────────┘
```

### 12.3 Cron 表达式

支持标准 5 段式 crontab：

```
* * * * *
│ │ │ │ │
│ │ │ │ └── 星期 (0-7, 0和7都是周日)
│ │ │ └───── 月份 (1-12)
│ │ └─────── 日   (1-31)
│ └───────── 时   (0-23)
└─────────── 分   (0-59)
```

支持的语法：
- `*` — 任意值
- `*/n` — 每 n 个单位
- `1,3,5` — 列表
- `1-5` — 范围

### 12.4 核心类型

#### Schedule 结构体

```go
type Schedule struct {
    Expr   string // cron 表达式原始字符串
    Next   int64  // 下次执行时间（UnixNano）
    prev   int64  // 上次执行时间
}
```

#### 全局实例

```go
var (
    defaultCron = New() // 默认 Cron 实例
)
```

#### Cron 结构体

```go
type Cron struct {
    timer     *gtimer.Timer      // 底层 gtimer 实例
    entriesMu sync.RWMutex       // 条目互斥锁
    entries   map[string]*entry  // name → entry 映射
}
```

### 12.5 关键方法

| 方法 | 说明 |
|------|------|
| `New()` | 创建 Cron 实例 |
| `Add(name, pattern, job)` | 添加无限循环任务 |
| `AddSingleton(name, pattern, job)` | 添加单例任务 |
| `AddOnce(name, pattern, job)` | 添加单次任务 |
| `AddRemove(name, pattern, job)` | 添加可移除任务 |
| `Remove(name)` | 移除任务 |
| `Size()` | 获取任务数量 |
| `Entries()` | 获取所有任务 |

> **注意**：`name` 参数为空时自动生成唯一名称。

### 12.6 使用示例

```go
package main

import (
    "context"
    "github.com/gogf/gf/v2/os/gcron"
    "time"
)

func main() {
    ctx := context.Background()

    // 每 5 分钟执行
    gcron.Add(ctx, "*/5 * * * *", func(ctx context.Context) {
        println("every 5 minutes")
    })

    // 每天凌晨执行（单例）
    gcron.AddSingleton(ctx, "0 2 * * *", func(ctx context.Context) {
        println("daily report at 2 AM")
    })

    // 单次执行
    gcron.AddOnce(ctx, "*/1 * * * *", func(ctx context.Context) {
        println("run once")
    })

    time.Sleep(10 * time.Minute)
}
```

### 12.7 模块间依赖

```
gcron → os/gtimer    (底层调度引擎)
      → text/gregex  (cron 表达式解析)
      → container/gtype (并发安全)
```

---

## 13. os/gfsnotify — 文件系统监听（fsnotify 封装）

### 13.1 包概述

`gfsnotify` 封装了第三方库 `fsnotify`，提供高性能的文件系统变更监听，支持**递归目录监控**、**重复事件过滤**、以及**回调链**。

源码位置：`os/gfsnotify/`

包注释见 `os/gfsnotify/gfsnotify.go:7`：
> Package gfsnotify provides a cross-platform file system notification mechanism.

### 13.2 核心设计

```
gfsnotify/
├── gfsnotify.go                — 公共 API + 常量定义
├── gfsnotify_watcher.go        — Watcher 核心结构
├── gfsnotify_watcher_loop.go   — 双协程循环（watchLoop + eventLoop）
├── gfsnotify_event.go          — Event 结构体
└── gfsnotify_z_unit_test.go
```

### 13.3 核心类型

#### Event 结构体（`os/gfsnotify/gfsnotify_event.go`）

```go
type Event struct {
    fsnotify.Event                       // 嵌入 fsnotify.Event
    op          Op                       // 操作类型
    Watcher     *Watcher                 // 关联的 Watcher
}
```

#### Op 操作类型

```go
const (
    Create Op = fsnotify.Create // 创建
    Write  Op = fsnotify.Write  // 写入
    Remove Op = fsnotify.Remove // 删除
    Rename Op = fsnotify.Rename // 重命名
    Chmod  Op = fsnotify.Chmod  // 权限变更
)
```

#### Watcher 结构体（`os/gfsnotify/gfsnotify_watcher.go`）

```go
type Watcher struct {
    watcher           *fsnotify.Watcher        // 底层 fsnotify 实例
    paths             map[string]bool          // 监控的路径集合
    callbacks         map[string]func(*Event)  // 路径 → 回调函数
    callbacksMu       sync.RWMutex             // 回调锁
    pathsMu           sync.RWMutex             // 路径锁
    closeOnce         sync.Once                // 关闭保护
}
```

### 13.4 双协程循环

`os/gfsnotify/gfsnotify_watcher_loop.go` 采用**双协程**模式：

```
┌─────────────┐         ┌─────────────┐
│  watchLoop  │ events  │  eventLoop  │ → callbacks[path](&Event)
│ (读取事件)   ├────────>│ (分发事件)   │
└─────────────┘         └─────────────┘
```

1. **watchLoop**：从 `fsnotify.Watcher.Events` channel 读取原始事件
2. **eventLoop**：接收 watchLoop 传递的事件，匹配路径并调用回调函数

分离的好处是即使回调函数执行缓慢，也不会阻塞 fsnotify 的事件缓冲 channel。

### 13.5 重复事件过滤

利用 `gcache` 缓存最近处理的（路径+操作）组合，在一定时间窗口内过滤重复事件：

```go
// 使用 gcache 缓存事件签名，避免短时间内重复触发
// key: "path:op", TTL: 短期（如 1 秒）
```

### 13.6 递归目录监控

当监控一个目录时，`gfsnotify` 会递归监控其所有子目录：

```go
func (w *Watcher) addWatchRecursive(path string) error {
    // 1. 添加当前目录
    // 2. 扫描子目录
    // 3. 递归添加所有子目录
    // 4. 新建子目录时通过事件自动添加监控
}
```

### 13.7 关键方法

| 方法 | 说明 |
|------|------|
| `New()` | 创建新的 Watcher |
| `Add(path, callback)` | 添加监控路径（递归） |
| `AddOnce(path, callback)` | 添加监控（确保只添加一次） |
| `Remove(path)` | 移除监控 |
| `Close()` | 关闭 Watcher |

### 13.8 使用示例

```go
package main

import (
    "github.com/gogf/gf/v2/os/gfsnotify"
    "time"
)

func main() {
    // 监控目录变化
    gfsnotify.Add("/tmp/config", func(event *gfsnotify.Event) {
        if event.IsCreate() {
            println("File created:", event.Path)
        }
        if event.IsWrite() {
            println("File modified:", event.Path)
        }
        if event.IsRemove() {
            println("File removed:", event.Path)
        }
    })

    time.Sleep(10 * time.Minute)
}
```

### 13.9 模块间依赖

```
gfsnotify → fsnotify.io/fsnotify (第三方库)
          → os/gcache           (重复事件过滤)
          → os/gfile            (递归目录扫描)
          → container/gmap      (路径和回调管理)
          → container/glist     (回调列表)
```

---

## 14. os/gmetric — 指标度量（OTel Metrics 抽象）

### 14.1 包概述

`gmetric` 是 GoFrame 对 OpenTelemetry Metrics API 的抽象层，定义了统一的指标度量接口。默认使用 noop performer（指标不导出），当注册了外部 Provider 后可对接 Prometheus、OTLP 等后端。

源码位置：`os/gmetric/`

包注释见 `os/gmetric/gmetric.go:7`：
> Package gmetric provides metric facilities.

### 14.2 核心设计

```
gmetric/
├── gmetric.go                  — 全局 Provider + 公共 API
├── gmetric_meter.go            — Meter 接口 + 实现
├── gmetric_meter_counter.go    — Counter 指标
├── gmetric_meter_updown.go     — UpDownCounter 指标
├── gmetric_meter_histogram.go  — Histogram 指标
├── gmetric_meter_observable_*.go — Observable 指标（Callback）
├── gmetric_attribute.go        — Attribute/Attributes
├── gmetric_instrument.go       — InstrumentKind/Option
├── gmetric_unit.go             — Unit 常量
└── gmetric_z_unit_test.go
```

**四层架构**：

```
┌──────────────────────────────────┐
│       Provider（指标提供者）       │
│   注册全局 Meter                  │
├──────────────────────────────────┤
│         Meter（计量器）            │
│   创建各类指标 Instrument         │
├──────────────────────────────────┤
│   Instrument（指标实例）           │
│   Counter/UpDown/Histogram/...   │
├──────────────────────────────────┤
│   Performer（执行器）              │
│   noop（默认） / OTel Performer   │
└──────────────────────────────────┘
```

### 14.3 核心类型和接口

#### Provider 接口

```go
type Provider interface {
    Meter(instrument InstrumentName, option ...Option) Meter
}
```

#### Meter 接口

```go
type Meter interface {
    // 指标创建
    Counter(name string, option ...Option) Counter
    UpDownCounter(name string, option ...Option) UpDownCounter
    Histogram(name string, option ...Option) Histogram
    ObservableCounter(name string, option ...Option) ObservableCounter
    ObservableUpDownCounter(name string, option ...Option) ObservableUpDownCounter
    ObservableGauge(name string, option ...Option) ObservableGauge

    // 回调注册
    RegisterCallback(function CallbackFunc, instruments ...Observable) (Callback, error)
}
```

#### MeterPerformer / Performer

```go
type MeterPerformer interface {
    CounterPerformer
    UpDownCounterPerformer
    HistogramPerformer
}

// Performer 是实际的指标操作执行者
// 默认为 noop（不导出）
// 注册 OTel Provider 后使用真正的 OTel performer
```

#### Attribute（`os/gmetric/gmetric_attribute.go`）

```go
type Attribute interface {
    Key() string
    Value() any
}

type Attributes []Attribute

func NewAttribute(key string, value any) Attribute
func CommonAttributes() Attributes // 返回内置通用属性（hostname, process path）
```

`CommonAttributes()` 在 `init()` 时自动收集：

```go
func init() {
    hostname, _ = os.Hostname()
    processPath = gfile.SelfPath()
}

func CommonAttributes() Attributes {
    return Attributes{
        NewAttribute(`os.host.name`, hostname),
        NewAttribute(`process.path`, processPath),
    }
}
```

#### InstrumentKind 枚举

```go
const (
    InstrumentKindCounter             InstrumentKind = 1 // 单调递增计数器
    InstrumentKindUpDownCounter       InstrumentKind = 2 // 可增减计数器
    InstrumentKindHistogram           InstrumentKind = 3 // 直方图
    InstrumentKindObservableCounter   InstrumentKind = 4 // 可观察计数器
    InstrumentKindObservableUpDownCounter InstrumentKind = 5 // 可观察可增减
    InstrumentKindObservableGauge     InstrumentKind = 6 // 可观察仪表盘
)
```

### 14.4 六种指标类型

| 类型 | 说明 | 方法 |
|------|------|------|
| Counter | 单调递增计数器 | `Inc()` / `Add(n)` |
| UpDownCounter | 可增减计数器 | `Add(n)` (n可为负) |
| Histogram | 直方图（统计分布） | `Record(n)` |
| ObservableCounter | 回调式单调计数器 | Callback 注册 |
| ObservableUpDownCounter | 回调式可增减计数器 | Callback 注册 |
| ObservableGauge | 回调式仪表盘 | Callback 注册 |

### 14.5 Noop Performer 模式

默认情况下（未注册 Provider），所有指标操作执行 noop（空操作）：

```go
// noop performer 实现 MeterPerformer 接口
// 所有方法为空实现，不影响应用性能
// 注册真实 Provider 后自动切换为有效实现
```

### 14.6 关键方法

| 方法 | 说明 |
|------|------|
| `SetProvider(provider)` | 设置全局 Provider |
| `GetMeter(name, opts...)` | 获取/创建 Meter |
| `NewCounter(name, opts...)` | 创建 Counter 指标 |
| `NewUpDownCounter(name, opts...)` | 创建 UpDownCounter |
| `NewHistogram(name, opts...)` | 创建 Histogram |
| `NewAttribute(key, value)` | 创建属性 |
| `CommonAttributes()` | 获取通用属性集 |

### 14.7 使用示例

```go
package main

import (
    "context"
    "github.com/gogf/gf/v2/os/gmetric"
)

var (
    meter = gmetric.GetMeter("myapp")
    requestCounter = meter.Counter(
        "http_requests_total",
        gmetric.WithDescription("Total HTTP requests"),
        gmetric.WithUnit(gmetric.UnitDimensionless),
    )
)

func handleRequest(ctx context.Context) {
    requestCounter.Inc(ctx,
        gmetric.NewAttribute("method", "GET"),
        gmetric.NewAttribute("status", 200),
    )
}

func main() {
    // 默认 noop（指标不导出）
    // 注册 OTel Provider 后自动收集
    // metric.SetProvider(otelmetric.NewMeterProvider())
}
```

### 14.8 模块间依赖

```
gmetric → go.opentelemetry/otel/metric (OTel Metrics API)
        → os/gfile                     (CommonAttributes)
```

---

## 15. os/gres — 资源嵌入（打包文件系统）

### 15.1 包概述

`gres` 提供资源文件嵌入和管理功能，允许将模板文件、配置文件、静态资源等打包到二进制可执行文件中，运行时通过虚拟文件系统访问。底层使用 `BTree` 进行高效文件索引。

源码位置：`os/gres/`

包注释见 `os/gres/gres.go:7`：
> Package gres makes it easy to pack files into a binary executable.

### 15.2 核心设计

```
gres/
├── gres.go               — 公共 API + 全局 Resource 实例
├── gres_resource.go      — Resource 核心实现（BTree 存储）
├── gres_file.go          — File 文件结构
├── gres_internal_pack.go — 打包工具（内部）
└── gres_z_unit_test.go
```

### 15.3 核心类型

#### Resource 结构体（`os/gres/gres_resource.go`）

```go
type Resource struct {
    mu       sync.RWMutex            // 读写锁
    tree     *gbtree.BTree           // BTree 文件索引（path → *File）
    manifest string                  // 清单文件内容
}
```

使用 BTree 是因为：
- 文件路径有序，适合范围查询
- 查找性能 O(log n)
- 支持前缀匹配（目录搜索）

#### File 结构体（`os/gres/gres_file.go`）

```go
type File struct {
    resource *Resource          // 所属 Resource
    file     *packFile          // 打包文件元数据
}

type packFile struct {
    Name     string             // 文件路径
    Content  []byte             // 文件内容
    IsDir    bool               // 是否为目录
}
```

### 15.4 打包格式

`gres` 使用自定义打包格式（类似 tar），通过 `gf pack` CLI 工具生成：

```
┌─────────────────────────────────────┐
│         Header / Manifest           │
│  文件名、大小、偏移量的索引表          │
├─────────────────────────────────────┤
│         File Data Section           │
│  按序号排列的文件内容                 │
└─────────────────────────────────────┘
```

### 15.5 关键方法

| 方法 | 说明 |
|------|------|
| `Add(path, content)` | 添加资源文件 |
| `Load(data)` | 加载打包数据（Go embed 或 init()） |
| `Get(path)` | 获取文件 |
| `GetWithIndex(path, indexFile)` | 获取文件（支持目录索引） |
| `Contains(path)` | 检查文件是否存在 |
| `IsDir(path)` | 是否为目录 |
| `ScanDir(path, pattern, recursive)` | 扫描目录 |
| `Export(src, dst)` | 导出资源到文件系统 |
| `Instance()` | 获取全局 Resource 实例 |

### 15.6 使用示例

```go
package main

import (
    "fmt"
    "github.com/gogf/gf/v2/os/gres"
    _ "myapp/packed" // 导入通过 gf pack 生成的打包文件
)

func main() {
    // 从嵌入资源读取配置
    content := gres.GetContent("config/config.yaml")
    fmt.Println(string(content))

    // 检查文件是否存在
    if gres.Contains("template/index.html") {
        // 从嵌入资源加载模板
    }

    // 导出资源到文件系统
    gres.Export("static", "/var/www/static")
}
```

**打包命令**：

```bash
# 使用 gf CLI 打包目录
gf pack ./template template/packed.go
# 生成 Go 源文件，通过 init() 自动注册到 gres
```

### 15.7 模块间依赖

```
gres → container/gbtree (BTree 文件索引)
      → os/gfile        (路径处理)
      → encoding/gjson  (清单解析)
```

---

## 16. os/gstructs — 结构体反射工具

### 16.1 包概述

`gstructs` 提供结构体反射的高级工具函数，用于字段枚举、标签解析、结构体键提取等。是 `gconv`（结构体转换）、`gvalid`（数据校验）、`gmeta`（元数据管理）等模块的基础设施。

源码位置：`os/gstructs/gstructs.go`

包注释见 `os/gstructs/gstructs.go:7`：
> Package gstructs provides functions for struct information inspecting.

### 16.2 核心类型

```go
// Type wraps reflect.Type with additional features.
type Type struct {
    reflect.Type
}

// Field wraps reflect.StructField with tag parsing.
type Field struct {
    reflect.StructField          // 嵌入标准库 StructField
    // 扩展方法...
}

// FieldsInput is the input parameter for Fields function.
type FieldsInput struct {
    Type    reflect.Type  // 结构体类型
    Option  FieldsOption  // 选项
}

// FieldsOption controls field enumeration behavior.
type FieldsOption struct {
    Recursive      bool     // 是否递归展开匿名（嵌入）字段
    TagFilter      []string // 只包含这些标签存在的字段
    NoTagFilter    []string // 排除包含这些标签的字段
    PriorityTag    []string // 字段名优先级标签（如 json > yaml > c）
}

// FieldMapInput is the input parameter for FieldMap function.
type FieldMapInput struct {
    Pointer         any      // 结构体指针
    PriorityTag     []string // 标签优先级
    RecursiveOption int      // 递归选项
}
```

### 16.3 关键方法

| 方法 | 说明 |
|------|------|
| `Fields(input)` | 枚举结构体所有字段（支持递归匿名字段） |
| `FieldMap(input)` | 将结构体转换为 map[string]any |
| `StructKey(structOrTypeName, tag)` | 提取结构体的键值（如 json tag） |
| `TagFields(structOrTypeName, tagName)` | 提取指定标签的字段列表 |
| `Type(structOrTypeName)` | 获取 Type 包装 |

### 16.4 递归字段展开

`Fields` 支持递归展开匿名（嵌入）字段：

```go
// 给定：
type Base struct {
    Id int `json:"id"`
}
type User struct {
    Base       // 匿名嵌入
    Name string `json:"name"`
}

// Fields(input) 返回:
// [{Name: "Id", Tag: json:"id"},
//  {Name: "Name", Tag: json:"name"}]
```

### 16.5 标签优先级

`PriorityTag` 控制字段名的优先级映射：

```go
// PriorityTag: ["json", "yaml", "c"]
// 对于字段 Username string `json:"user" yaml:"username" c:"u"`
// 最终字段名为 "user"（json 标签优先）
```

### 16.6 使用示例

```go
package main

import (
    "fmt"
    "reflect"
    "github.com/gogf/gf/v2/os/gstructs"
)

type User struct {
    Id       int    `json:"id" db:"id"`
    Username string `json:"username" db:"name"`
    Password string `json:"-" db:"-"`
}

func main() {
    fields, _ := gstructs.Fields(gstructs.FieldsInput{
        Type: reflect.TypeOf(User{}),
    })
    for _, field := range fields {
        fmt.Printf("Field: %s, JSON Tag: %s\n",
            field.Name,
            field.Tag.Get("json"),
        )
    }
}
```

### 16.7 模块间依赖

```
gstructs → reflect (标准库反射)
         → gerror  (错误处理)
```

> **注意**：`gstructs` 故意保持零框架内部依赖，是 `util/gconv`、`util/gvalid` 等包的底层依赖。

---

## 17. os/gtime — 时间扩展（Time 包装器）

### 17.1 包概述

`gtime` 对 Go 标准库 `time` 包进行扩展，提供更便捷的时间操作：时间字符串智能解析（支持多种格式自动识别）、时间格式化、时间计算、以及丰富的短常量。

源码位置：`os/gtime/`

包注释见 `os/gtime/gtime.go:7`：
> Package gtime provides functionality for time measuring and formatting.

### 17.2 核心设计

```
gtime/
├── gtime.go              — 公共 API + 常量
├── gtime_time.go         — Time 结构体核心实现
├── gtime_time_convert.go — Time 类型转换
├── gtime_time_format.go  — 时间格式化（Go/PHP 混合格式）
├── gtime_time_operation.go — 时间运算（Add/Sub）
├── gtime_gtime.go        — 全局函数（Now/Format 等）
├── gtime_z_unit_test.go
```

### 17.3 短常量

```go
const (
    D   = 24 * time.Hour          // 1 天
    H   = time.Hour               // 1 小时
    M   = time.Minute             // 1 分钟
    S   = time.Second             // 1 秒
    MS  = time.Millisecond        // 1 毫秒
    US  = time.Microsecond        // 1 微秒
    NS  = time.Nanosecond         // 1 纳秒
)
```

### 17.4 Time 结构体

```go
// Time is a wrapper for time.Time for additional features.
type Time struct {
    wrapper
}

// wrapper embeds time.Time
type wrapper struct {
    time.Time
}
```

`Time` 包装了 `time.Time`，额外支持：
- 灵活的字符串解析
- 混合格式化（同时支持 Go 和 PHP 格式化符号）
- JSON 序列化/反序列化
- 链式运算

### 17.5 智能时间解析

`NewFromStr` 使用正则表达式自动识别多种时间格式：

```go
func NewFromStr(str string) *Time
```

支持自动识别的格式包括：
- `2006-01-02 15:04:05`
- `2006-01-02T15:04:05Z` (ISO 8601)
- `2006/01/02 15:04:05`
- `2006-01-02`
- `15:04:05`
- Unix 时间戳（整数）

### 17.6 时间格式化

同时支持 **Go 格式**和 **PHP 格式**：

| Go 格式 | PHP 格式 | 含义 |
|---------|---------|------|
| `2006` | `Y` | 4 位年份 |
| `01` | `m` | 月份（01-12） |
| `02` | `d` | 日（01-31） |
| `15` | `H` | 时（00-23） |
| `04` | `i` | 分（00-59） |
| `05` | `s` | 秒（00-59） |

```go
func (t *Time) Format(layout string) string
// Format("Y-m-d H:i:s") → "2026-06-13 14:30:00"
// Format("2006-01-02 15:04:05") → "2026-06-13 14:30:00"
```

### 17.7 关键方法

**全局函数**：

| 方法 | 说明 |
|------|------|
| `Now()` | 当前时间（返回 `*Time`） |
| `New(param...)` | 从多种类型创建 Time |
| `NewFromTime(t)` | 从 time.Time 创建 |
| `NewFromStr(str)` | 从字符串自动解析创建 |
| `NewFromStrFormat(str, format)` | 从指定格式字符串创建 |
| `NewFromTimeStamp(ts)` | 从 Unix 时间戳创建 |
| `Timestamp()` | 当前 Unix 时间戳（秒） |
| `TimestampMilli()` | 当前 Unix 时间戳（毫秒） |
| `TimestampMicro()` | 当前 Unix 时间戳（微秒） |
| `TimestampNano()` | 当前 Unix 时间戳（纳秒） |
| `Date()` | 当前日期字符串 |
| `Datetime()` | 当前日期时间字符串 |

**Time 方法**：

| 方法 | 说明 |
|------|------|
| `Format(layout)` | 格式化（Go/PHP 格式） |
| `Layout(layout)` | 格式化（Go layout 格式） |
| `Add(d)` | 加时间间隔 |
| `AddDate(years, months, days)` | 加日期 |
| `Sub(t)` | 减去时间（返回 Duration） |
| `Unix()` | Unix 时间戳（秒） |
| `UnixNano()` | Unix 时间戳（纳秒） |
| `Year()` / `Month()` / `Day()` | 日期组件 |
| `Hour()` / `Minute()` / `Second()` | 时间组件 |
| `StartOfDay()` | 当天开始（00:00:00） |
| `EndOfDay()` | 当天结束（23:59:59） |
| `StartOfWeek()` | 本周开始 |
| `EndOfWeek()` | 本周结束 |
| `StartOfMonth()` | 本月开始 |
| `EndOfMonth()` | 本月结束 |
| `StartOfYear()` | 本年开始 |
| `EndOfYear()` | 本年结束 |
| `Equal(t)` | 是否相等 |
| `Before(t)` | 是否在 t 之前 |
| `After(t)` | 是否在 t 之后 |
| `ToLocation(loc)` | 时区转换 |

### 17.8 使用示例

```go
package main

import (
    "fmt"
    "github.com/gogf/gf/v2/os/gtime"
)

func main() {
    // 当前时间
    now := gtime.Now()
    fmt.Println(now.Format("Y-m-d H:i:s")) // 2026-06-13 14:30:00

    // 从字符串创建
    t := gtime.NewFromStr("2026-06-13 14:30:00")

    // 时间运算
    tomorrow := now.Add(gtime.D)       // 加一天
    nextWeek := now.AddDate(0, 0, 7)   // 加一周

    // 时间范围
    startOfDay := now.StartOfDay()     // 2026-06-13 00:00:00
    endOfMonth := now.EndOfMonth()     // 2026-06-30 23:59:59

    // Unix 时间戳
    fmt.Println(gtime.Timestamp())     // 1718272200
    fmt.Println(gtime.TimestampMilli()) // 1718272200000

    // 短常量
    duration := 3 * gtime.H + 30 * gtime.M // 3小时30分钟
    fmt.Println(duration) // 3h30m0s
}
```

### 17.9 模块间依赖

```
gtime → text/gregex (时间格式智能解析)
      → gerror      (错误处理)
```

---

## 模块间依赖总览

```
                        ┌─────────┐
                        │  gtime  │
                        └────┬────┘
                             │
            ┌────────────────┼────────────────┐
            │                │                │
        ┌───┴───┐       ┌───┴───┐       ┌───┴────┐
        │ gfile │       │ gproc │       │ gtimer │
        └───┬───┘       └───────┘       └───┬────┘
            │                                 │
   ┌────────┼────────┐                 ┌─────┴─────┐
   │        │        │                 │           │
┌──┴──┐ ┌──┴──┐ ┌──┴───┐          ┌───┴───┐  ┌────┴────┐
│gres │ │gctx │ │gfsnotify│       │ gcron │  │  glog   │
└──┬──┘ └─────┘ └──┬──────┘       └───────┘  └────┬────┘
   │               │                              │
   │               │  ┌───────────┐               │
   │               │  │  gcache   │               │
   │               │  └─────┬─────┘               │
   │               │        │                     │
┌──┴──────┐  ┌─────┴──┐  ┌──┴──────┐       ┌─────┴─────┐
│ gview   │  │ gi18n  │  │ gsession│       │  gmetric  │
└─────────┘  └────────┘  └─────────┘       └───────────┘

┌──────────────────────────────────────┐
│              gcfg                    │
│  (依赖 gjson, gfile, gres, gvar)     │
└──────────────────────────────────────┘

┌──────────────────────────────────────┐
│              gcmd                    │
│  (依赖 gctx, gerror, gconv)          │
└──────────────────────────────────────┘

┌──────────────────────────────────────┐
│            gstructs                  │
│  (零内部依赖，纯反射工具)              │
└──────────────────────────────────────┘
```

## 设计模式总结

| 模式 | 应用模块 | 说明 |
|------|---------|------|
| **适配器模式** | gcfg, gcache, gsession | 统一接口 + 可替换后端 |
| **处理器链** | glog | Handler 中间件链 |
| **单例模式** | gcfg, glog, gview, gi18n | 全局实例管理 |
| **观察者模式** | gfsnotify | 文件变更事件通知 |
| **策略模式** | gcache (LRU/TTL) | 可选淘汰策略 |
| **泛型** | gcfg (Loader[T]) | 类型安全配置绑定 |
| **位掩码** | glog (LEVEL_*) | 日志级别位运算 |
| **优先队列** | gtimer | 堆调度定时任务 |
| **装饰器模式** | gctx (NeverDone) | context 包装扩展 |
| **Noop 模式** | gmetric | 空操作降级策略 |

# 第一部分：Facade 门面层（frame/g、frame/gins、internal/instance）

## 目录

- [1. 包概述与设计理念](#1-包概述与设计理念)
- [2. 模块全景架构](#2-模块全景架构)
- [3. frame/g — 类型别名体系](#3-frameg--类型别名体系)
  - [3.1 基础类型别名](#31-基础类型别名)
  - [3.2 Map 类型别名矩阵](#32-map-类型别名矩阵)
  - [3.3 List 类型别名矩阵](#33-list-类型别名矩阵)
  - [3.4 Slice 类型别名](#34-slice-类型别名)
  - [3.5 Array 类型别名](#35-array-类型别名)
  - [3.6 别名设计权衡](#36-别名设计权衡)
- [4. frame/g — 函数与对象工厂](#4-frameg--函数与对象工厂)
  - [4.1 对象创建函数一览](#41-对象创建函数一览)
  - [4.2 工具函数](#42-工具函数)
- [5. internal/instance — 单例容器实现](#5-internalinstance--单例容器实现)
  - [5.1 分段锁架构](#51-分段锁架构)
  - [5.2 DJB 哈希算法](#52-djb-哈希算法)
  - [5.3 API 列表](#53-api-列表)
  - [5.4 GetOrSetFuncLock 的双重语义](#54-getorsetfunclock-的双重语义)
- [6. frame/gins — 核心组件单例管理](#6-framegins--核心组件单例管理)
  - [6.1 组件名常量](#61-组件名常量)
  - [6.2 Config — 配置管理单例](#62-config--配置管理单例)
  - [6.3 Resource — 资源管理单例](#63-resource--资源管理单例)
  - [6.4 I18n — 国际化单例](#64-i18n--国际化单例)
  - [6.5 Log — 日志单例](#65-log--日志单例)
  - [6.6 View — 模板引擎单例](#66-view--模板引擎单例)
  - [6.7 HttpClient — HTTP 客户端单例](#67-httpclient--http-客户端单例)
  - [6.8 Server — HTTP 服务器单例](#68-server--http-服务器单例)
  - [6.9 Database — 数据库 ORM 单例](#69-database--数据库-orm-单例)
  - [6.10 Redis — Redis 客户端单例](#610-redis--redis-客户端单例)
- [7. 调用链全景分析](#7-调用链全景分析)
  - [7.1 g.DB() 完整调用链](#71-gdb-完整调用链)
  - [7.2 g.Server() 完整调用链](#72-gserver-完整调用链)
  - [7.3 g.Log() 完整调用链](#73-glog-完整调用链)
  - [7.4 g.Cfg() 完整调用链](#74-gcfg-完整调用链)
- [8. 配置集成与三级查找策略](#8-配置集成与三级查找策略)
  - [8.1 三级配置查找策略](#81-三级配置查找策略)
  - [8.2 配置节点名常量](#82-配置节点名常量)
  - [8.3 大小写不敏感查找](#83-大小写不敏感查找)
- [9. 完整方法/类型别名速查表](#9-完整方法类型别名速查表)
- [10. 使用示例与最佳实践](#10-使用示例与最佳实践)
- [11. 与其他模块的依赖关系](#11-与其他模块的依赖关系)
- [12. 内部实现注意事项](#12-内部实现注意事项)

---

## 1. 包概述与设计理念

GoFrame 的 `frame/` 目录是整个框架的**门面层（Facade）**，由三个紧密协作的模块组成：

| 模块 | 路径 | 职责 |
| --- | --- | --- |
| `g` | `frame/g/` | 类型别名 + 对象创建快捷函数，用户直接交互的入口 |
| `gins` | `frame/gins/` | 核心组件的单例获取与管理，桥接 `g` 与底层模块 |
| `instance` | `internal/instance/` | 通用的单例容器，提供分段锁并发安全存储 |

**设计理念**：

1. **门面模式（Facade Pattern）**：`g` 包是一个极薄的封装层，不包含任何业务逻辑，仅做类型别名定义和函数委托。用户代码只需要 `import "github.com/gogf/gf/v2/frame/g"` 即可获得框架的全部常用能力。

2. **懒加载（Lazy Initialization）**：所有核心组件（数据库、Redis、日志、服务器等）在首次调用时才创建，不会在程序启动时初始化未使用的资源。

3. **配置驱动**：组件创建时自动从配置文件读取配置，支持三级查找策略和大小写不敏感匹配。

4. **分段锁优化**：`internal/instance` 使用 64 段分段锁，将不同 key 分散到不同的 `gmap.StrAnyMap` 中，降低锁竞争。

源码注释明确指出了使用 `g` 包的权衡（`frame/g/g.go:9`）：

> Note that, using package g might make the compiled binary a little bit bigger, as it imports a few frequently-used packages whatever you use them or not.

---

## 2. 模块全景架构

```
用户代码
  │
  ▼
┌─────────────────────────────────────────────┐
│  frame/g （门面层）                          │
│  ┌──────────────┐  ┌──────────────────────┐ │
│  │ g.go         │  │ g_object.go          │ │
│  │ (类型别名)   │  │ (Server/DB/Log/...)  │ │
│  └──────────────┘  └──────┬───────────────┘ │
│  ┌──────────────┐  ┌──────┴───────────────┐ │
│  │ g_func.go    │  │ g_setting.go         │ │
│  │ (Go/Try/Dump)│  │ (SetDebug)           │ │
│  └──────────────┘  └──────────────────────┘ │
└────────┬────────────────────────────────────┘
         │ 委托
         ▼
┌─────────────────────────────────────────────┐
│  frame/gins （单例管理层）                   │
│  ┌────────────┐ ┌─────────┐ ┌────────────┐  │
│  │gins_config │ │gins_db  │ │gins_server │  │
│  │gins_log    │ │gins_redis│ │gins_view  │  │
│  │gins_httpclient │gins_resource│gins_i18n│  │
│  └─────┬──────┘ └────┬────┘ └─────┬──────┘  │
│        │             │             │         │
│        └──────┬──────┴──────┬──────┘         │
│               ▼             │                │
│     internal/instance ──────┘                │
│     (64 段分段锁单例容器)                     │
└────────┬────────────────────────────────────┘
         │ 委托
         ▼
┌─────────────────────────────────────────────┐
│  底层模块                                     │
│  os/gcfg │ os/glog │ os/gview │ os/gres     │
│  database/gdb │ database/gredis             │
│  net/ghttp │ net/gclient │ net/gtcp/gudp    │
│  i18n/gi18n │ util/gvalid                   │
└─────────────────────────────────────────────┘
```

---

## 3. frame/g — 类型别名体系

源码位置：`frame/g/g.go`

### 3.1 基础类型别名

```go
type (
    Var  = gvar.Var        // 通用变量接口，类似泛型
    Ctx  = context.Context // context.Context 的简写
    Meta = gmeta.Meta      // 结构体元数据
)
```

| 别名 | 底层类型 | 用途 |
| --- | --- | --- |
| `g.Var` | `gvar.Var` | 万能动态变量，支持运行时类型转换 |
| `g.Ctx` | `context.Context` | 上下文，贯穿所有请求链路 |
| `g.Meta` | `gmeta.Meta` | 结构体元数据标签解析 |

### 3.2 Map 类型别名矩阵

`g` 包定义了 13 种 Map 组合别名，覆盖了所有常用的 key-value 类型对：

| 别名 | 底层类型 | 使用场景 |
| --- | --- | --- |
| `g.Map` | `map[string]any` | 最常用的通用 Map（API 参数、配置传递） |
| `g.MapAnyAny` | `map[any]any` | 全动态键值 |
| `g.MapAnyStr` | `map[any]string` | |
| `g.MapAnyInt` | `map[any]int` | |
| `g.MapStrAny` | `map[string]any` | 等同 `g.Map` |
| `g.MapStrStr` | `map[string]string` | 字符串字典 |
| `g.MapStrInt` | `map[string]int` | |
| `g.MapIntAny` | `map[int]any` | |
| `g.MapIntStr` | `map[int]string` | |
| `g.MapIntInt` | `map[int]int` | |
| `g.MapAnyBool` | `map[any]bool` | |
| `g.MapStrBool` | `map[string]bool` | 集合标记 |
| `g.MapIntBool` | `map[int]bool` | |

**命名规则**：`Map{KeyType}{ValueType}`，如 `MapStrAny` = `map[string]any`。

### 3.3 List 类型别名矩阵

`List` 是 `[]Map` 的别名，每个 List 类型对应一种 Map 类型：

| 别名 | 底层类型 |
| --- | --- |
| `g.List` | `[]Map` = `[]map[string]any` |
| `g.ListAnyAny` | `[]MapAnyAny` |
| `g.ListAnyStr` | `[]MapAnyStr` |
| `g.ListAnyInt` | `[]MapAnyInt` |
| `g.ListStrAny` | `[]MapStrAny` |
| `g.ListStrStr` | `[]MapStrStr` |
| `g.ListStrInt` | `[]MapStrInt` |
| `g.ListIntAny` | `[]MapIntAny` |
| `g.ListIntStr` | `[]MapIntStr` |
| `g.ListIntInt` | `[]MapIntInt` |
| `g.ListAnyBool` | `[]MapAnyBool` |
| `g.ListStrBool` | `[]MapStrBool` |
| `g.ListIntBool` | `[]MapIntBool` |

### 3.4 Slice 类型别名

| 别名 | 底层类型 |
| --- | --- |
| `g.Slice` | `[]any` |
| `g.SliceAny` | `[]any` |
| `g.SliceStr` | `[]string` |
| `g.SliceInt` | `[]int` |

### 3.5 Array 类型别名

`Array` 系列与 `Slice` 系列定义完全相同，仅名称不同，为开发者提供语义选择：

| 别名 | 底层类型 |
| --- | --- |
| `g.Array` | `[]any` |
| `g.ArrayAny` | `[]any` |
| `g.ArrayStr` | `[]string` |
| `g.ArrayInt` | `[]int` |

### 3.6 别名设计权衡

这些别名使用 Go 的 **type alias**（`=` 语法），而非 `type` 定义：

```go
// 类型别名（alias）— 与原类型完全等价，无需类型转换
type Map = map[string]any

// 类型定义（defined）— 与原类型不同，需要显式转换
type Map map[string]any  // g 包没有使用这种方式
```

**优势**：别名类型可以直接赋值给原始类型，无需 `Map(m)` 或 `m.(Map)` 这样的转换操作。

**代价**：`g` 包会引入所有别名涉及的包的编译依赖，即使开发者只使用其中一小部分。正如包注释所言，这会让编译产物略大。

---

## 4. frame/g — 函数与对象工厂

### 4.1 对象创建函数一览

源码位置：`frame/g/g_object.go`

所有对象创建函数都接受可选的 `name` 参数（变参），不传时使用默认实例名：

| 函数 | 签名 | 委托目标 | 返回类型 |
| --- | --- | --- | --- |
| `g.Server()` | `(name ...any) *ghttp.Server` | `gins.Server()` | HTTP 服务器 |
| `g.TCPServer()` | `(name ...any) *gtcp.Server` | `gtcp.GetServer()` | TCP 服务器 |
| `g.UDPServer()` | `(name ...any) *gudp.Server` | `gudp.GetServer()` | UDP 服务器 |
| `g.View()` | `(name ...string) *gview.View` | `gins.View()` | 模板引擎 |
| `g.Config()` | `(name ...string) *gcfg.Config` | `gins.Config()` | 配置管理 |
| `g.Cfg()` | `(name ...string) *gcfg.Config` | `gins.Config()` | 配置管理（别名） |
| `g.Resource()` | `(name ...string) *gres.Resource` | `gins.Resource()` | 资源管理 |
| `g.Res()` | `(name ...string) *gres.Resource` | `gins.Resource()` | 资源管理（别名） |
| `g.I18n()` | `(name ...string) *gi18n.Manager` | `gins.I18n()` | 国际化 |
| `g.Log()` | `(name ...string) *glog.Logger` | `gins.Log()` | 日志器 |
| `g.DB()` | `(name ...string) gdb.DB` | `gins.Database()` | 数据库 ORM |
| `g.Redis()` | `(name ...string) *gredis.Redis` | `gins.Redis()` | Redis 客户端 |
| `g.Client()` | `() *gclient.Client` | `gclient.New()` | HTTP 客户端（每次新建） |
| `g.Validator()` | `() *gvalid.Validator` | `gvalid.New()` | 校验器（每次新建） |
| `g.Model()` | `(tableOrStruct ...any) *gdb.Model` | `DB().Model()` | ORM 模型（基于默认 DB） |
| `g.ModelRaw()` | `(rawSql string, args ...any) *gdb.Model` | `DB().Raw()` | 原生 SQL 模型 |

> **注意**：`g.Client()` 和 `g.Validator()` 不走单例管理，每次调用都创建新实例。`g.Model()` 和 `g.ModelRaw()` 是便捷方法，内部调用 `g.DB()` 获取默认数据库实例。

### 4.2 工具函数

源码位置：`frame/g/g_func.go`、`frame/g/g_setting.go`

#### 异步与异常处理

| 函数 | 说明 |
| --- | --- |
| `g.Go(ctx, goroutineFunc, recoverFunc)` | 启动带 recover 的异步 goroutine |
| `g.Try(ctx, try)` | try 逻辑，返回 error |
| `g.TryCatch(ctx, try, catch)` | try-catch 逻辑 |
| `g.Throw(exception)` | 抛出异常（可被 TryCatch 捕获） |

#### 调试输出

| 函数 | 说明 |
| --- | --- |
| `g.Dump(values...)` | 美化输出到 stdout |
| `g.DumpTo(writer, value, option)` | 美化输出到指定 writer |
| `g.DumpWithType(values...)` | 带类型信息的美化输出 |
| `g.DumpWithOption(value, option)` | 带配置选项的美化输出 |
| `g.DumpJson(value)` | JSON 格式美化输出 |

#### 类型检查

| 函数 | 说明 |
| --- | --- |
| `g.IsNil(value, traceSource...)` | 检查是否为 nil，支持指针溯源 |
| `g.IsEmpty(value, traceSource...)` | 检查是否为空值（0/nil/false/""/空集合） |

#### 其他

| 函数 | 说明 |
| --- | --- |
| `g.NewVar(i, safe...)` | 创建 `gvar.Var` 实例 |
| `g.Wait()` | 阻塞等待所有 HTTP 服务器关闭（委托 `ghttp.Wait()`） |
| `g.Listen()` | 监听信号并执行注册的处理器（委托 `gproc.Listen()`） |
| `g.RequestFromCtx(ctx)` | 从 context 获取 `*ghttp.Request` |
| `g.SetDebug(enabled)` | 启用/禁用框架内部日志（非并发安全，应在启动阶段调用） |

---

## 5. internal/instance — 单例容器实现

源码位置：`internal/instance/instance.go`

包注释（`internal/instance/instance.go:7`）：

> Package instance provides instances management.
>
> Note that this package is not used for cache, as it has no cache expiration.

### 5.1 分段锁架构

`instance` 包的核心设计是 **64 段分段锁**：

```go
const (
    groupNumber = 64
)

var (
    groups = make([]*gmap.StrAnyMap, groupNumber)
)

func init() {
    for i := 0; i < groupNumber; i++ {
        groups[i] = gmap.NewStrAnyMap(true)  // true = concurrent-safe
    }
}
```

**工作原理**：

1. 启动时预创建 64 个并发安全的 `gmap.StrAnyMap`（分段）。
2. 根据 key 的 DJB 哈希值取模，将 key 路由到某一个分段。
3. 每个分段有独立的 `sync.RWMutex`，不同分段的读写操作互不阻塞。

**性能优势**：在高并发场景下，64 个分段将锁竞争分散，相比单一全局锁，理论上可支持 64 倍的并发写入吞吐量（实际取决于 key 分布的均匀程度）。

### 5.2 DJB 哈希算法

路由函数（`internal/instance/instance.go:31`）：

```go
func getGroup(key string) *gmap.StrAnyMap {
    return groups[int(ghash.DJB([]byte(key))%groupNumber)]
}
```

DJB 算法实现（`encoding/ghash/ghash_djb.go:10`）：

```go
func DJB(str []byte) uint32 {
    var hash uint32 = 5381
    for i := 0; i < len(str); i++ {
        hash += (hash << 5) + uint32(str[i])
    }
    return hash
}
```

DJB（Daniel J. Bernstein）哈希是一种快速且分布均匀的非加密哈希算法，初始值 5381，通过 `hash * 33 + c` 的位运算加速形式实现。

### 5.3 API 列表

| 函数 | 签名 | 说明 |
| --- | --- | --- |
| `Get` | `(name string) any` | 按 name 获取实例，不存在返回 nil |
| `Set` | `(name string, instance any)` | 设置实例（覆盖已有值） |
| `GetOrSet` | `(name string, instance any) any` | 获取或设置（传入值） |
| `GetOrSetFunc` | `(name string, f func() any) any` | 获取或设置（回调延迟计算） |
| `GetOrSetFuncLock` | `(name string, f func() any) any` | 获取或设置（回调在锁内执行） |
| `SetIfNotExist` | `(name string, instance any) bool` | 不存在时设置，返回是否成功 |
| `Clear` | `()` | 清除所有分段的全部实例 |

### 5.4 GetOrSetFuncLock 的双重语义

`GetOrSetFuncLock` 与 `GetOrSetFunc` 的关键区别（`internal/instance/instance.go:58`）：

```go
// GetOrSetFuncLock — 回调函数 f 在持有写锁的情况下执行
func GetOrSetFuncLock(name string, f func() any) any {
    return getGroup(name).GetOrSetFuncLock(name, f)
}
```

- **`GetOrSetFunc`**：先检查 key 是否存在（读锁），不存在时释放锁并执行回调 `f`，再尝试写入。**如果两个 goroutine 同时发现 key 不存在，都会执行回调**，但只有一个写入成功，另一个被丢弃。适用于回调无副作用的场景。
- **`GetOrSetFuncLock`**：在持有写锁的情况下执行回调 `f`。**保证回调全局只执行一次**。适用于回调有副作用（如创建数据库连接、打开文件）的场景。

`gins` 包中的所有组件创建函数（`Database`、`Redis`、`Server`、`Log`、`View`、`HttpClient`）都使用 `GetOrSetFuncLock`，确保重量级的初始化操作（读取配置、建立连接）只执行一次。

---

## 6. frame/gins — 核心组件单例管理

源码位置：`frame/gins/`

### 6.1 组件名常量

`gins` 包为每个核心组件定义了固定的实例 key 前缀（`frame/gins/gins.go:11`）：

```go
const (
    frameCoreComponentNameViewer     = "gf.core.component.viewer"
    frameCoreComponentNameDatabase   = "gf.core.component.database"
    frameCoreComponentNameHttpClient = "gf.core.component.httpclient"
    frameCoreComponentNameLogger     = "gf.core.component.logger"
    frameCoreComponentNameRedis      = "gf.core.component.redis"
    frameCoreComponentNameServer     = "gf.core.component.server"
)
```

实例 key 格式为 `{组件前缀}.{实例名}`，例如：
- `gf.core.component.database.default` — 默认数据库
- `gf.core.component.database.order` — 名为 "order" 的数据库
- `gf.core.component.server.default` — 默认 HTTP 服务器

### 6.2 Config — 配置管理单例

源码位置：`frame/gins/gins_config.go`

```go
func Config(name ...string) *gcfg.Config {
    return gcfg.Instance(name...)
}
```

`gins.Config()` 是最简单的组件函数——直接委托给 `gcfg.Instance()`，不经过 `internal/instance` 容器。`gcfg` 包内部维护了自己的实例映射（`gmap.NewKVMapWithChecker[string, *Config]`）。

**配置文件查找规则**（`os/gcfg/gcfg.go:56`）：
- 默认实例名：`"config"`（对应默认配置文件 `config.toml`）
- 如果传入自定义名称（如 `"test"`），则查找 `test.toml`
- 支持的文件格式：toml（默认）、yaml、yml、json、ini、xml

### 6.3 Resource — 资源管理单例

源码位置：`frame/gins/gins_resource.go`

```go
func Resource(name ...string) *gres.Resource {
    return gres.Instance(name...)
}
```

直接委托给 `gres.Instance()`，用于管理打包嵌入的资源文件。

### 6.4 I18n — 国际化单例

源码位置：`frame/gins/gins_i18n.go`

```go
func I18n(name ...string) *gi18n.Manager {
    return gi18n.Instance(name...)
}
```

直接委托给 `gi18n.Instance()`，管理多语言翻译。

### 6.5 Log — 日志单例

源码位置：`frame/gins/gins_log.go`

```go
func Log(name ...string) *glog.Logger
```

**调用链**：`gins.Log()` → `instance.GetOrSetFuncLock()` → 创建 `glog.Logger` → 从配置读取日志参数 → 应用配置。

**配置查找策略**（三级）：
1. 查找 `logger.{instanceName}` — 特定日志实例的配置
2. 查找 `logger` — 全局日志配置
3. 使用 `glog` 默认配置

如果配置节点名在大小写上有变体（如 `Logger`、`LOGGER`），通过 `gutil.MapPossibleItemByKey` 进行大小写不敏感匹配。

### 6.6 View — 模板引擎单例

源码位置：`frame/gins/gins_view.go`

```go
func View(name ...string) *gview.View
```

**调用链**：`gins.View()` → `instance.GetOrSetFuncLock()` → `getViewInstance()` → 创建 `gview.View` → 从配置读取模板参数 → 应用配置。

**配置查找策略**：
1. 查找 `viewer.{instanceName}` — 特定模板实例的配置
2. 查找 `viewer` — 全局模板配置
3. 使用 `gview` 默认配置

`Server` 组件创建时会隐式调用 `getViewInstance()`（`frame/gins/gins_server.go:97`），确保服务器可以使用模板渲染功能。

### 6.7 HttpClient — HTTP 客户端单例

源码位置：`frame/gins/gins_httpclient.go`

```go
func HttpClient(name ...any) *gclient.Client
```

这是最简单的经过 `instance` 容器管理的组件——直接创建新的 `gclient.Client`，无配置读取逻辑。

> **注意**：`g.Client()`（在 `g_object.go` 中）并不委托给 `gins.HttpClient()`，而是调用 `gclient.New()` 每次创建新实例。`gins.HttpClient()` 才是走单例管理的版本。

### 6.8 Server — HTTP 服务器单例

源码位置：`frame/gins/gins_server.go`

```go
func Server(name ...any) *ghttp.Server
```

这是最复杂的组件创建函数，涉及多级配置查找和关联组件初始化。

**初始化流程**：

1. 确定实例名（默认 `ghttp.DefaultServerName`）
2. `instance.GetOrSetFuncLock()` 尝试获取或创建
3. 获取 `ghttp.GetServer(instanceName)` 底层服务器
4. 如果配置可用（`Config().Available(ctx)`）：
   - 查找配置节点名：先找 `"server"`，找不到再找 `"httpserver"`（二级回退）
   - 查找 `server.{instanceName}` 服务器特定配置
  . 找不到则回退到全局 `server` 配置
   - 通过 `SetConfigWithMap()` 应用配置
   - 查找 `server.{instanceName}.logger` 日志配置并应用
5. 设置服务器名称（如果未配置）
6. 隐式初始化 View 组件（`getViewInstance()`）

### 6.9 Database — 数据库 ORM 单例

源码位置：`frame/gins/gins_database.go`

```go
func Database(name ...string) gdb.DB
```

**初始化流程**：

1. 确定组名（默认 `gdb.DefaultGroupName` = `"default"`）
2. `instance.GetOrSetFuncLock()` 尝试获取或创建
3. 从配置读取 `database` 节点（支持大小写不敏感匹配）
4. 解析配置为 `gdb.ConfigGroup` 结构：
   - 支持数组格式（多节点，用于负载均衡）
   - 支持 map 格式（单节点）
5. 注册到 `gdb` 全局配置（`gdb.SetConfigGroup()`）
6. 如果配置完全缺失且 `gdb` 也未通过 API 预配置，则 panic
7. 调用 `gdb.NewByGroup(name...)` 创建 ORM 实例
8. 从配置读取日志参数并应用到 ORM 的 Logger
9. 如果创建失败（如组名不存在），panic

**配置格式示例**：

```toml
# 多组配置（数组格式 = 负载均衡）
[database]
    [database.default]
        [[database.default.1]]
            link = "mysql:root:password@tcp(127.0.0.1:3306)/test"
        [[database.default.2]]
            link = "mysql:root:password@tcp(127.0.0.2:3306)/test"
    [database.order]
        [[database.order.1]]
            link = "mysql:root:password@tcp(127.0.0.1:3306)/order_db"
```

### 6.10 Redis — Redis 客户端单例

源码位置：`frame/gins/gins_redis.go`

```go
func Redis(name ...string) *gredis.Redis
```

**初始化流程**：

1. 确定组名（默认 `gredis.DefaultGroupName`）
2. `instance.GetOrSetFuncLock()` 尝试获取或创建
3. 先检查是否已通过 `gredis.SetConfig()` API 预配置
4. 如果未预配置且配置文件可用：
   - 从配置读取 `redis` 节点
   - 按组名查找特定配置
   - 调用 `gredis.ConfigFromMap()` 解析配置
   - 创建 `gredis.Redis` 实例
5. 如果完全无配置，panic

**配置格式示例**：

```toml
[redis]
    [redis.default]
        address = "127.0.0.1:6379"
        db       = 0
        pass     = "password"
    [redis.cache]
        address = "127.0.0.1:6380"
        db       = 1
```

---

## 7. 调用链全景分析

### 7.1 g.DB() 完整调用链

```
g.DB("order")
  │
  ├─ g_object.go:87 → gins.Database("order")
  │
  ├─ gins_database.go:27
  │    group = "order"
  │    instanceKey = "gf.core.component.database.order"
  │
  ├─ instance.GetOrSetFuncLock("gf.core.component.database.order", func() {
  │    │
  │    ├─ Config().Data(ctx) → 读取全部配置
  │    ├─ gutil.MapPossibleItemByKey(data, "database") → 大小写不敏感查找节点名
  │    ├─ Config().Get(ctx, "database") → 读取 database 配置段
  │    │
  │    ├─ 遍历配置 map，解析每个 group 的节点
  │    │    └─ parseDBConfigNode(v) → gconv.Struct() 转换为 gdb.ConfigNode
  │    │
  │    ├─ gdb.SetConfigGroup(group, cg) → 注册到 gdb 全局配置
  │    │
  │    ├─ gdb.NewByGroup("order")
  │    │    └─ newDBByConfigNode(node, "order")
  │    │         └─ driverMap[node.Type].New(core, node) → 调用具体驱动创建
  │    │
  │    └─ 读取 logger 配置 → logger.SetConfigWithMap()
  │  })
  │
  └─ return db.(gdb.DB)
```

### 7.2 g.Server() 完整调用链

```
g.Server("myapi")
  │
  ├─ g_object.go:31 → gins.Server("myapi")
  │
  ├─ gins_server.go:23
  │    instanceName = "myapi"
  │    instanceKey = "gf.core.component.server.[myapi]"
  │
  ├─ instance.GetOrSetFuncLock(instanceKey, func() {
  │    │
  │    ├─ ghttp.GetServer("myapi") → 从 ghttp 内部获取/创建服务器
  │    │
  │    ├─ Config().Available(ctx)? → 检查配置是否可用
  │    │    │
  │    │    ├─ 查找节点名：先 "server"，再 "httpserver"
  │    │    ├─ Config().MustGet("server.myapi") → 服务器特定配置
  │    │    ├─ Config().MustGet("server") → 全局配置（回退）
  │    │    └─ server.SetConfigWithMap(configMap)
  │    │
  │    ├─ 查找并应用日志配置
  │    │    └─ server.Logger().SetConfigWithMap()
  │    │
  │    ├─ server.SetName("myapi") → 确保名称已设置
  │    │
  │    └─ getViewInstance() → 隐式初始化模板引擎
  │  })
  │
  └─ return server.(*ghttp.Server)
```

### 7.3 g.Log() 完整调用链

```
g.Log("access")
  │
  ├─ g_object.go:81 → gins.Log("access")
  │
  ├─ gins_log.go:22
  │    instanceName = "access"
  │    instanceKey = "gf.core.component.logger.access"
  │
  ├─ instance.GetOrSetFuncLock(instanceKey, func() {
  │    │
  │    ├─ glog.Instance("access")
  │    │    └─ glog 内部 instances 容器（GetOrSetFuncLock）
  │    │
  │    ├─ 查找配置：logger.access → 特定日志配置
  │    ├─ 查找配置：logger → 全局日志配置（回退）
  │    │
  │    └─ logger.SetConfigWithMap(configMap)
  │  })
  │
  └─ return logger.(*glog.Logger)
```

> **注意**：这里存在**双层实例容器**——`gins` 通过 `internal/instance` 管理一份实例，`glog` 内部也维护了自己的实例映射。`gins.Log()` 的作用是在获取 `glog` 实例后额外读取配置文件并应用。

### 7.4 g.Cfg() 完整调用链

```
g.Cfg()
  │
  ├─ g_object.go:57 → gins.Config()  (Cfg 是 Config 的别名)
  │
  ├─ gins_config.go:15
  │    └─ gcfg.Instance("config")  (默认实例名)
  │
  ├─ gcfg.go:56
  │    └─ localInstances.GetOrSetFuncLock("config", func() {
  │         ├─ NewAdapterFile()
  │         └─ NewWithAdapter(adapterFile)
  │       })
  │
  └─ return *gcfg.Config
```

> `gcfg` 包使用自己的 `localInstances` 容器（`gmap.NewKVMapWithChecker[string, *Config]`），不经过 `internal/instance`。这是因为 `gcfg` 是所有其他组件的基础依赖，需要避免与 `instance` 包产生潜在的循环依赖。

---

## 8. 配置集成与三级查找策略

### 8.1 三级配置查找策略

GoFrame 的组件创建函数在读取配置时遵循统一的三级查找策略：

```
第一级：{节点名}.{实例名}    → 实例特定配置（最高优先级）
第二级：{节点名}             → 全局配置（回退）
第三级：组件默认配置          → 代码内置默认值（最终回退）
```

各组件的具体查找路径：

| 组件 | 第一级 | 第二级 | 默认行为 |
| --- | --- | --- | --- |
| Log | `logger.{name}` | `logger` | glog 内置默认 |
| View | `viewer.{name}` | `viewer` | gview 内置默认 |
| Server | `server.{name}` 或 `httpserver.{name}` | `server` 或 `httpserver` | ghttp 内置默认 |
| Database | `database.{group}` 内的节点数组 | `database` 全局 | panic（配置缺失） |
| Redis | `redis.{group}` | `redis` | panic（配置缺失） |

### 8.2 配置节点名常量

定义于 `internal/consts/consts.go:10`：

```go
const (
    ConfigNodeNameDatabase        = "database"
    ConfigNodeNameLogger          = "logger"
    ConfigNodeNameRedis           = "redis"
    ConfigNodeNameViewer          = "viewer"
    ConfigNodeNameServer          = "server"      // 通用版本配置项名
    ConfigNodeNameServerSecondary = "httpserver"  // v2 新增的配置项名
)
```

Server 组件支持两个节点名（`server` 和 `httpserver`），通过二级回退保证向后兼容。

### 8.3 大小写不敏感查找

所有组件在查找配置节点名时，都使用 `gutil.MapPossibleItemByKey()` 进行大小写不敏感匹配：

```go
// gins_database.go:45 示例
if configData, _ := Config().Data(ctx); len(configData) > 0 {
    if v, _ := gutil.MapPossibleItemByKey(configData, consts.ConfigNodeNameDatabase); v != "" {
        configNodeKey = v  // 使用实际找到的 key（可能是 "Database"、"DATABASE" 等）
    }
}
```

这意味着以下配置写法等价：

```toml
[database]    # 推荐（全小写）
[Database]    # 可用
[DATABASE]    # 可用
```

---

## 9. 完整方法/类型别名速查表

### 类型别名总表

```go
// 基础类型
Var  = gvar.Var
Ctx  = context.Context
Meta = gmeta.Meta

// Map 系列（13 种）
Map        = map[string]any
MapAnyAny  = map[any]any
MapAnyStr  = map[any]string
MapAnyInt  = map[any]int
MapStrAny  = map[string]any
MapStrStr  = map[string]string
MapStrInt  = map[string]int
MapIntAny  = map[int]any
MapIntStr  = map[int]string
MapIntInt  = map[int]int
MapAnyBool = map[any]bool
MapStrBool = map[string]bool
MapIntBool = map[int]bool

// List 系列（13 种）
List        = []Map
ListAnyAny  = []MapAnyAny
ListAnyStr  = []MapAnyStr
ListAnyInt  = []MapAnyInt
ListStrAny  = []MapStrAny
ListStrStr  = []MapStrStr
ListStrInt  = []MapStrInt
ListIntAny  = []MapIntAny
ListIntStr  = []MapIntStr
ListIntInt  = []MapIntInt
ListAnyBool = []MapAnyBool
ListStrBool = []MapStrBool
ListIntBool = []MapIntBool

// Slice 系列（4 种）
Slice    = []any
SliceAny = []any
SliceStr = []string
SliceInt = []int

// Array 系列（4 种）
Array    = []any
ArrayAny = []any
ArrayStr = []string
ArrayInt = []int
```

### 函数总表

```go
// 对象创建（单例）
Server(name ...any) *ghttp.Server
TCPServer(name ...any) *gtcp.Server
UDPServer(name ...any) *gudp.Server
View(name ...string) *gview.View
Config(name ...string) *gcfg.Config
Cfg(name ...string) *gcfg.Config           // Config 的别名
Resource(name ...string) *gres.Resource
Res(name ...string) *gres.Resource         // Resource 的别名
I18n(name ...string) *gi18n.Manager
Log(name ...string) *glog.Logger
DB(name ...string) gdb.DB
Redis(name ...string) *gredis.Redis

// 对象创建（非单例）
Client() *gclient.Client
Validator() *gvalid.Validator
Model(tableNameOrStruct ...any) *gdb.Model
ModelRaw(rawSql string, args ...any) *gdb.Model

// 异步与异常
Go(ctx, goroutineFunc, recoverFunc)
Try(ctx, try) error
TryCatch(ctx, try, catch)
Throw(exception)

// 调试输出
Dump(values ...any)
DumpTo(writer, value, option)
DumpWithType(values ...any)
DumpWithOption(value, option)
DumpJson(value)

// 类型检查
IsNil(value, traceSource ...) bool
IsEmpty(value, traceSource ...) bool

// 其他
NewVar(i any, safe ...bool) *Var
Wait()
Listen()
RequestFromCtx(ctx) *ghttp.Request
SetDebug(enabled bool)
```

---

## 10. 使用示例与最佳实践

### 10.1 Hello World

```go
package main

import (
    "github.com/gogf/gf/v2/frame/g"
    "github.com/gogf/gf/v2/net/ghttp"
)

func main() {
    s := g.Server()
    s.BindHandler("/", func(r *ghttp.Request) {
        r.Response.Write("hello world")
    })
    s.SetPort(8999)
    s.Run()
}
```

### 10.2 数据库操作

```go
package main

import (
    "github.com/gogf/gf/v2/frame/g"
    "github.com/gogf/gf/v2/os/gctx"
)

func main() {
    ctx := gctx.New()

    // 使用默认数据库组
    g.Model("users").Ctx(ctx).Insert(g.Map{
        "name":  "john",
        "email": "john@example.com",
    })

    // 使用指定数据库组
    g.DB("order").Model("orders").Ctx(ctx).Where("status", 1).All()

    // 使用便捷类型
    userList := g.List{}
    g.DB().Ctx(ctx).Raw("SELECT * FROM users").Scan(&userList)
}
```

### 10.3 配置读取

```go
package main

import (
    "fmt"
    "github.com/gogf/gf/v2/frame/g"
    "github.com/gogf/gf/v2/os/gctx"
)

func main() {
    ctx := gctx.New()

    // 通过 g.Cfg() 别名读取
    value := g.Cfg().MustGet(ctx, "app.name").String()
    fmt.Println(value)

    // 带默认值
    port := g.Cfg().MustGet(ctx, "server.port", 8080).Int()
    fmt.Println(port)
}
```

### 10.4 日志记录

```go
package main

import (
    "github.com/gogf/gf/v2/frame/g"
    "github.com/gogf/gf/v2/os/gctx"
)

func main() {
    ctx := gctx.New()

    // 默认日志器
    g.Log().Info(ctx, "application started")

    // 指定日志器（对应配置中的 logger.access）
    g.Log("access").Info(ctx, "access log")
}
```

### 10.5 Redis 操作

```go
package main

import (
    "github.com/gogf/gf/v2/frame/g"
    "github.com/gogf/gf/v2/os/gctx"
)

func main() {
    ctx := gctx.New()

    // 默认 Redis 组
    g.Redis().Set(ctx, "key", "value")

    // 指定 Redis 组
    g.Redis("cache").Set(ctx, "cache_key", "cache_value")
}
```

### 10.6 异步任务与异常处理

```go
package main

import (
    "github.com/gogf/gf/v2/frame/g"
    "github.com/gogf/gf/v2/os/gctx"
)

func main() {
    ctx := gctx.New()

    // 安全的异步任务
    g.Go(ctx, func(ctx context.Context) {
        // 业务逻辑
    }, func(ctx context.Context, err error) {
        // panic 恢复处理
        g.Log().Error(ctx, err)
    })

    // try-catch
    g.TryCatch(ctx,
        func(ctx context.Context) {
            g.Throw("something went wrong")
        },
        func(ctx context.Context, err error) {
            g.Log().Error(ctx, err)
        },
    )
}
```

### 10.7 最佳实践

1. **统一使用 `g.*` 入口**：在业务代码中，始终使用 `g.Server()`、`g.DB()`、`g.Log()` 等函数，而不是直接 import 底层包。这样代码更简洁，且自动获得配置集成。

2. **在框架内部代码中直接 import 底层包**：框架自身的实现代码不应依赖 `g` 包（它是用户层 Facade），而应直接 import `os/glog`、`database/gdb` 等。

3. **多实例场景传 name 参数**：当需要多个数据库连接或多个日志器时，通过 name 参数区分：
   ```go
   orderDB := g.DB("order")
   userDB  := g.DB("user")
   ```

4. **配置文件命名遵循约定**：默认配置文件名是 `config`（在 `manifest/config/` 目录下），支持 toml/yaml/json 等格式。

5. **避免在运行时调用 `SetDebug`**：`SetDebug` 不是并发安全的，应在 `main()` 的启动阶段调用。

6. **使用 `g.Go` 替代裸 `go`**：框架提供的 `g.Go` 自带 panic 恢复机制，比直接使用 Go 的 `go` 关键字更安全。

---

## 11. 与其他模块的依赖关系

### 11.1 依赖图

```
frame/g
  ├── container/gvar         (Var 类型)
  ├── util/gmeta             (Meta 类型)
  ├── util/gutil             (Go/Try/Dump 等函数)
  ├── internal/empty         (IsNil/IsEmpty)
  ├── internal/utils         (SetDebug)
  ├── database/gdb           (DB/Model)
  ├── database/gredis        (Redis)
  ├── frame/gins             (所有单例函数委托)
  ├── i18n/gi18n             (I18n)
  ├── net/gclient            (Client)
  ├── net/ghttp              (Server/Wait/RequestFromCtx)
  ├── net/gtcp               (TCPServer)
  ├── net/gudp               (UDPServer)
  ├── os/gcfg                (Config)
  ├── os/glog                (Log)
  ├── os/gproc               (Listen)
  ├── os/gres                (Resource)
  ├── os/gview               (View)
  └── util/gvalid            (Validator)

frame/gins
  ├── internal/instance      (所有需要单例容器的组件)
  ├── internal/consts        (配置节点名常量)
  ├── internal/intlog        (内部日志)
  ├── database/gdb           (Database)
  ├── database/gredis        (Redis)
  ├── errors/gcode           (错误码)
  ├── errors/gerror          (错误包装)
  ├── i18n/gi18n             (I18n)
  ├── net/gclient            (HttpClient)
  ├── net/ghttp              (Server)
  ├── os/gcfg                (Config)
  ├── os/glog                (Log)
  ├── os/gres                (Resource)
  ├── os/gview               (View)
  ├── util/gconv             (配置解析)
  └── util/gutil             (MapPossibleItemByKey)

internal/instance
  ├── container/gmap         (StrAnyMap 并发安全映射)
  └── encoding/ghash         (DJB 哈希)
```

### 11.2 依赖方向约束

- `frame/g` 可以依赖 `frame/gins`，但 `frame/gins` **不能**依赖 `frame/g`（避免循环导入）。
- `internal/instance` 是最底层的容器，只依赖 `container/gmap` 和 `encoding/ghash`，不依赖任何业务包。
- `os/gcfg` 不依赖 `internal/instance`，而是维护自己的实例映射，以避免与其他包形成循环依赖。

### 11.3 实例管理的分层结构

GoFrame 中存在三个层级的实例管理：

| 层级 | 容器 | 管理者 | 管理对象 |
| --- | --- | --- | --- |
| L1 | `internal/instance`（64 段分段锁） | `gins` 包 | Server、Database、Redis、Log、View、HttpClient |
| L2 | 各组件包内部的 instance 映射 | `gcfg`/`glog`/`gview`/`gres`/`gi18n`/`gdb`/`gredis` | 组件自身的单例 |
| L3 | `ghttp.GetServer()` / `gtcp.GetServer()` | 网络服务器包 | 服务器实例 |

当通过 `g.Log("access")` 创建日志器时，实际上经过了 L1 和 L2 两层缓存：
1. `gins` 先查 `internal/instance`（L1），找到则直接返回
2. 未找到则进入回调，调用 `glog.Instance("access")`（L2）
3. L2 内部也是 `GetOrSetFuncLock`，再次保证只创建一次

这种双层缓存设计确保了即使直接调用 `glog.Instance()`（不经过 `gins`）也能获得单例，同时 `gins` 在此基础上增加了配置集成能力。

---

## 12. 内部实现注意事项

### 12.1 panic 策略

`gins` 包中的组件创建函数在遇到不可恢复的错误时会 panic，而不是返回 error：

- 配置文件缺失（Database、Redis）
- 配置解析失败（所有组件）
- ORM 创建失败（Database）
- Redis 客户端创建失败（Redis）
- 服务器配置应用失败（Server）

这是设计决策而非缺陷——GoFrame 认为**核心组件初始化失败应该导致程序立即终止**，而不是在运行时以降级模式继续。

### 12.2 context 使用

`gins` 中的组件创建函数使用 `context.Background()` 而非传入的 context：

```go
ctx = context.Background()  // gins_database.go:29, gins_log.go:24, etc.
```

原因：组件创建通常发生在程序初始化阶段，此时还没有请求上下文。使用 `context.Background()` 确保不会被请求超时取消。

### 12.3 gins.Config 不经过 instance 容器

`gins.Config()` 直接委托给 `gcfg.Instance()`，不经过 `internal/instance`。这是因为 `gcfg` 是所有其他组件的基础依赖——Database、Redis、Server、Log、View 在初始化时都需要先获取 Config 实例。如果 Config 也存放在 `internal/instance` 中，虽然不会直接导致循环依赖，但增加了耦合度。

### 12.4 Server 的 instanceKey 格式差异

注意 Server 组件的 instanceKey 使用了 `%v` 格式化（`gins_server.go:28`）：

```go
instanceKey = fmt.Sprintf("%s.%v", frameCoreComponentNameServer, name)
```

当 `name` 为空变参切片 `[]any{}` 时，`%v` 输出为 `[]`，因此 instanceKey 变为 `"gf.core.component.server.[]"`。这不影响正确性（因为后续会用 `instanceName` 做实际查找），但与其他组件的 key 格式有细微差异。

### 12.5 Database 的配置双重注册保护

`gins.Database()` 在注册配置到 `gdb` 时有双重检查（`gins_database.go:92`）：

```go
if gcg, _ := gdb.GetConfigGroup(group); gcg == nil {
    // 配置组不存在 → 注册
    gdb.SetConfigGroup(g, cg)
} else {
    // 配置组已存在 → 忽略，仅打印内部日志
    intlog.Printf(ctx, "ignore configuration as it already exists for group: %s", g)
}
```

这确保即使用户在代码中通过 `gdb.SetConfigGroup()` 预配置了数据库，`gins.Database()` 也不会覆盖用户的配置。

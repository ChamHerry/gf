# GoFrame HTTP Server 与 HTTP Client 开发规范

> 本文档基于 GoFrame v2 源码（`net/ghttp`、`net/gclient`）深入分析 HTTP 服务端与客户端的架构设计、核心机制和完整 API。

---

## 目录

1. [模块概览与依赖关系](#1-模块概览与依赖关系)
2. [Server 生命周期](#2-server-生命周期)
3. [多层哈希路由树实现](#3-多层哈希路由树实现)
4. [洋葱模型中间件机制](#4-洋葱模型中间件机制)
5. [请求对象 Request](#5-请求对象-request)
6. [响应对象 Response](#6-响应对象-response)
7. [分组路由与规范化路由](#7-分组路由与规范化路由)
8. [Hook 机制](#8-hook-机制)
9. [WebSocket 支持](#9-websocket-支持)
10. [OpenAPI 自动生成](#10-openapi-自动生成)
11. [静态文件服务](#11-静态文件服务)
12. [HTTP Client](#12-http-client)
13. [Server 配置参考](#13-server-配置参考)
14. [模块间依赖关系图](#14-模块间依赖关系图)

---

## 1. 模块概览与依赖关系

### 1.1 包结构

| 包路径 | 用途 |
|--------|------|
| `net/ghttp` | HTTP Server（路由、中间件、请求/响应处理、静态文件、WebSocket、OpenAPI） |
| `net/gclient` | HTTP Client（链式调用、服务发现、中间件、重试、代理） |
| `net/goai` | OpenAPI v3 规范生成 |
| `net/gsvc` | 服务注册与发现抽象层 |
| `net/gsel` | 负载均衡选择器 |
| `net/gtrace` | OpenTelemetry 链路追踪集成 |

### 1.2 核心类型一览

```go
// net/ghttp/ghttp.go

type Server struct { ... }                    // HTTP 服务器
type Request struct { ... }                  // 请求上下文对象
type Response struct { ... }                 // 响应管理器
type RouterGroup struct { ... }              // 分组路由
type HandlerFunc = func(r *Request)          // 处理函数签名
type HookName string                         // Hook 名称枚举
type HandlerType string                      // 处理器类型枚举
```

```go
// net/gclient/gclient.go

type Client struct { ... }                   // HTTP 客户端
type Response struct { ... }                 // 客户端响应
type HandlerFunc = func(c *Client, r *http.Request) (*Response, error)  // 客户端中间件签名
```

---

## 2. Server 生命周期

### 2.1 生命周期流程

```
GetServer(name) → Server 实例（单例）
     │
     ▼
SetConfig / SetConfigWithMap / SetAddr / SetPort / EnableHTTPS ...
     │
     ▼
Group / BindHandler / BindObject / BindObjectRest / Use (中间件)
     │
     ▼
Start() ─── 非阻塞启动
  ├── handlePreBindItems()     处理预绑定路由
  ├── serverProcessInit()      进程级初始化（仅一次）
  ├── sessionManager 初始化
  ├── EnablePProf()            （可选）
  ├── startServer()            启动底层 HTTP/HTTPS 监听
  ├── initOpenApi()            生成 OpenAPI 规范
  ├── doServiceRegister()      服务注册
  └── doRouterMapDump()        打印路由表
     │
     ▼
Wait() ─── 阻塞等待所有 Server 退出
```

### 2.2 关键方法

| 方法 | 说明 |
|------|------|
| `GetServer(name ...any) *Server` | 获取/创建命名服务器实例（单例模式） |
| `Start() error` | 非阻塞启动，可启动多个 Server 后调用 `Wait()` |
| `Run()` | 阻塞启动单个 Server（内部调用 `Start` + 阻塞等待） |
| `Wait()` | 阻塞等待所有 Server 退出 |
| `Shutdown()` | 优雅关闭当前 Server |
| `Status() ServerStatus` | 获取服务器状态（`ServerStatusRunning` / `ServerStatusStopped`） |

### 2.3 基础示例

```go
package main

import (
    "github.com/gogf/gf/v2/frame/g"
    "github.com/gogf/gf/v2/net/ghttp"
)

func main() {
    s := g.Server()
    s.BindHandler("/", func(r *ghttp.Request) {
        r.Response.Write("Hello GoFrame")
    })
    s.SetPort(8199)
    s.Run()
}
```

### 2.4 多 Server 运行

```go
func main() {
    s1 := g.Server("api")
    s1.SetPort(8000)
    s1.BindHandler("/", handler1)

    s2 := g.Server("admin")
    s2.SetPort(8001)
    s2.BindHandler("/", handler2)

    s1.Start()
    s2.Start()

    ghttp.Wait() // 阻塞等待所有 Server
}
```

### 2.5 优雅重启

通过 `ServerConfig.Graceful` 配置启用，配合 `gf` CLI 工具实现不中断服务的热重启。底层使用 `internal/graceful` 包管理文件描述符传递。

---

## 3. 多层哈希路由树实现

### 3.1 路由树结构

GoFrame 的路由表使用**多层嵌套哈希表**实现，核心数据结构为 `Server.serveTree`：

```go
serveTree map[string]any
```

层级结构为：

```
serveTree
  ├── "default" (domain)          ← 第一层：域名
  │     ├── "api" (path node)     ← 第二层：URI 路径段
  │     │     ├── "v1"
  │     │     │     ├── "*list"   ← glist.List（存放匹配的 HandlerItem）
  │     │     │     └── "user"
  │     │     │           ├── "*fuzz"  ← 模糊匹配节点（:xxx / {xxx} / *xxx）
  │     │     │           │     └── "*list"
  │     │     │           └── "*list"
  │     │     └── "*list"
  │     └── "*list"
  └── "api.example.com" (domain)
        └── ...
```

### 3.2 路由注册过程 (`setHandler`)

路由注册核心流程在 `ghttp_server_router.go:setHandler` → `doSetHandler`：

1. **解析 pattern**：`parsePattern` 从路由规则中提取 `domain`、`method`、`uri`
   - 格式：`[METHOD]:[URI]@[DOMAIN]`
   - 示例：`GET:/api/user@api.example.com`

2. **构建哈希表路径**：按 `/` 分割 URI，逐层创建 `map[string]any` 节点
   - 模糊段（`:xxx`、`{xxx}`、`*xxx`）统一映射为 `*fuzz` 键
   - 每个模糊节点和叶子节点创建 `*list` 键（`glist.List`）

3. **按优先级插入**：`compareRouterPriority` 决定插入位置
   - 中间件优先级最高
   - URI 深度越深优先级越高
   - 规则类型：`{xxx}` > `:xxx` > `*xxx`

### 3.3 路由查找过程 (`searchHandlers`)

路由查找在 `ghttp_server_router_serve.go` 中，核心逻辑：

```
输入: method="GET", path="/api/v1/user/123", host="example.com"

1. 遍历域名列表: ["default", "example.com"]
2. 按 "/" 分割路径: ["api", "v1", "user", "123"]
3. 逐层遍历哈希表:
   - 精确匹配节点 → 继续深入
   - 无精确匹配 → 尝试 *fuzz 节点
   - 收集沿途所有 *list 到 lists 数组
4. 从 lists 尾部到头部遍历（尾部优先级最高）
5. 用正则 RegRule 匹配路径，提取路由参数
6. 按类型组装结果: middleware → handler → hook
```

### 3.4 路由缓存

查找结果通过 `gcache.Cache` 缓存（默认 TTL 1 小时），缓存键格式：

```
METHOD:/path@domain
```

### 3.5 支持的路由规则

| 规则 | 说明 | 示例 |
|------|------|------|
| 精确匹配 | 静态路径 | `/user/profile` |
| `:name` | 命名参数（匹配单个路径段） | `/user/:id` → `/user/123` |
| `{name}` | 字段参数（匹配单个路径段） | `/user/{id}` → `/user/123` |
| `*name` | 通配参数（匹配剩余所有路径） | `/file/*path` → `/file/a/b/c` |
| `{name}.ext` | 带扩展名的字段参数 | `/{hash}.{type}` → `/abc.png` |

### 3.6 支持的 HTTP 方法

```
GET, PUT, POST, DELETE, PATCH, HEAD, CONNECT, OPTIONS, TRACE
```

以及 `ALL`（匹配所有方法）。

---

## 4. 洋葱模型中间件机制

### 4.1 HandlerFunc 签名

```go
type HandlerFunc = func(r *Request)
```

中间件和处理函数共享同一个签名，通过 `r.Middleware.Next()` 控制调用链。

### 4.2 middleware 结构

```go
// ghttp_request_middleware.go
type middleware struct {
    served         bool     // 请求是否已被处理
    request        *Request
    handlerIndex   int      // 当前 handler 索引
    handlerMDIndex int      // 当前 handler 绑定的中间件索引
}
```

### 4.3 Next() 调用链

`middleware.Next()` 是核心流程控制函数：

```
handlers 链:
  [全局中间件1] → [全局中间件2] → [路由中间件] → [InitFunc] → [业务Handler] → [ShutFunc]

执行顺序（洋葱模型）:
  全局中间件1 前 → 全局中间件2 前 → 路由中间件 前 → InitFunc → Handler → ShutFunc
                                                                            ↑
  全局中间件1 后 ← 全局中间件2 后 ← 路由中间件 后 ← ────────────────────────────┘
```

处理逻辑：

1. 遍历 `request.handlers` 数组
2. 跳过 `HandlerTypeHook` 类型（由 ServeHTTP 单独调用）
3. 先执行绑定中间件 (`item.Handler.Middleware`)
4. 再按类型执行：
   - `HandlerTypeObject`：依次调用 `InitFunc` → 业务方法 → `ShutFunc`
   - `HandlerTypeHandler`：直接调用处理函数
   - `HandlerTypeMiddleware`：调用后停止循环（需内部调用 `Next` 继续）

### 4.4 中间件注册

**全局中间件**：

```go
s := g.Server()
s.Use(func(r *ghttp.Request) {
    // 前置处理
    r.Middleware.Next()
    // 后置处理
})
```

**分组路由中间件**：

```go
s.Group("/api", func(group *ghttp.RouterGroup) {
    group.Middleware(func(r *ghttp.Request) {
        // 前置
        r.Middleware.Next()
        // 后置
    })
    group.GET("/user", handler)
})
```

### 4.5 请求退出机制

| 方法 | 说明 |
|------|------|
| `r.Exit()` | 退出当前 handler（后续中间件后置代码仍执行） |
| `r.ExitAll()` | 退出当前及所有后续 handler |
| `r.ExitHook()` | 退出当前及后续 Hook handler |
| `r.IsExited() bool` | 检查请求是否已退出 |

底层通过 `panic(exceptionExit)` 实现，由 `niceCallFunc` 的 `recover` 捕获。

### 4.6 内置中间件

| 中间件 | 文件 | 说明 |
|--------|------|------|
| `internalMiddlewareServerTracing` | `ghttp_middleware_tracing.go` | OpenTelemetry 链路追踪（默认启用） |
| `MiddlewareCORS` | `ghttp_middleware_cors.go` | CORS 跨域处理 |
| `MiddlewareHandlerResponse` | `ghttp_middleware_handler_response.go` | 统一响应格式 |
| `MiddlewareJsonBody` | `ghttp_middleware_json_body.go` | JSON Body 解析校验 |
| `MiddlewareGzip` | `ghttp_middleware_gzip.go` | Gzip 压缩 |
| `MiddlewareNeverDoneCtx` | `ghttp_middleware_never_done_ctx.go` | 防止 context 被 cancel |

---

## 5. 请求对象 Request

### 5.1 结构定义

```go
type Request struct {
    *http.Request                     // 内嵌标准 http.Request
    Server     *Server                // 所属服务器
    Cookie     *Cookie                // Cookie 管理
    Session    *gsession.Session      // Session 管理
    Response   *Response              // 关联的响应对象
    Router     *Router                // 匹配到的路由信息
    EnterTime  *gtime.Time            // 请求进入时间
    LeaveTime  *gtime.Time            // 请求离开时间
    Middleware *middleware             // 中间件管理器
    StaticFile *staticFile            // 静态文件对象
}
```

### 5.2 参数获取优先级

`Get/GetRequest` 系列方法按以下优先级合并参数：

```
routerMap > paramsMap(自定义) > queryMap > formMap > bodyMap
```

即：**路由参数 > 自定义参数 > URL Query > 表单参数 > 请求体**

### 5.3 参数获取方法

#### 通用参数获取

| 方法 | 说明 |
|------|------|
| `Get(key string, def ...any) *gvar.Var` | 获取参数（别名 `GetRequest`） |
| `GetMap(def ...map[string]any) map[string]any` | 获取所有参数 map |
| `GetStruct(pointer any, mapping ...map[string]string) error` | 参数绑定到结构体 |
| `Parse(pointer any) error` | 参数绑定 + 自动校验（最常用） |

#### Query 参数

| 方法 | 说明 |
|------|------|
| `GetQuery(key string, def ...any) *gvar.Var` | 获取 URL Query 参数 |
| `GetQueryMap(def ...map[string]any) map[string]any` | 获取所有 Query 参数 |
| `ParseQuery(pointer any) error` | Query 参数绑定到结构体 |

#### 表单参数

| 方法 | 说明 |
|------|------|
| `GetForm(key string, def ...any) *gvar.Var` | 获取表单参数 |
| `GetFormMap(def ...map[string]any) map[string]any` | 获取所有表单参数 |
| `ParseForm(pointer any) error` | 表单参数绑定到结构体 |

#### 路由参数

| 方法 | 说明 |
|------|------|
| `GetRouter(key string, def ...any) *gvar.Var` | 获取路由匹配参数 |

#### 自定义参数

| 方法 | 说明 |
|------|------|
| `SetParam(key string, value any)` | 设置自定义参数 |
| `SetParamMap(data map[string]any)` | 批量设置自定义参数 |
| `GetParam(key string, def ...any) *gvar.Var` | 获取自定义参数 |

### 5.4 请求体处理

| 方法 | 说明 |
|------|------|
| `GetBody() []byte` | 获取请求体字节（可重复读取） |
| `GetBodyString() string` | 获取请求体字符串 |
| `GetJson() (*gjson.Json, error)` | 解析请求体为 JSON 对象 |
| `MakeBodyRepeatableRead(repeatableRead bool) []byte` | 标记请求体可重复读取 |

### 5.5 文件上传

| 方法 | 说明 |
|------|------|
| `GetFile(name string) *multipart.FileHeader` | 获取上传文件 |
| `GetFiles(name string) []*multipart.FileHeader` | 获取多个上传文件 |
| `GetMultipartForm() *multipart.Form` | 获取完整 multipart 表单 |
| `GetMultipartFiles(name string) []*multipart.FileHeader` | 获取 multipart 文件列表 |

### 5.6 请求信息

| 方法 | 说明 |
|------|------|
| `GetHost() string` | 获取主机名（不含端口） |
| `GetClientIp() string` | 获取客户端 IP（支持代理头解析） |
| `GetRemoteIp() string` | 获取 RemoteAddr IP |
| `GetSchema() string` | 获取协议（http/https） |
| `GetUrl() string` | 获取完整 URL |
| `GetReferer() string` | 获取 Referer |
| `GetHeader(key string, def ...string) string` | 获取请求头 |
| `GetSessionId() string` | 获取 Session ID |
| `IsAjaxRequest() bool` | 判断是否 AJAX 请求 |
| `IsFileRequest() bool` | 判断是否文件请求 |
| `GetError() error` | 获取请求处理中的错误 |
| `SetError(err error)` | 设置请求错误 |
| `ReloadParam()` | 清除解析标记，使参数重新解析 |

### 5.7 参数解析与校验示例

```go
type UserCreateReq struct {
    g.Meta `path:"/user" method:"post" summary:"创建用户" tags:"用户"`
    Name   string `v:"required|length:2,20" dc:"用户名"`
    Email  string `v:"required|email" dc:"邮箱"`
    Age    int    `v:"between:1,150" dc:"年龄"`
}

func (c *Controller) UserCreate(r *ghttp.Request) {
    var req *UserCreateReq
    if err := r.Parse(&req); err != nil {
        r.Response.WriteStatusExit(400, err.Error())
        return
    }
    r.Response.WriteJson(req)
}
```

`Parse` 内部流程：
1. 合并所有参数源到 `map[string]any`
2. `gconv.Struct` 转换到目标结构体
3. `gvalid.New()` 执行结构体校验

---

## 6. 响应对象 Response

### 6.1 结构定义

```go
type Response struct {
    *response.BufferWriter          // 带缓冲的 ResponseWriter
    Server                 *Server  // 所属服务器
    Request                *Request // 关联请求
}
```

`BufferWriter` 提供缓冲写入能力，在 `Flush()` 时一次性写入底层 `http.ResponseWriter`。

### 6.2 输出方法

#### 基础输出

| 方法 | 说明 |
|------|------|
| `Write(content ...any)` | 写入内容到缓冲区 |
| `WriteExit(content ...any)` | 写入并退出当前 handler |
| `WriteOver(content ...any)` | 清空缓冲区后写入 |
| `WriteOverExit(content ...any)` | 清空缓冲区写入并退出 |
| `Writef(format string, params ...any)` | 格式化写入 |
| `WritefExit(format string, params ...any)` | 格式化写入并退出 |
| `Writeln(content ...any)` | 写入并追加换行 |
| `Writefln(format string, params ...any)` | 格式化写入并换行 |

#### JSON/XML 输出

| 方法 | 说明 |
|------|------|
| `WriteJson(content any)` | 输出 JSON（自动设置 Content-Type） |
| `WriteJsonExit(content any)` | 输出 JSON 并退出 |
| `WriteJsonP(content any)` | 输出 JSONP（读取 `callback` 参数） |
| `WriteJsonPExit(content any)` | 输出 JSONP 并退出 |
| `WriteXml(content any, rootTag ...string)` | 输出 XML |
| `WriteXmlExit(content any, rootTag ...string)` | 输出 XML 并退出 |

#### HTTP 状态码

| 方法 | 说明 |
|------|------|
| `WriteStatus(status int, content ...any)` | 写入状态码和内容 |
| `WriteStatusExit(status int, content ...any)` | 写入状态码和内容并退出 |

#### 重定向

| 方法 | 说明 |
|------|------|
| `RedirectTo(location string, code ...int)` | 重定向到指定 URL（默认 302） |
| `RedirectBack(code ...int)` | 重定向到 Referer |

#### 文件服务

| 方法 | 说明 |
|------|------|
| `ServeFile(path string, allowIndex ...bool)` | 提供文件服务 |
| `ServeFileDownload(path string, name ...string)` | 提供文件下载（设置 Content-Disposition） |
| `ServeContent(name string, modTime time.Time, content io.ReadSeeker)` | 提供 io.ReadSeeker 内容服务 |

#### 其他

| 方法 | 说明 |
|------|------|
| `Flush()` | 将缓冲内容输出到客户端 |
| `ClearBuffer()` | 清空缓冲区 |

### 6.3 Cookie 操作（通过 Request.Cookie）

```go
// 设置
r.Cookie.Set("key", "value")

// 获取
value := r.Cookie.Get("key")

// 删除
r.Cookie.Remove("key")
```

---

## 7. 分组路由与规范化路由

### 7.1 RouterGroup

```go
type RouterGroup struct {
    parent     *RouterGroup       // 父组
    server     *Server            // 服务器
    domain     *Domain            // 绑定域名
    prefix     string             // 路由前缀
    middleware []HandlerFunc      // 中间件链
}
```

### 7.2 分组路由注册

```go
s := g.Server()

s.Group("/api/v1", func(group *ghttp.RouterGroup) {
    // 中间件
    group.Middleware(authMiddleware)

    // 方法级路由
    group.GET("/user", UserList)
    group.POST("/user", UserCreate)
    group.PUT("/user/:id", UserUpdate)
    group.DELETE("/user/:id", UserDelete)

    // RESTful 对象路由
    group.REST("/order", &OrderController{})

    // 嵌套分组
    group.Group("/admin", func(group *ghttp.RouterGroup) {
        group.Middleware(adminAuthMiddleware)
        group.GET("/dashboard", AdminDashboard)
    })

    // 批量绑定（函数/结构体对象）
    group.Bind(
        UserList,
        &ArticleController{},
    )
})
```

### 7.3 RESTful 路由自动映射

`REST` 方法注册对象后，按 RESTful 规范自动映射方法名到 HTTP 方法：

| 对象方法 | HTTP 方法 | 路径 |
|---------|-----------|------|
| `Get` / `GetList` | GET | /path |
| `Put` / `Update` | PUT | /path |
| `Post` / `Create` | POST | /path |
| `Delete` / `Remove` | DELETE | /path |
| `Patch` | PATCH | /path |

### 7.4 规范化路由（对象注册）

通过 `g.Meta` 标签声明路由元信息：

```go
type GetUserReq struct {
    g.Meta `path:"/user/{id}" method:"get" summary:"获取用户" tags:"用户" mime:"json"`
    Id     int `v:"required" dc:"用户ID"`
}

type GetUserRes struct {
    Id    int
    Name  string
    Email string
}

func (c *Controller) GetUser(ctx context.Context, req *GetUserReq) (res *GetUserRes, err error) {
    // 业务逻辑
    return &GetUserRes{Id: req.Id, Name: "test", Email: "test@example.com"}, nil
}

// 注册
s.Group("/", func(group *ghttp.RouterGroup) {
    group.Bind(
        &Controller{},
    )
})
```

### 7.5 URI 转换类型

通过 `ServerConfig.NameToUriType` 配置方法名到 URI 的转换规则：

| 常量 | 值 | 说明 | 示例 |
|------|-----|------|------|
| `UriTypeDefault` | 0 | 小写 + 连字符 | `GetUserList` → `/get-user-list` |
| `UriTypeFullName` | 1 | 保持原名 | `GetUserList` → `/GetUserList` |
| `UriTypeAllLower` | 2 | 全小写 | `GetUserList` → `/getuserlist` |
| `UriTypeCamel` | 3 | 驼峰 | `GetUserList` → `/getUserList` |

---

## 8. Hook 机制

### 8.1 Hook 类型

```go
const (
    HookBeforeServe HookName = "HOOK_BEFORE_SERVE"   // 路由处理前
    HookAfterServe  HookName = "HOOK_AFTER_SERVE"    // 路由处理后
    HookBeforeOutput HookName = "HOOK_BEFORE_OUTPUT"  // 响应输出前
    HookAfterOutput  HookName = "HOOK_AFTER_OUTPUT"   // 响应输出后
)
```

### 8.2 执行时序

```
ServeHTTP()
  │
  ├── searchStaticFile()         查找静态文件
  ├── getHandlersWithCache()     查找动态路由 handler
  │
  ├── callHookHandler(HookBeforeServe)
  │
  ├── 核心处理:
  │   ├── 静态文件服务 OR
  │   ├── request.Middleware.Next()  动态路由处理
  │   └── 404 处理
  │
  ├── callHookHandler(HookAfterServe)
  ├── callHookHandler(HookBeforeOutput)
  ├── handleResponse()           响应处理（状态码、Session、Cookie）
  └── callHookHandler(HookAfterOutput)
```

### 8.3 Hook 注册

```go
s.BindHookHandler("/*", ghttp.HookBeforeServe, func(r *ghttp.Request) {
    // 全局前置处理
})

s.Group("/api", func(group *ghttp.RouterGroup) {
    group.Hook("/user/*", ghttp.HookBeforeServe, authHook)
})
```

---

## 9. WebSocket 支持

### 9.1 当前状态

> **Deprecated**: `Request.WebSocket()` 和 `WebSocket` 类型将在未来版本移除。建议使用第三方 WebSocket 库（如 `gorilla/websocket`、`nhooyr.io/websocket`）。

### 9.2 底层实现

基于 `github.com/gorilla/websocket` 包，默认 Upgrader 配置：

```go
wsUpGrader = websocket.Upgrader{
    CheckOrigin: func(r *http.Request) bool {
        return true  // 默认不检查 Origin
    },
}
```

### 9.3 消息类型常量

| 常量 | 值 | 说明 |
|------|----|------|
| `WsMsgText` | 1 | 文本消息 |
| `WsMsgBinary` | 2 | 二进制消息 |
| `WsMsgClose` | 8 | 关闭消息 |
| `WsMsgPing` | 9 | Ping 消息 |
| `WsMsgPong` | 10 | Pong 消息 |

### 9.4 使用第三方库的推荐方式

```go
import "github.com/gorilla/websocket"

var upgrader = websocket.Upgrader{
    CheckOrigin: func(r *http.Request) bool { return true },
}

func WebSocketHandler(r *ghttp.Request) {
    conn, err := upgrader.Upgrade(r.Response.Writer, r.Request, nil)
    if err != nil {
        return
    }
    defer conn.Close()
    for {
        mt, msg, err := conn.ReadMessage()
        if err != nil {
            break
        }
        conn.WriteMessage(mt, msg)
    }
}
```

---

## 10. OpenAPI 自动生成

### 10.1 配置

```go
s := g.Server()
s.SetOpenApiPath("/api.json")     // OpenAPI 规范路径
s.SetSwaggerPath("/swagger")      // Swagger UI 路径
```

或通过配置文件：

```yaml
server:
  openapiPath: "/api.json"
  swaggerPath: "/swagger"
```

### 10.2 生成过程

在 `Server.Start()` 中调用 `initOpenApi()`：

1. 遍历所有已注册路由（`GetRoutes()`）
2. 跳过 `HandlerTypeMiddleware` 和 `HandlerTypeHook`
3. 对 `IsStrictRoute` 的路由（规范化路由），调用 `openapi.Add()` 自动生成
4. 从 Handler 函数签名的请求/响应结构体提取：
   - 路径、方法、标签、摘要（从 `g.Meta` 标签）
   - 请求参数（从结构体字段标签）
   - 响应格式

### 10.3 规范化路由的 OpenAPI 标签

```go
type CreateReq struct {
    g.Meta `path:"/user" method:"post" summary:"创建用户" tags:"用户管理" mime:"json"`
    Name   string `v:"required" dc:"用户名" in:"formData"`
    Age    int    `dc:"年龄" d:"18"`
}
```

支持的 `g.Meta` 标签字段：

| 标签 | 说明 |
|------|------|
| `path` | 路由路径 |
| `method` | HTTP 方法（支持逗号分隔多方法） |
| `domain` | 绑定域名 |
| `summary` | 接口摘要 |
| `tags` | API 分组标签 |
| `mime` | Content-Type |

### 10.4 访问

- OpenAPI JSON：`http://host:port/api.json`
- Swagger UI：`http://host:port/swagger/`

---

## 11. 静态文件服务

### 11.1 优先级规则

```
静态文件 > 动态路由 > 静态目录
```

如果请求同时匹配静态文件和动态路由：
- 如果是文件（非目录），优先返回静态文件
- 如果是目录且有动态路由匹配，优先使用动态路由

### 11.2 配置方法

```go
// 设置静态文件根目录
s.SetServerRoot("/var/www/public")

// 添加 URI 到目录的映射
s.AddStaticPath("/uploads", "/data/uploads")

// 添加搜索路径
s.AddSearchPath("/data/assets")
```

### 11.3 配置项

| 配置 | 默认值 | 说明 |
|------|--------|------|
| `ServerRoot` | `""` | 静态文件根目录 |
| `IndexFiles` | `["index.html", "index.htm"]` | 目录索引文件 |
| `IndexFolder` | `false` | 是否允许目录列表 |
| `SearchPaths` | `[]` | 额外搜索路径 |
| `Rewrites` | `{}` | URI 重写规则 |

### 11.4 静态文件搜索顺序

1. `StaticPaths` 映射（精确前缀匹配）
2. `SearchPaths` 目录搜索
3. 资源管理器（`gres` 内嵌资源）

支持从 `gres` 资源管理器中提供内嵌静态文件。

---

## 12. HTTP Client

### 12.1 Client 结构

```go
type Client struct {
    http.Client                         // 内嵌标准 http.Client
    header            map[string]string // 自定义请求头
    cookies           map[string]string // 自定义 Cookie
    prefix            string            // URL 前缀
    authUser          string            // Basic Auth 用户名
    authPass          string            // Basic Auth 密码
    retryCount        int               // 重试次数
    noUrlEncode       bool              // 是否禁用 URL 编码
    retryInterval     time.Duration     // 重试间隔
    middlewareHandler []HandlerFunc     // 中间件处理器
    discovery         gsvc.Discovery    // 服务发现
    builder           gsel.Builder      // 负载均衡选择器
}
```

### 12.2 创建与配置

```go
// 创建（默认启用 OpenTelemetry 和服务发现中间件）
client := gclient.New()

// 链式配置（每次返回新 Client）
client = client.Timeout(10 * time.Second).
    BasicAuth("user", "pass").
    Header(map[string]string{"X-Custom": "value"}).
    Cookie(map[string]string{"session": "abc"}).
    ContentJson().
    Prefix("http://api.example.com/v1")

// 浏览器模式（自动管理 Cookie）
client.SetBrowserMode(true)

// 代理设置（支持 HTTP 和 SOCKS5）
client.SetProxy("http://user:pass@proxy:8080")
client.SetProxy("socks5://user:pass@proxy:1080")

// 重试
client.SetRetry(3, 1*time.Second)
```

### 12.3 请求方法

| 方法 | HTTP 方法 | 返回 |
|------|-----------|------|
| `Get(ctx, url, data...)` | GET | `(*Response, error)` |
| `Put(ctx, url, data...)` | PUT | `(*Response, error)` |
| `Post(ctx, url, data...)` | POST | `(*Response, error)` |
| `Delete(ctx, url, data...)` | DELETE | `(*Response, error)` |
| `Head(ctx, url, data...)` | HEAD | `(*Response, error)` |
| `Patch(ctx, url, data...)` | PATCH | `(*Response, error)` |
| `Connect(ctx, url, data...)` | CONNECT | `(*Response, error)` |
| `Options(ctx, url, data...)` | OPTIONS | `(*Response, error)` |
| `Trace(ctx, url, data...)` | TRACE | `(*Response, error)` |
| `DoRequest(ctx, method, url, data...)` | 自定义 | `(*Response, error)` |
| `PostForm(ctx, url, data)` | POST (multipart) | `(*Response, error)` |

### 12.4 便捷内容方法（自动关闭响应）

| 方法 | 说明 |
|------|------|
| `GetContent(ctx, url, data...) string` | GET 请求返回字符串内容 |
| `PostContent(ctx, url, data...) string` | POST 请求返回字符串内容 |
| `PutContent(ctx, url, data...) string` | PUT 请求返回字符串内容 |
| `DeleteContent(ctx, url, data...) string` | DELETE 请求返回字符串内容 |
| `RequestContent(ctx, method, url, data...) string` | 自定义方法返回字符串内容 |

### 12.5 链式配置方法

| 方法 | 说明 |
|------|------|
| `Prefix(prefix string) *Client` | 设置 URL 前缀 |
| `Header(m map[string]string) *Client` | 设置请求头 |
| `HeaderRaw(headers string) *Client` | 原始格式设置请求头 |
| `Cookie(m map[string]string) *Client` | 设置 Cookie |
| `ContentType(contentType string) *Client` | 设置 Content-Type |
| `ContentJson() *Client` | 设置 JSON Content-Type |
| `ContentXml() *Client` | 设置 XML Content-Type |
| `Timeout(t time.Duration) *Client` | 设置超时 |
| `BasicAuth(user, pass string) *Client` | 设置 Basic Auth |
| `Retry(count int, interval time.Duration) *Client` | 设置重试 |
| `Proxy(proxyURL string) *Client` | 设置代理 |
| `RedirectLimit(limit int) *Client` | 设置重定向限制 |
| `NoUrlEncode() *Client` | 禁用 URL 编码 |
| `Discovery(discovery gsvc.Discovery) *Client` | 设置服务发现 |

### 12.6 Response 方法

| 方法 | 说明 |
|------|------|
| `ReadAll() []byte` | 读取完整响应体 |
| `ReadAllString() string` | 读取完整响应体为字符串 |
| `GetCookie(key string) string` | 获取响应 Cookie |
| `GetCookieMap() map[string]string` | 获取所有响应 Cookie |
| `SetBodyContent(content []byte)` | 覆盖响应体内容 |
| `Close() error` | 关闭响应（**必须调用**） |

### 12.7 客户端中间件

```go
client := gclient.New()
client.Use(func(c *gclient.Client, r *http.Request) (*gclient.Response, error) {
    // 前置处理：添加请求头
    r.Header.Set("X-Request-Id", uuid.New().String())

    // 调用下一层
    resp, err := c.Next(r)

    // 后置处理
    return resp, err
})
```

客户端中间件签名：`func(c *Client, r *http.Request) (*Response, error)`

### 12.8 数据发送格式自动检测

`DoRequest` 的参数处理逻辑：

1. 如果设置了 `Content-Type: application/json`，自动 JSON 序列化
2. 如果设置了 `Content-Type: application/xml`，自动 XML 序列化
3. 默认情况下：
   - 参数含 `@file:` 前缀 → 自动 `multipart/form-data` 上传
   - 参数以 `{` 或 `[` 开头 → 自动设为 `application/json`
   - 其他 → `application/x-www-form-urlencoded`

### 12.9 请求示例

```go
// JSON POST
resp, err := gclient.New().
    ContentJson().
    Post(ctx, "http://localhost:8000/api/user", g.Map{
        "name":  "john",
        "email": "john@example.com",
    })
if err != nil {
    panic(err)
}
defer resp.Close()
fmt.Println(resp.ReadAllString())

// 文件上传
resp, err := gclient.New().Post(ctx, "http://localhost:8000/upload", g.Map{
    "file":  "@file:/path/to/file.txt",
    "title": "my file",
})

// 服务发现 + 负载均衡
client := gclient.New().Discovery(etcdDiscovery)
resp, err := client.Get(ctx, "http://user-service/api/users")
```

---

## 13. Server 配置参考

### 13.1 ServerConfig 完整字段

```go
type ServerConfig struct {
    // ===== 基础配置 =====
    Name                    string          // 服务名（用于服务注册）
    Address                 string          // 监听地址（默认 ":0"）
    HTTPSAddr               string          // HTTPS 监听地址
    Listeners               []net.Listener  // 自定义 Listener
    Endpoints               []string        // 自定义服务注册端点
    HTTPSCertPath           string          // HTTPS 证书路径
    HTTPSKeyPath            string          // HTTPS 密钥路径
    TLSConfig               *tls.Config     // 自定义 TLS 配置
    Handler                 func(w http.ResponseWriter, r *http.Request)
    ReadTimeout             time.Duration   // 读超时（默认 60s）
    WriteTimeout            time.Duration   // 写超时（默认无限制）
    IdleTimeout             time.Duration   // 空闲超时（默认 60s）
    MaxHeaderBytes          int             // 最大请求头大小（默认 10KB）
    KeepAlive               bool            // HTTP Keep-Alive（默认 true）
    ServerAgent             string          // Server 头（默认 "GoFrame HTTP Server"）
    View                    *gview.View     // 默认模板引擎

    // ===== 静态文件 =====
    Rewrites                map[string]string       // URI 重写规则
    IndexFiles              []string                // 索引文件（默认 ["index.html","index.htm"]）
    IndexFolder             bool                    // 允许目录列表（默认 false）
    ServerRoot              string                  // 静态文件根目录
    SearchPaths             []string                // 额外搜索路径
    StaticPaths             []staticPathItem        // URI→目录映射
    FileServerEnabled       bool                    // 静态服务开关

    // ===== Cookie =====
    CookieMaxAge            time.Duration   // Cookie TTL（默认 365 天）
    CookiePath              string          // Cookie 路径（默认 "/"）
    CookieDomain            string          // Cookie 域名
    CookieSameSite          string          // SameSite 属性
    CookieSecure            bool            // Secure 属性
    CookieHttpOnly          bool            // HttpOnly 属性

    // ===== Session =====
    SessionIdName           string              // Session ID 名（默认 "gfsessionid"）
    SessionMaxAge           time.Duration       // Session TTL（默认 24h）
    SessionPath             string              // Session 文件存储路径
    SessionStorage          gsession.Storage    // 自定义 Session 存储
    SessionCookieMaxAge     time.Duration       // Session Cookie TTL
    SessionCookieOutput     bool                // 自动输出 Session Cookie（默认 true）

    // ===== 日志 =====
    Logger                  *glog.Logger    // 日志 Logger
    LogPath                 string          // 日志目录
    LogLevel                string          // 日志级别（默认 "all"）
    LogStdout               bool            // 输出到标准输出（默认 true）
    ErrorStack              bool            // 错误堆栈（默认 true）
    ErrorLogEnabled         bool            // 错误日志开关（默认 true）
    ErrorLogPattern         string          // 错误日志文件名模式（默认 "error-{Ymd}.log"）
    AccessLogEnabled        bool            // 访问日志开关（默认 false）
    AccessLogPattern        string          // 访问日志文件名模式（默认 "access-{Ymd}.log"）

    // ===== PProf =====
    PProfEnabled            bool            // PProf 开关
    PProfPattern            string          // PProf 路径

    // ===== API & Swagger =====
    OpenApiPath             string          // OpenAPI 规范路径
    SwaggerPath             string          // Swagger UI 路径
    SwaggerUITemplate       string          // 自定义 Swagger UI 模板

    // ===== 优雅重启 =====
    Graceful                bool            // 启用优雅重启
    GracefulTimeout         int             // 父进程最大存活时间（秒，默认 2）
    GracefulShutdownTimeout int             // 关闭超时（秒，默认 5）

    // ===== 其他 =====
    ClientMaxBodySize       int64           // 请求体大小限制（默认 8MB）
    FormParsingMemory       int64           // 表单解析内存限制（默认 1MB）
    NameToUriType           int             // 方法名→URI 转换类型
    RouteOverWrite          bool            // 允许路由覆盖
    DumpRouterMap           bool            // 启动时打印路由表（默认 true）
}
```

### 13.2 配置方法

```go
// 代码配置
s.SetConfig(config)
s.SetConfigWithMap(map[string]any{...})
s.SetAddr(":8080")
s.SetPort(8080, 8081)
s.SetHTTPSAddr(":443")
s.SetHTTPSPort(443)
s.EnableHTTPS(certFile, keyFile)
s.SetReadTimeout(30 * time.Second)
s.SetWriteTimeout(30 * time.Second)
s.SetKeepAlive(true)
s.SetServerAgent("MyServer/1.0")
s.SetName("my-api")

// YAML 配置
```

```yaml
server:
  address: ":8000"
  openapiPath: "/api.json"
  swaggerPath: "/swagger"
  serverRoot: "public"
  accessLogEnabled: true
  errorStack: true
  session:
    maxAge: "24h"
  cookie:
    maxAge: "365d"
```

---

## 14. 模块间依赖关系图

```
┌─────────────────────────────────────────────────────┐
│                    frame/g                           │
│          g.Server() / g.Client() 门面入口             │
└──────────────┬──────────────────────┬───────────────┘
               │                      │
               ▼                      ▼
┌──────────────────────┐  ┌───────────────────────┐
│    net/ghttp         │  │    net/gclient         │
│  ┌──────────────┐    │  │  ┌─────────────────┐  │
│  │ Server       │    │  │  │ Client          │  │
│  │  - serveTree │    │  │  │  - http.Client  │  │
│  │  - routesMap │    │  │  │  - middleware    │  │
│  │  - openapi   │    │  │  │  - discovery     │  │
│  └──────────────┘    │  │  └─────────────────┘  │
│  ┌──────────────┐    │  └───────────┬───────────┘
│  │ Request      │    │              │
│  │  - params    │    │              ▼
│  │  - query/form│    │  ┌───────────────────────┐
│  └──────────────┘    │  │ net/gsvc              │
│  ┌──────────────┐    │  │ 服务注册/发现抽象       │
│  │ Response     │    │  └───────────┬───────────┘
│  │  - BufferWr  │    │              │
│  └──────────────┘    │              ▼
│  ┌──────────────┐    │  ┌───────────────────────┐
│  │ RouterGroup  │    │  │ net/gsel              │
│  └──────────────┘    │  │ 负载均衡选择器          │
└──────────┬───────────┘  └───────────────────────┘
           │
           ▼
┌──────────────────────┐
│ net/goai             │
│ OpenAPI v3 规范生成   │
└──────────────────────┘
           │
           ▼
┌──────────────────────────────────────────────────┐
│                   依赖模块                         │
├──────────┬──────────┬──────────┬──────────────────┤
│ os/glog  │ os/gsession│ os/gcache│ container/gmap  │
│ os/gfile │ os/gview   │ os/gtime │ container/gtype │
│ os/gres  │ os/gctx    │ os/gproc │ container/glist │
├──────────┴──────────┴──────────┴──────────────────┤
│ util/gconv  │ util/gvalid  │ errors/gerror        │
│ text/gstr   │ text/gregex  │ encoding/gjson       │
├─────────────┴─────────────┴───────────────────────┤
│             net/gtrace (OpenTelemetry)             │
│             net/gsvc  (Service Discovery)          │
│             internal/graceful (Graceful Restart)   │
└───────────────────────────────────────────────────┘
```

### 关键依赖说明

| 依赖模块 | 在 ghttp/gclient 中的用途 |
|---------|--------------------------|
| `container/gmap` | Server 单例映射（`serverMapping`） |
| `container/gtype` | 并发安全的原子计数器 |
| `container/glist` | 路由树中 handler 列表 |
| `os/gcache` | 路由查找缓存 |
| `os/gsession` | Session 管理 |
| `os/gview` | 模板渲染 |
| `os/glog` | 日志输出 |
| `util/gconv` | 参数类型转换 |
| `util/gvalid` | 请求参数校验 |
| `encoding/gjson` | JSON 请求体解析 |
| `errors/gerror` | 错误处理与堆栈追踪 |
| `net/gtrace` | OpenTelemetry 集成（默认启用） |
| `net/gsvc` | 服务注册与发现 |
| `net/gsel` | 客户端负载均衡 |
| `github.com/gorilla/websocket` | WebSocket 支持 |
| `github.com/olekukonko/tablewriter` | 路由表格式化输出 |

---

## 附录：完整 Server 方法列表

### 生命周期

| 方法 | 签名 |
|------|------|
| `Start` | `func (s *Server) Start() error` |
| `Run` | `func (s *Server) Run()` |
| `Shutdown` | `func (s *Server) Shutdown()` |
| `Status` | `func (s *Server) Status() ServerStatus` |
| `Wait` | `func Wait()` |

### 配置

| 方法 | 签名 |
|------|------|
| `SetConfig` | `func (s *Server) SetConfig(c ServerConfig) error` |
| `SetConfigWithMap` | `func (s *Server) SetConfigWithMap(m map[string]any) error` |
| `SetAddr` | `func (s *Server) SetAddr(address string)` |
| `SetPort` | `func (s *Server) SetPort(port ...int)` |
| `SetHTTPSAddr` | `func (s *Server) SetHTTPSAddr(address string)` |
| `SetHTTPSPort` | `func (s *Server) SetHTTPSPort(port ...int)` |
| `SetListener` | `func (s *Server) SetListener(listeners ...net.Listener) error` |
| `EnableHTTPS` | `func (s *Server) EnableHTTPS(certFile, keyFile string, tlsConfig ...*tls.Config)` |
| `SetTLSConfig` | `func (s *Server) SetTLSConfig(tlsConfig *tls.Config)` |
| `SetReadTimeout` | `func (s *Server) SetReadTimeout(t time.Duration)` |
| `SetWriteTimeout` | `func (s *Server) SetWriteTimeout(t time.Duration)` |
| `SetIdleTimeout` | `func (s *Server) SetIdleTimeout(t time.Duration)` |
| `SetMaxHeaderBytes` | `func (s *Server) SetMaxHeaderBytes(b int)` |
| `SetServerAgent` | `func (s *Server) SetServerAgent(agent string)` |
| `SetKeepAlive` | `func (s *Server) SetKeepAlive(enabled bool)` |
| `SetView` | `func (s *Server) SetView(view *gview.View)` |
| `GetName` | `func (s *Server) GetName() string` |
| `SetName` | `func (s *Server) SetName(name string)` |
| `SetHandler` | `func (s *Server) SetHandler(h func(w http.ResponseWriter, r *http.Request))` |
| `GetHandler` | `func (s *Server) GetHandler() func(w http.ResponseWriter, r *http.Request)` |
| `SetRegistrar` | `func (s *Server) SetRegistrar(registrar gsvc.Registrar)` |
| `GetRegistrar` | `func (s *Server) GetRegistrar() gsvc.Registrar` |
| `SetEndpoints` | `func (s *Server) SetEndpoints(endpoints []string)` |

### 路由注册

| 方法 | 签名 |
|------|------|
| `BindHandler` | `func (s *Server) BindHandler(pattern string, handler HandlerFunc)` |
| `BindObject` | `func (s *Server) BindObject(pattern string, obj any, methods ...string)` |
| `BindObjectRest` | `func (s *Server) BindObjectRest(pattern string, obj any)` |
| `BindHookHandler` | `func (s *Server) BindHookHandler(pattern string, hook HookName, handler HandlerFunc)` |
| `BindStatusHandler` | `func (s *Server) BindStatusHandler(status int, handler HandlerFunc)` |
| `BindStatusHandlerByMap` | `func (s *Server) BindStatusHandlerByMap(handlerMap map[int]HandlerFunc)` |
| `Use` | `func (s *Server) Use(handlers ...HandlerFunc)` |
| `Group` | `func (s *Server) Group(prefix string, groups ...func(group *RouterGroup)) *RouterGroup` |
| `Domain` | `func (s *Server) Domain(domains ...string) *Domain` |

### 静态文件

| 方法 | 签名 |
|------|------|
| `SetServerRoot` | `func (s *Server) SetServerRoot(root string)` |
| `AddSearchPath` | `func (s *Server) AddSearchPath(path string)` |
| `AddStaticPath` | `func (s *Server) AddStaticPath(prefix string, path string)` |

### 信息获取

| 方法 | 签名 |
|------|------|
| `GetRoutes` | `func (s *Server) GetRoutes() []RouterItem` |
| `GetOpenApi` | `func (s *Server) GetOpenApi() *goai.OpenApiV3` |
| `GetListenedPort` | `func (s *Server) GetListenedPort() int` |
| `GetListenedHTTPSPort` | `func (s *Server) GetListenedHTTPSPort() int` |
| `GetListenedPorts` | `func (s *Server) GetListenedPorts() []int` |
| `GetListenedAddress` | `func (s *Server) GetListenedAddress() string` |
| `Logger` | `func (s *Server) Logger() *glog.Logger` |

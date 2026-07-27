# GoFrame v2 AI Agent 开发参考手册

> 本文档是给 AI 编程助手（Claude、Copilot 等）的 GoFrame 开发规范。所有代码模板可直接使用。

---

## 1. 项目结构

### 1.1 标准目录（`gf init` 生成）

```
myapp/
├── api/                    # API 接口定义
│   └── {module}/
│       └── v1/
│           └── {module}.go # Req/Res 结构体
├── internal/
│   ├── controller/         # 控制器（处理 HTTP 请求）
│   │   └── {module}/
│   ├── service/            # 业务逻辑（不要使用 logic/ 目录）
│   │   └── {module}/
│   ├── model/
│   │   ├── entity/         # 数据库表实体（gf gen dao 生成，禁止手动修改）
│   │   └── do/             # 数据操作对象（gf gen dao 生成，禁止手动修改）
│   └── dao/                # 数据访问层（gf gen dao 生成，禁止手动修改）
├── manifest/
│   ├── config/             # 配置文件
│   │   └── config.yaml
│   └── sql/                # SQL 文件
├── resource/               # 静态资源（模板、公共文件等）
├── api.go                  # 入口：注册所有 API
├── logic.go                # 入口：注册所有 Service（如有 logic 目录）
└── main.go                 # 程序入口
```

### 1.2 main.go 模板

```go
package main

import (
    "github.com/gogf/gf/v2/os/gctx"

    "myapp/internal/cmd"
)

func main() {
    cmd.Main.Run(gctx.New())
}
```

### 1.3 internal/cmd/cmd.go 模板

```go
package cmd

import (
    "context"
    "myapp/internal/controller/hello"
    "myapp/api/hello/v1"

    "github.com/gogf/gf/v2/frame/g"
    "github.com/gogf/gf/v2/net/ghttp"
    "github.com/gogf/gf/v2/os/gcmd"
)

var (
    Main = gcmd.Command{
        Name:  "main",
        Usage: "myapp",
        Brief: "My application",
        Func: func(ctx context.Context, parser *gcmd.Parser) (err error) {
            s := g.Server()
            hello.NewV1().Register(s.Group("/api"))
            s.Run()
            return nil
        },
    }
)
```

---

## 2. API 定义（规范化路由）

### 2.1 api/hello/v1/hello.go

```go
package v1

import "github.com/gogf/gf/v2/frame/g"

// Req/Res 结构体通过 g.Meta 标签定义路由
type HelloReq struct {
    g.Meta `path:"/hello" method:"get" tags:"Hello" summary:"打招呼"`
    Name   string `v:"required|length:1,50" dc:"用户名" in:"query"`
}

type HelloRes struct {
    Message string `json:"message" dc:"返回消息"`
}

// 使用 g.Meta 定义多个路由到同一个处理函数
type ListReq struct {
    g.Meta `path:"/list" method:"get" tags:"Items" summary:"列表查询"`
    Page   int `v:"required|min:1" d:"1" dc:"页码" in:"query"`
    Size   int `v:"required|between:1,100" d:"20" dc:"每页数量" in:"query"`
}

type ListRes struct {
    List  []Item `json:"list"`
    Total int    `json:"total"`
    Page  int    `json:"page"`
    Size  int    `json:"size"`
}

type Item struct {
    Id   int    `json:"id"`
    Name string `json:"name"`
}
```

### 2.2 g.Meta 路由标签速查

| 标签 | 说明 | 示例 |
|------|------|------|
| `path` | URL 路径 | `path:"/user/{id}"` |
| `method` | HTTP 方法 | `method:"post"` 或 `method:"put,patch"` |
| `tags` | OpenAPI 分组 | `tags:"用户管理"` |
| `summary` | 接口摘要 | `summary:"创建用户"` |
| `mime` | 请求 Content-Type | `mime:"application/json"` |
| `deprecated` | 标记废弃 | `deprecated:"true"` |
| `security` | 安全方案 | `security:"Bearer"` |

### 2.3 参数来源（`in` 标签）

| `in` 值 | 来源 | 默认 |
|---------|------|------|
| `query` | URL 查询参数 | GET 默认 |
| `form` | 表单/body | POST 默认 |
| `header` | HTTP 头 | - |
| `cookie` | Cookie | - |
| `path` | URL 路径参数 | `{id}` 字段自动 |

### 2.4 校验标签（`v` 标签）

```go
type CreateUserReq struct {
    g.Meta   `path:"/user" method:"post"`
    Name     string `v:"required|length:2,30" dc:"姓名"`
    Email    string `v:"required|email" dc:"邮箱"`
    Age      int    `v:"required|between:1,150" dc:"年龄"`
    Password string `v:"required|length:8,32" dc:"密码"`
}

// 自定义错误消息（用 # 分隔）
type LoginReq struct {
    g.Meta   `path:"/login" method:"post"`
    Username string `v:"required#请输入用户名|length:3,30#用户名长度为3-30位" dc:"用户名"`
    Password string `v:"required#请输入密码" dc:"密码"`
}
```

---

## 3. Controller

### 3.1 标准控制器模板

```go
package hello

import (
    "context"
    "myapp/api/hello/v1"
)

type ControllerV1 struct{}

func NewV1() *ControllerV1 {
    return &ControllerV1{}
}

func (c *ControllerV1) Hello(ctx context.Context, req *v1.HelloReq) (res *v1.HelloRes, err error) {
    return &v1.HelloRes{
        Message: "Hello " + req.Name,
    }, nil
}

func (c *ControllerV1) List(ctx context.Context, req *v1.ListReq) (res *v1.ListRes, err error) {
    // 调用 logic 层
    return &v1.ListRes{
        List:  []v1.Item{},
        Total: 0,
        Page:  req.Page,
        Size:  req.Size,
    }, nil
}
```

### 3.2 注册路由

```go
// 在 internal/cmd/cmd.go 或 api.go 中
s := g.Server()
hello.NewV1().Register(s)  // 自动注册所有 g.Meta 定义的路由
```

---

## 4. 数据库操作

### 4.1 配置（manifest/config/config.yaml）

```yaml
database:
  default:
    type: "mysql"
    link: "mysql:root:123456@tcp(127.0.0.1:3306)/mydb?charset=utf8mb4&parseTime=true&loc=Local"
    debug: true          # 开发环境开启 SQL 日志
    maxIdle: 10
    maxOpen: 100
    maxLifetime: "30s"
```

**必须空导入驱动**：

```go
import _ "github.com/gogf/gf/contrib/drivers/mysql/v2"
```

### 4.2 Model CRUD 速查

> **重要**：所有数据库写操作（Insert/Update/Save）必须使用 DO 对象，禁止使用 `g.Map` 或 `map[string]interface{}`。DO 结构体的字段类型为 `interface{}`，未设置的字段保持 `nil`，ORM 会自动忽略。

```go
// ========== 查询 ==========
// 单条（使用 DAO）
one, err := dao.User.Ctx(ctx).Where("id", 1).One()
// 多条
all, err := dao.User.Ctx(ctx).Where("age > ?", 18).All()
// 分页
all, totalCount, err := dao.User.Ctx(ctx).Page(1, 20).AllAndCount()

// 指定字段
one, err := dao.User.Ctx(ctx).Fields("id,name").Where("id", 1).One()

// 排序
all, err := dao.User.Ctx(ctx).Order("created_at DESC").All()

// IN 查询
all, err := dao.User.Ctx(ctx).WhereIn("id", []int{1, 2, 3}).All()

// LIKE
all, err := dao.User.Ctx(ctx).WhereLike("name", "%张%").All()

// 统计
count, err := dao.User.Ctx(ctx).Where("status", 1).Count()

// ========== 插入（必须用 DO 对象）==========
_, err := dao.User.Ctx(ctx).Data(do.User{
    Name:  "张三",
    Age:   25,
    Email: "zhang@example.com",
}).Insert()

// 条件字段更新：仅设置需要更新的字段，未设置字段为 nil 自动忽略
data := do.User{}
if name != "" {
    data.Name = name
}
if age > 0 {
    data.Age = age
}
_, err = dao.User.Ctx(ctx).Where("id", id).Data(data).Update()

// 批量插入
_, err := dao.User.Ctx(ctx).Data([]do.User{
    {Name: "张三", Age: 25},
    {Name: "李四", Age: 30},
}).Batch(100).Insert()

// 插入并获取自增 ID
id, err := dao.User.Ctx(ctx).Data(do.User{Name: "张三"}).InsertAndGetId()

// ========== 更新（必须用 DO 对象）==========
_, err := dao.User.Ctx(ctx).Data(do.User{Age: 26}).Where("id", 1).Update()

// 显式设置字段为 NULL
_, err := dao.User.Ctx(ctx).Where("id", id).Data(do.User{
    AvatarUrl: gdb.Raw("NULL"),
}).Update()

// 增量更新
_, err := dao.User.Ctx(ctx).Where("id", 1).Increment("login_count", 1)

// ========== 删除（支持软删除，见第 4.6 节）==========
_, err := dao.User.Ctx(ctx).Where("id", 1).Delete()

// ========== Upsert ==========
_, err := dao.User.Ctx(ctx).
    Data(do.User{Id: 1, Name: "张三"}).
    OnConflict("id").
    Save()
```

### 4.3 使用 Entity/DO（推荐）

```go
// gf gen dao 自动生成以下文件：
// internal/model/entity/user.go   — 表结构映射
// internal/model/do/user.go       — 数据操作对象（字段为指针，可区分零值和未设置）
// internal/dao/user.go            — DAO 层

// 使用 DAO
_, err := dao.User.Ctx(ctx).Data(do.User{
    Name:  "张三",
    Age:   25,
    Email: "zhang@example.com",
}).Insert()

// 查询映射到 Entity
var entity *entity.User
err := dao.User.Ctx(ctx).Where("id", 1).Scan(&entity)

// 查询映射到自定义结构体
type UserInfo struct {
    Id   int    `json:"id"`
    Name string `json:"name"`
}
var list []UserInfo
err := dao.User.Ctx(ctx).Fields("id,name").Scan(&list)
```

### 4.4 事务

```go
// 方式 1：闭包事务（推荐，自动提交/回滚）
err := g.DB().Transaction(ctx, func(ctx context.Context, tx gdb.TX) error {
    _, err := dao.User.Ctx(ctx).Data(do.User{Name: "张三"}).Insert()
    if err != nil {
        return err // 返回 error 自动回滚
    }
    _, err = dao.Account.Ctx(ctx).Data(do.Account{UserId: 1}).Insert()
    return err // 返回 nil 自动提交
})

// 方式 2：在闭包内用 Ctx 传递事务（推荐）
err := g.DB().Transaction(ctx, func(ctx context.Context, tx gdb.TX) error {
    // dao 内部通过 ctx 获取事务连接
    _, err := dao.User.Ctx(ctx).Data(do.User{Name: "张三"}).Insert()
    if err != nil {
        return err
    }
    _, err = dao.Account.Ctx(ctx).Data(do.Account{UserId: 1}).Insert()
    return err
})
```

### 4.5 Where 条件构建

```go
// 直接参数
Where("id", 1)                    // WHERE id = 1
Where("age > ?", 18)              // WHERE age > 18
Where("status", 1).Where("age > ?", 18) // WHERE status = 1 AND age > 18

// OmitEmpty — 跳过零值条件（构建动态查询非常有用）
m := db.Model("user").OmitEmpty()
m = m.Where("name", req.Name)     // req.Name="" 时自动跳过
m = m.Where("age", req.Age)       // req.Age=0 时自动跳过
all, err := m.All()

// Map 条件
Where(g.Map{"status": 1, "age": 25})

// WhereIn / WhereNotIn
WhereIn("id", []int{1, 2, 3})
WhereNotIn("status", []int{0, -1})

// WhereLike / WhereNotLike
WhereLike("name", "%张%")

// WhereBetween
WhereBetween("age", 18, 60)

// WhereOR
Where("status", 1).WhereOR("is_admin", 1)
```

### 4.6 软删除与时间维护

GoFrame 提供**自动**软删除和时间维护功能。当表包含 `created_at`、`updated_at`、`deleted_at` 字段时，ORM 自动处理。

**自动时间字段行为**：

| 字段 | 自动行为 |
|------|----------|
| `created_at` | Insert/InsertAndGetId 时自动写入，之后不再修改 |
| `updated_at` | Insert/Update/Save 时自动写入 |
| `deleted_at` | Delete 时自动写入（软删除），查询时自动过滤 |

**关键规则**：

```go
// ❌ 错误：手动设置时间字段（冗余！框架自动处理）
dao.User.Ctx(ctx).Data(do.User{
    Name:      "john",
    CreatedAt: gtime.Now(),  // 冗余！框架自动处理
    UpdatedAt: gtime.Now(),  // 冗余！框架自动处理
}).Insert()

// ✅ 正确：让框架处理时间字段
dao.User.Ctx(ctx).Data(do.User{
    Name: "john",
}).Insert()

// ❌ 错误：手动添加软删除条件（冗余！框架自动添加）
dao.User.Ctx(ctx).
    Where(do.User{Status: 1}).
    WhereNull(cols.DeletedAt).  // 冗余！框架自动添加 deleted_at IS NULL
    Scan(&list)

// ✅ 正确：框架自动添加 deleted_at IS NULL
dao.User.Ctx(ctx).
    Where(do.User{Status: 1}).
    Scan(&list)

// ✅ 正确：使用 Delete() 进行软删除
// 框架自动转为 UPDATE SET deleted_at = NOW()
dao.User.Ctx(ctx).Where("id", id).Delete()

// ❌ 错误：手动 Update 设置 deleted_at
dao.User.Ctx(ctx).
    Where("id", id).
    Data(do.User{DeletedAt: gtime.Now()}).  // 冗余！
    Update()
```

**`deleted_at` 支持多种字段类型**：
- DateTime/Timestamp：存储删除时间（默认）
- Integer：存储 Unix 时间戳（秒）
- Boolean：存储 0/1 删除状态

**可选配置**（`config.yaml`）：

```yaml
database:
  default:
    createdAt: "created_at"      # 自定义字段名
    updatedAt: "updated_at"
    deletedAt: "deleted_at"
    timeMaintainDisabled: false  # 设为 true 禁用此功能
```

---

## 5. 配置管理

### 5.1 config.yaml 结构模板

```yaml
server:
  address: ":8000"
  openapiPath: "/api.json"
  swaggerPath: "/swagger"

database:
  default:
    type: "mysql"
    link: "mysql:root:123456@tcp(127.0.0.1:3306)/mydb"
    debug: true

redis:
  default:
    address: "127.0.0.1:6379"
    db: 0

logger:
  level: "all"
  stdout: true

# 自定义配置
app:
  name: "myapp"
  mode: "develop"   # develop / testing / staging / product
```

### 5.2 代码中读取配置

```go
// 获取字符串
val := g.Cfg().MustGet(ctx, "app.name").String()        // "myapp"

// 获取整数
port := g.Cfg().MustGet(ctx, "server.port").Int()        // 0（默认值）

// 带默认值
val := g.Cfg().MustGet(ctx, "app.mode", "product").String()

// 直接映射到结构体
type AppConfig struct {
    Name string `json:"name"`
    Mode string `json:"mode"`
}
var config AppConfig
g.Cfg().MustGet(ctx, "app").Scan(&config)
// config = {Name: "myapp", Mode: "develop"}

// 获取整个配置
data, err := g.Cfg().Data(ctx)
```

---

## 6. Redis 操作

### 6.1 配置

```yaml
redis:
  default:
    address: "127.0.0.1:6379"
    pass: ""
    db: 0
```

**必须空导入**：

```go
import _ "github.com/gogf/gf/contrib/nosql/redis/v2"
```

### 6.2 常用操作

```go
redis := g.Redis()

// String
redis.Set(ctx, "key", "value")
redis.Set(ctx, "key", "value", 3600)  // 1小时过期
val, _ := redis.Get(ctx, "key")
redis.Del(ctx, "key")

// 检查是否存在
n, _ := redis.Exists(ctx, "key")

// 设置过期
redis.Expire(ctx, "key", 3600)

// Hash
redis.HSet(ctx, "user:1", "name", "张三")
redis.HMSet(ctx, "user:1", g.Map{"name": "张三", "age": "25"})
val, _ := redis.HGet(ctx, "user:1", "name")
all, _ := redis.HGetAll(ctx, "user:1")

// List
redis.LPush(ctx, "queue", "item1", "item2")
val, _ := redis.RPop(ctx, "queue")

// Set
redis.SAdd(ctx, "tags", "go", "golang")
members, _ := redis.SMembers(ctx, "tags")

// 自增
n, _ := redis.Incr(ctx, "counter")

// 通用 Do（可执行任意 Redis 命令）
result, err := redis.Do(ctx, "ZADD", "leaderboard", 100, "player1")
```

---

## 7. 缓存（gcache）

```go
cache := gcache.New()

// Set
cache.Set(ctx, "key", "value", 3600*time.Second)

// Get
val, err := cache.Get(ctx, "key")

// GetOrSet（不存在时自动创建，适合热点数据）
val, err := cache.GetOrSet(ctx, "user:1", func(ctx context.Context) (any, error) {
    return dao.User.Ctx(ctx).Where("id", 1).One()
}, 3600*time.Second)

// 删除
cache.Remove(ctx, "key")

// 清空
cache.Clear(ctx)
```

---

## 8. 日志

```go
// 基本日志
g.Log().Info(ctx, "用户登录", "userId", 123)
g.Log().Error(ctx, "数据库查询失败", "err", err)
g.Log().Warning(ctx, "限流警告", "ip", "1.2.3.4")
g.Log().Debug(ctx, "调试信息", "data", result)

// 格式化
g.Log().Infof(ctx, "用户 %s 登录成功，耗时 %dms", name, duration)

// 分类日志（写到不同文件）
g.Log("cron").Info(ctx, "定时任务执行", "task", "cleanup")

// 日志中打印结构体
g.Log().Info(ctx, "请求参数", g.Map{"name": "张三", "age": 25})
```

### 日志配置

```yaml
logger:
  level: "all"        # all/dev/info/warn/error/panic/fatal
  stdout: true        # 控制台输出
  ctxKeys: ["RequestId", "UserId"]  # 自动打印上下文字段
```

---

## 9. 中间件

### 9.1 自定义中间件

```go
func MiddlewareAuth(r *ghttp.Request) {
    token := r.Header.Get("Authorization")
    if token == "" {
        r.Response.WriteStatus(401, g.Map{"code": 401, "message": "未登录"})
        return
    }
    // 验证 token...
    userId := parseToken(token)
    r.SetParam("userId", userId)  // 传递到后续处理
    r.Middleware.Next()            // 继续执行
}
```

### 9.2 注册中间件

```go
s := g.Server()

// 全局中间件
s.Use(MiddlewareAuth)

// 分组中间件
s.Group("/api", func(group *ghttp.RouterGroup) {
    group.Middleware(MiddlewareAuth)
    group.Bind(hello.NewV1())
})

// 公开路由（不需要认证）
s.Group("/api", func(group *ghttp.RouterGroup) {
    group.Bind(login.NewV1())  // 不加中间件
})
```

### 9.3 CORS 中间件

```go
func MiddlewareCORS(r *ghttp.Request) {
    r.Response.CORSDefault()
    r.Middleware.Next()
}
```

---

## 10. 响应格式

### 10.1 统一响应结构

```go
// 定义统一响应
type JsonRes struct {
    Code    int         `json:"code"`
    Message string      `json:"message"`
    Data    interface{} `json:"data,omitempty"`
}

// 在中间件或 Res 结构体中使用
func success(data interface{}) *JsonRes {
    return &JsonRes{Code: 0, Message: "ok", Data: data}
}

func fail(code int, msg string) *JsonRes {
    return &JsonRes{Code: code, Message: msg}
}
```

### 10.2 控制器中返回错误

```go
import "github.com/gogf/gf/v2/errors/gerror"
import "github.com/gogf/gf/v2/errors/gcode"

func (c *ControllerV1) GetUser(ctx context.Context, req *v1.GetUserReq) (res *v1.GetUserRes, err error) {
    user, err := dao.User.Ctx(ctx).Where("id", req.Id).One()
    if err != nil {
        return nil, err
    }
    if user == nil {
        return nil, gerror.NewCode(gcode.CodeNotFound, "用户不存在")
    }
    return &v1.GetUserRes{
        Name: user["name"].String(),
    }, nil
}
```

---

## 11. 错误处理

### 11.1 创建和包装错误

```go
// 创建错误
err := gerror.New("参数无效")
err := gerror.Newf("用户 %d 不存在", userId)

// 带错误码
err := gerror.NewCode(gcode.CodeNotFound, "用户不存在")
err := gerror.NewCodef(gcode.CodeInvalidParameter, "参数 %s 格式错误", field)

// 包装错误（保留堆栈 + 错误码）
err = gerror.Wrap(err, "查询用户失败")
err = gerror.Wrapf(err, "查询用户 %d 失败", userId)

// 带错误码包装
err = gerror.WrapCode(gcode.CodeDbOperationError, err, "数据库操作失败")
```

### 11.2 提取错误信息

```go
// 获取错误码
code := gerror.Code(err)             // gcode.Code

// 检查错误码
if gerror.HasCode(err, gcode.CodeNotFound) {
    // 处理 404
}

// 获取根因
cause := gerror.Cause(err)           // 最内层的 error

// 获取堆栈
stack := gerror.Stack(err)           // 堆栈字符串
```

---

## 12. 类型转换（gconv）

```go
// 基本转换
gconv.String(123)         // "123"
gconv.Int("456")          // 456
gconv.Float64("3.14")     // 3.14
gconv.Bool("true")        // true

// Map → Struct
type User struct {
    Name string `json:"name" c:"user_name"` // c 标签优先于 json
    Age  int    `json:"age"`
}
var user User
gconv.Struct(map[string]any{"user_name": "张三", "age": "25"}, &user)
// user = {Name: "张三", Age: 25}

// 切片
gconv.SliceInt("1,2,3")           // [1,2,3]
gconv.SliceStr("a,b,c")           // ["a","b","c"]

// JSON 字符串 → Struct
jsonStr := `{"name":"张三","age":25}`
var user User
gconv.Struct(jsonStr, &user)
```

---

## 13. 校验（gvalid）

```go
// 单值校验
err := gvalid.New().Rules("required|email").Data("invalid").Run(ctx)

// 结构体标签校验（自动在 Controller 中执行，无需手动调用）
type Req struct {
    g.Meta `path:"/user" method:"post"`
    Name   string `v:"required|length:2,30" dc:"姓名"`
    Email  string `v:"required|email" dc:"邮箱"`
}

// 手动校验
err := gvalid.New().Data(req).Run(ctx)
if err != nil {
    // err 是 gvalid.Error 接口
    msgs := err.Strings()       // 所有错误消息
    first := err.FirstError()   // 第一个错误
}
```

---

## 14. 常用工具

### 14.1 字符串（gstr）

```go
gstr.Contains("hello", "ell")       // true
gstr.ToUpper("hello")               // "HELLO"
gstr.ToLower("HELLO")               // "hello"
gstr.Trim("  hello  ")              // "hello"
gstr.Replace("hello", "l", "L")     // "heLLo"
gstr.Split("a,b,c", ",")            // ["a","b","c"]
gstr.Join([]string{"a","b"}, "-")   // "a-b"
gstr.CamelCase("hello_world")       // "HelloWorld"
gstr.SnakeCase("HelloWorld")        // "hello_world"
gstr.SubStr("hello world", 0, 5)    // "hello"
gstr.IsNumeric("12345")             // true
```

### 14.2 时间（gtime）

```go
now := gtime.Now()
now.Format("Y-m-d H:i:s")         // "2026-06-13 15:30:00"
now.Timestamp()                     // 1749802200
now.StartOfDay()                    // 当天 00:00:00
now.EndOfDay()                      // 当天 23:59:59

// 解析
t, _ := gtime.StrToTime("2026-06-13 15:30:00")

// 计算
t.AddDate(0, 1, 0)                  // 加一个月
t.StartOfMonth()                    // 月初
```

### 14.3 文件（gfile）

```go
gfile.GetContents("/path/to/file")        // 读取文件内容
gfile.PutContents("/path/to/file", data)  // 写入文件
gfile.Exists("/path/to/file")             // 文件是否存在
gfile.IsDir("/path/to/dir")               // 是否是目录
gfile.Mkdir("/path/to/dir")               // 创建目录
gfile.Remove("/path/to/file")             // 删除
gfile.ScanDir("/path", "*.go", true)      // 扫描目录
```

### 14.4 随机（grand）

```go
grand.N(1, 100)       // [1, 100] 随机整数
grand.S(16)           // 16位随机字符串（字母+数字）
grand.S(8, true)      // 8位随机字符串（含特殊符号）
```

### 14.5 唯一 ID（guid）

```go
id := guid.S()        // 32位全局唯一字符串
```

---

## 15. 服务发现与注册

```go
import "github.com/gogf/gf/contrib/registry/etcd/v2"
import "github.com/gogf/gf/v2/net/gsvc"

// 设置全局注册中心
gsvc.SetRegistry(etcd.New("127.0.0.1:2379"))

// HTTP Server 自动注册
s := g.Server()
s.SetRegistry(etcd.New("127.0.0.1:2379"))
s.Run()
```

---

## 16. 链路追踪

```go
import "github.com/gogf/gf/contrib/trace/otlpgrpc/v2"

func main() {
    shutdown, err := otlpgrpc.Init("my-service", "localhost:4317", "")
    if err != nil {
        panic(err)
    }
    defer shutdown(context.Background())

    // 后续所有 HTTP/Redis/DB 操作自动链路追踪
    s := g.Server()
    s.Run()
}
```

---

## 17. 关键开发约束（必须遵守）

### 17.1 代码生成文件禁止手动修改

`gf gen dao` 生成的以下文件**禁止手动创建或修改**：

- `internal/dao/` — 数据访问层
- `internal/model/do/` — 数据操作对象
- `internal/model/entity/` — 表实体

如需修改数据库结构，应修改 SQL 后重新执行 `gf gen dao`。

### 17.2 业务逻辑放 `service/` 而非 `logic/`

除非用户明确要求，**不要使用 `logic/` 目录**。业务逻辑直接放在 `service/` 目录中。

### 17.3 数据库操作必须用 DO 对象

```go
// ✅ 正确 — 使用 DO 对象
dao.Users.Ctx(ctx).Where("id", id).Data(do.User{Uid: uid}).Update()

// ✅ 正确 — 条件字段，未设置的字段为 nil，ORM 自动忽略
data := do.User{}
if password != "" {
    data.PasswordHash = hash
}
if isAdmin != nil {
    data.IsAdmin = *isAdmin
}
dao.Users.Ctx(ctx).Where("id", id).Data(data).Update()

// ✅ 正确 — 显式设置字段为 NULL
dao.Instances.Ctx(ctx).Where("id", id).
    Data(do.Instance{IdleSince: gdb.Raw("NULL")}).Update()

// ❌ 错误 — 禁止使用 g.Map 进行数据库操作
dao.Users.Ctx(ctx).Data(g.Map{"uid": uid}).Update()
```

### 17.4 var 块分组声明

定义 3 个及以上相关变量时，使用 `var` 块分组：

```go
// ✅ 正确
var (
    authSvc       *auth.Service
    bizCtxSvc     *bizctx.Service
    k8sSvc        *svcK8s.Service
    notebookSvc   *notebook.Service
    middlewareSvc *middleware.Service
)

// ❌ 避免
authSvc := auth.New()
bizCtxSvc := bizctx.New()
k8sSvc := svcK8s.New()
```

### 17.5 错误处理必须用 gerror

所有错误处理使用 `gerror` 组件，确保完整堆栈追踪：

```go
// ✅ 正确 — 使用 gerror
return nil, gerror.WrapCode(gcode.CodeDbOperationError, err, "查询用户失败")

// ❌ 错误 — 使用 fmt.Errorf
return nil, fmt.Errorf("查询用户失败: %w", err)

// ❌ 错误 — 使用 errors.New
return nil, errors.New("查询用户失败")
```

### 17.6 AI 容易犯的错误

| 错误 | 正确做法 |
|------|---------|
| 用 `g.Map` 做数据库操作 | 必须用 `do.Xxx{}` DO 对象 |
| 手动设置 `created_at`/`updated_at` | 框架自动处理，不要手动设 |
| 手动添加 `WhereNull(cols.DeletedAt)` | 框架自动添加软删除过滤 |
| 用 `logic/` 目录放业务逻辑 | 放 `service/` 目录 |
| 手动修改 `dao/`/`do/`/`entity/` 文件 | 重新执行 `gf gen dao` |
| 用 `fmt.Errorf` 创建错误 | 用 `gerror.New/Wrap` 确保堆栈追踪 |
| `ghttp.Request` 中用 `r.FormValue("id")` | 用 `r.Get("id")` 或在 Req 结构体中定义 |
| 手动写 `json.NewDecoder(r.Body).Decode(&req)` | 使用规范化路由，框架自动解析到 Req |
| 手动 `r.Response.WriteHeader(200)` + `json.Marshal` | 直接 return Res 结构体 |
| `_ = db.Model("user").Insert()` 忽略错误 | 必须 `_, err := ...` 并处理 error |
| `fmt.Sprintf("WHERE id = %d", id)` 拼接 SQL | 使用 `Where("id", id)` 参数化查询 |
| 手动 `sql.Open("mysql", dsn)` | 使用 `g.DB()` + 配置文件 |
| 使用 `http.NewServeMux()` | 使用 `g.Server()` |
| 使用 `log.Printf()` | 使用 `g.Log()` |
| 使用 `time.Parse()` | 使用 `gtime.StrToTime()` 或 `gconv.Time()` |
| 直接用 `database/sql` | 使用 `gdb.Model()` 链式操作 |
| 设置字段为 NULL 用 `nil` | 用 `gdb.Raw("NULL")` |

### 17.7 其他关键约束

1. **错误码**：框架保留 < 1000，自定义错误码 >= 1000
2. **配置缺失行为**：`g.DB()`/`g.Redis()` 配置缺失会 panic，其他组件静默用默认值
3. **驱动必须空导入**：`import _ "github.com/gogf/gf/contrib/drivers/mysql/v2"`
4. **事务内使用 Ctx 传递**：`dao.User.Ctx(ctx)` 中的 ctx 必须是事务闭包传入的 ctx
5. **校验自动执行**：规范化路由中 Req 的 `v` 标签校验是自动的，不需要手动调用
6. **gconv 标签优先级**：`gconv` > `param` > `c` > `p` > `json` > 字段名
7. **gmetric noop 模式**：未设置 Provider 时，所有指标操作为空操作，不影响性能
8. **DO 结构体用指针区分零值**：`do.User{Age: nil}` 表示不更新 age，`do.User{Age: &age}` 表示更新为 0

### 17.8 性能注意事项

- `gconv.Struct` 首次调用会反射缓存，后续调用极快
- `grand` 使用异步缓冲管道预生成随机数，性能远高于直接 `crypto/rand`
- `gcache` 的 `GetOrSet` 使用 `sync.Once` 语义，防止缓存击穿
- `gregex` 内部编译缓存，相同 pattern 不会重复编译正则
- `gdb.Model` 是轻量对象，每次 `db.Model("table")` 创建开销极小

---

## 18. 完整示例：RESTful CRUD

### 18.1 api/user/v1/user.go

```go
package v1

import "github.com/gogf/gf/v2/frame/g"

type CreateUserReq struct {
    g.Meta `path:"/user" method:"post" tags:"用户" summary:"创建用户"`
    Name   string `v:"required|length:2,30" dc:"姓名"`
    Email  string `v:"required|email" dc:"邮箱"`
    Age    int    `v:"required|between:1,150" dc:"年龄"`
}
type CreateUserRes struct {
    Id int `json:"id"`
}

type GetUserReq struct {
    g.Meta `path:"/user/{id}" method:"get" tags:"用户" summary:"获取用户"`
    Id     int `v:"required|min:1" dc:"用户ID" in:"path"`
}
type GetUserRes struct {
    Id    int    `json:"id"`
    Name  string `json:"name"`
    Email string `json:"email"`
    Age   int    `json:"age"`
}

type UpdateUserReq struct {
    g.Meta `path:"/user/{id}" method:"put" tags:"用户" summary:"更新用户"`
    Id     int    `v:"required|min:1" dc:"用户ID" in:"path"`
    Name   string `v:"length:2,30" dc:"姓名"`
    Email  string `v:"email" dc:"邮箱"`
    Age    int    `v:"between:1,150" dc:"年龄"`
}
type UpdateUserRes struct{}

type DeleteUserReq struct {
    g.Meta `path:"/user/{id}" method:"delete" tags:"用户" summary:"删除用户"`
    Id     int `v:"required|min:1" dc:"用户ID" in:"path"`
}
type DeleteUserRes struct{}

type ListUserReq struct {
    g.Meta `path:"/user" method:"get" tags:"用户" summary:"用户列表"`
    Page   int    `v:"required|min:1" d:"1" dc:"页码" in:"query"`
    Size   int    `v:"required|between:1,100" d:"20" dc:"每页数量" in:"query"`
    Name   string `dc:"按姓名搜索" in:"query"`
}
type ListUserRes struct {
    List  []GetUserRes `json:"list"`
    Total int          `json:"total"`
    Page  int          `json:"page"`
    Size  int          `json:"size"`
}
```

### 18.2 internal/controller/user/user.go

```go
package user

import (
    "context"
    "myapp/api/user/v1"
    "myapp/internal/dao"
    "myapp/internal/model/do"

    "github.com/gogf/gf/v2/errors/gerror"
    "github.com/gogf/gf/v2/errors/gcode"
    "github.com/gogf/gf/v2/frame/g"
)

type ControllerV1 struct{}

func NewV1() *ControllerV1 {
    return &ControllerV1{}
}

func (c *ControllerV1) CreateUser(ctx context.Context, req *v1.CreateUserReq) (res *v1.CreateUserRes, err error) {
    id, err := dao.User.Ctx(ctx).Data(do.User{
        Name:  req.Name,
        Email: req.Email,
        Age:   req.Age,
    }).InsertAndGetId()
    if err != nil {
        return nil, gerror.WrapCode(gcode.CodeDbOperationError, err, "创建用户失败")
    }
    return &v1.CreateUserRes{Id: int(id)}, nil
}

func (c *ControllerV1) GetUser(ctx context.Context, req *v1.GetUserReq) (res *v1.GetUserRes, err error) {
    var result *v1.GetUserRes
    err = dao.User.Ctx(ctx).Where("id", req.Id).Scan(&result)
    if err != nil {
        return nil, err
    }
    if result == nil {
        return nil, gerror.NewCode(gcode.CodeNotFound, "用户不存在")
    }
    return result, nil
}

func (c *ControllerV1) UpdateUser(ctx context.Context, req *v1.UpdateUserReq) (res *v1.UpdateUserRes, err error) {
    _, err = dao.User.Ctx(ctx).
        OmitEmpty().
        Data(do.User{
            Name:  req.Name,
            Email: req.Email,
            Age:   req.Age,
        }).
        Where("id", req.Id).
        Update()
    if err != nil {
        return nil, gerror.WrapCode(gcode.CodeDbOperationError, err, "更新用户失败")
    }
    return &v1.UpdateUserRes{}, nil
}

func (c *ControllerV1) DeleteUser(ctx context.Context, req *v1.DeleteUserReq) (res *v1.DeleteUserRes, err error) {
    _, err = dao.User.Ctx(ctx).Where("id", req.Id).Delete()
    if err != nil {
        return nil, gerror.WrapCode(gcode.CodeDbOperationError, err, "删除用户失败")
    }
    return &v1.DeleteUserRes{}, nil
}

func (c *ControllerV1) ListUser(ctx context.Context, req *v1.ListUserReq) (res *v1.ListUserRes, err error) {
    m := dao.User.Ctx(ctx).OmitEmpty()
    if req.Name != "" {
        m = m.WhereLike("name", "%"+req.Name+"%")
    }
    var list []v1.GetUserRes
    totalCount, err := m.Page(req.Page, req.Size).ScanAndCount(&list, true)
    if err != nil {
        return nil, gerror.WrapCode(gcode.CodeDbOperationError, err, "查询用户列表失败")
    }
    return &v1.ListUserRes{
        List:  list,
        Total: totalCount,
        Page:  req.Page,
        Size:  req.Size,
    }, nil
}
```

### 18.3 internal/cmd/cmd.go 注册路由

```go
package cmd

import (
    "context"
    "myapp/api/user/v1"
    "myapp/internal/controller/user"

    "github.com/gogf/gf/v2/frame/g"
    "github.com/gogf/gf/v2/os/gcmd"

    _ "github.com/gogf/gf/contrib/drivers/mysql/v2"
)

var (
    Main = gcmd.Command{
        Name:  "main",
        Usage: "myapp",
        Brief: "My application",
        Func: func(ctx context.Context, parser *gcmd.Parser) (err error) {
            s := g.Server()
            s.Use(func(r *ghttp.Request) {
                r.Middleware.Next()
            })
            s.Group("/api/v1", func(group *ghttp.RouterGroup) {
                group.Bind(
                    user.NewV1(),
                )
            })
            s.Run()
            return nil
        },
    }
)
```

---

## 19. 代码生成命令速查

```bash
gf init myapp                    # 创建新项目
gf init myapp -m                 # mono-repo
gf run main.go                   # 热编译运行
gf gen dao                       # 从数据库生成 entity/do/dao
gf gen ctrl                      # 从 api 定义生成控制器
gf gen service                   # 从 logic 生成服务接口
gf build main.go --os linux --arch amd64  # 交叉编译
gf pack resource,manifest packed/packed.go # 打包资源
```

---

## 20. 导入路径速查

```go
// 核心模块
"github.com/gogf/gf/v2/frame/g"
"github.com/gogf/gf/v2/net/ghttp"
"github.com/gogf/gf/v2/database/gdb"
"github.com/gogf/gf/v2/os/gctx"
"github.com/gogf/gf/v2/os/glog"
"github.com/gogf/gf/v2/os/gcfg"
"github.com/gogf/gf/v2/os/gtime"
"github.com/gogf/gf/v2/os/gfile"
"github.com/gogf/gf/v2/os/gcache"
"github.com/gogf/gf/v2/util/gconv"
"github.com/gogf/gf/v2/util/gvalid"
"github.com/gogf/gf/v2/util/guid"
"github.com/gogf/gf/v2/util/grand"
"github.com/gogf/gf/v2/text/gstr"
"github.com/gogf/gf/v2/container/gvar"
"github.com/gogf/gf/v2/container/gmap"
"github.com/gogf/gf/v2/container/garray"
"github.com/gogf/gf/v2/errors/gerror"
"github.com/gogf/gf/v2/errors/gcode"
"github.com/gogf/gf/v2/encoding/gjson"

// 扩展模块（必须空导入）
_ "github.com/gogf/gf/contrib/drivers/mysql/v2"
_ "github.com/gogf/gf/contrib/drivers/pgsql/v2"
_ "github.com/gogf/gf/contrib/nosql/redis/v2"
_ "github.com/gogf/gf/contrib/registry/etcd/v2"
"github.com/gogf/gf/contrib/trace/otlpgrpc/v2"
"github.com/gogf/gf/contrib/metric/otelmetric/v2"
```

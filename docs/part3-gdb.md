# GoFrame 数据库模块开发规范

> 本文档基于 GoFrame v2 源码（`database/gdb` + `database/gredis`）深度分析，面向框架开发者和高级使用者。

---

## 目录

- [1. 模块概览与架构设计](#1-模块概览与架构设计)
- [2. DB 接口全貌与 Core 实现](#2-db-接口全貌与-core-实现)
- [3. TX 接口与事务管理](#3-tx-接口与事务管理)
- [4. Model 链式 CRUD](#4-model-链式-crud)
- [5. WhereBuilder 条件构建器](#5-wherebuilder-条件构建器)
- [6. 驱动工厂注册机制](#6-驱动工厂注册机制)
- [7. Result/Record 数据映射](#7-resultrecord-数据映射)
- [8. 查询缓存](#8-查询缓存)
- [9. Hook 链](#9-hook-链)
- [10. 软删除支持](#10-软删除支持)
- [11. 分库分表 / Sharding](#11-分库分表--sharding)
- [12. With 关联查询](#12-with-关联查询)
- [13. 行锁（Lock）](#13-行锁lock)
- [14. gredis 适配器架构](#14-gredis-适配器架构)
- [15. 与其他模块的依赖关系](#15-与其他模块的依赖关系)

---

## 1. 模块概览与架构设计

### 1.1 包定位

| 包路径 | 职责 |
|--------|------|
| `database/gdb` | 关系型数据库 ORM 抽象层，包含查询构建器、事务管理、缓存、Hook、软删除、分库分表等 |
| `database/gredis` | Redis 客户端抽象层，定义 Adapter/Conn/Pipeline 接口，具体实现由 `contrib/nosql/redis` 提供 |

### 1.2 核心设计原则

- **驱动无关**：`gdb` 核心仅定义接口（`DB`、`TX`、`Driver`），具体驱动（MySQL、PostgreSQL 等）通过 `contrib/drivers/` 注册
- **链式调用**：`Model` 结构体提供完整的链式 API（Where → Order → Limit → Select/Insert/Update/Delete）
- **事务传播**：支持 7 种传播行为，嵌套事务通过 SAVEPOINT 实现
- **可扩展 Hook**：Select/Insert/Update/Delete 四个生命周期节点支持自定义拦截

### 1.3 核心类型层次

```
DB (interface)
 ├── Core (struct)         — 基础实现，管理连接池、配置、缓存
 ├── DriverDefault         — 默认空驱动
 └── DriverWrapperDB       — 包装层，拦截 DoCommit 做日志/追踪

TX (interface)
 └── TXCore (struct)       — 事务实现，管理嵌套事务计数、Savepoint

Model (struct)             — 链式操作构建器
 └── WhereBuilder (struct) — WHERE 条件构建器

Driver (interface)         — 驱动注册接口，仅一个方法 New()
```

---

## 2. DB 接口全貌与 Core 实现

### 2.1 DB 接口方法清单

DB 接口定义在 `gdb.go:32`，按功能分组如下：

#### 模型创建

| 方法 | 说明 |
|------|------|
| `Model(tableNameOrStruct ...any) *Model` | 创建 ORM 模型，支持表名、结构体、子查询 |
| `Raw(rawSql string, args ...any) *Model` | 基于原生 SQL 创建模型 |
| `Schema(schema string) *Schema` | 切换到指定数据库 |
| `With(objects ...any) *Model` | 基于对象元数据创建关联查询模型 |
| `Ctx(ctx context.Context) DB` | 绑定上下文（返回浅拷贝 DB） |

#### 原生查询

| 方法 | 说明 |
|------|------|
| `Query(ctx, sql, args...) (Result, error)` | 执行返回行的查询 |
| `Exec(ctx, sql, args...) (sql.Result, error)` | 执行非返回行语句 |
| `Prepare(ctx, sql, execOnMaster...) (*Stmt, error)` | 创建预处理语句 |

#### 快捷 CRUD

| 方法 | 说明 |
|------|------|
| `Insert(ctx, table, data, batch...)` | INSERT |
| `InsertIgnore(ctx, table, data, batch...)` | INSERT IGNORE |
| `InsertAndGetId(ctx, table, data, batch...)` | INSERT 并返回自增 ID |
| `Replace(ctx, table, data, batch...)` | REPLACE INTO |
| `Save(ctx, table, data, batch...)` | INSERT ON DUPLICATE KEY UPDATE |
| `Update(ctx, table, data, condition, args...)` | UPDATE |
| `Delete(ctx, table, condition, args...)` | DELETE |

#### 便捷查询

| 方法 | 说明 |
|------|------|
| `GetAll(ctx, sql, args...)` | 返回所有行 `Result` |
| `GetOne(ctx, sql, args...)` | 返回首行 `Record` |
| `GetValue(ctx, sql, args...)` | 返回首行首列 `Value` |
| `GetArray(ctx, sql, args...)` | 返回首列所有值 `Array` |
| `GetCount(ctx, sql, args...)` | 返回 COUNT 结果 |
| `GetScan(ctx, objPointer, sql, args...)` | 自动扫描到结构体/切片 |
| `Union(unions ...*Model) *Model` | UNION 查询 |
| `UnionAll(unions ...*Model) *Model` | UNION ALL 查询 |

#### 主从 / 事务 / 配置

| 方法 | 说明 |
|------|------|
| `Master(schema...) (*sql.DB, error)` | 获取主库连接 |
| `Slave(schema...) (*sql.DB, error)` | 获取从库连接 |
| `Begin(ctx) (TX, error)` | 开启事务 |
| `BeginWithOptions(ctx, opts) (TX, error)` | 带选项开启事务 |
| `Transaction(ctx, f)` | 闭包事务 |
| `TransactionWithOptions(ctx, opts, f)` | 带传播行为的闭包事务 |
| `GetCache() *gcache.Cache` | 获取缓存实例 |
| `SetDebug(bool)` / `GetDebug() bool` | 调试模式 |
| `SetDryRun(bool)` / `GetDryRun() bool` | Dry-Run 模式 |
| `SetLogger(glog.ILogger)` | 自定义日志 |
| `GetConfig() *ConfigNode` | 获取配置节点 |

#### 内部可覆写方法

| 方法 | 说明 |
|------|------|
| `DoSelect(ctx, link, sql, args...)` | SELECT 执行 |
| `DoInsert(ctx, link, table, list, option)` | INSERT 执行 |
| `DoUpdate(ctx, link, table, data, condition, args...)` | UPDATE 执行 |
| `DoDelete(ctx, link, table, condition, args...)` | DELETE 执行 |
| `DoQuery(ctx, link, sql, args...)` | Query 执行 |
| `DoExec(ctx, link, sql, args...)` | Exec 执行 |
| `DoFilter(ctx, link, sql, args)` | SQL 过滤 |
| `DoCommit(ctx, in DoCommitInput)` | 提交拦截 |
| `DoPrepare(ctx, link, sql)` | Prepare 执行 |

### 2.2 Core 结构体

```go
// gdb.go:515
type Core struct {
    db            DB                               // DB 接口对象
    ctx           context.Context                  // 链式操作上下文
    group         string                           // 配置分组名
    schema        string                           // 自定义 schema
    debug         *gtype.Bool                      // 调试开关
    cache         *gcache.Cache                    // SQL 结果缓存
    links         *gmap.KVMap[ConfigNode, *sql.DB] // 连接池缓存
    logger        glog.ILogger                     // 日志器
    config        *ConfigNode                      // 当前配置节点
    localTypeMap  *gmap.StrAnyMap                  // 字段类型转换映射
    dynamicConfig dynamicConfig                    // 运行时可调配置
    innerMemCache *gcache.Cache                    // 内部缓存（表结构等）
}
```

### 2.3 ConfigNode 配置结构

```go
// gdb_core_config.go:30
type ConfigNode struct {
    Host             string        // 服务地址
    Port             string        // 端口
    User             string        // 用户名
    Pass             string        // 密码
    Name             string        // 数据库名
    Type             string        // 数据库类型：mysql/pgsql/sqlite/...
    Link             string        // 连接串（可选，优先于其他字段）
    Role             Role          // 角色：master/slave
    Debug            bool          // 调试模式
    Prefix           string        // 表名前缀
    DryRun           bool          // Dry-Run 模式
    Weight           int           // 负载均衡权重
    Charset          string        // 字符集（默认 utf8）
    Protocol         string        // 协议（默认 tcp）
    MaxIdleConnCount int           // 最大空闲连接数
    MaxOpenConnCount int           // 最大打开连接数
    MaxConnLifeTime  time.Duration // 连接最大生命周期
    MaxIdleConnTime  time.Duration // 空闲连接最大时间
    QueryTimeout     time.Duration // 查询超时
    ExecTimeout      time.Duration // 执行超时
    TranTimeout      time.Duration // 事务超时
    PrepareTimeout   time.Duration // Prepare 超时
    CreatedAt        string        // 自动填充创建时间字段名
    UpdatedAt        string        // 自动填充更新时间字段名
    DeletedAt        string        // 自动填充删除时间字段名
    TimeMaintainDisabled bool      // 禁用时间自动维护
}
```

### 2.4 连接获取流程

```
Core.getSqlDb(master, schema...)
  ├── 通过 group 获取 ConfigGroup（支持主从分离）
  ├── 按权重随机选择 ConfigNode
  ├── 填充默认值（charset=utif8, 连接池参数）
  └── 通过 links KVMap 缓存 *sql.DB（按 ConfigNode 值做 key）
```

---

## 3. TX 接口与事务管理

### 3.1 TX 接口

TX 接口定义在 `gdb.go:351`，内嵌 `Link` 接口，提供与 `DB` 平级的 CRUD 方法。

**关键方法**：

| 方法 | 说明 |
|------|------|
| `Begin() error` | 开启嵌套事务（SAVEPOINT） |
| `Commit() error` | 提交/释放 Savepoint |
| `Rollback() error` | 回滚到 Savepoint |
| `Transaction(ctx, f)` | 闭包嵌套事务 |
| `SavePoint(point string) error` | 创建命名保存点 |
| `RollbackTo(point string) error` | 回滚到命名保存点 |

### 3.2 七种事务传播行为

定义在 `gdb_core_transaction.go:19`：

```go
type Propagation string

const (
    PropagationNested        Propagation = "NESTED"         // 默认：已有事务则嵌套，否则新建
    PropagationRequired      Propagation = "REQUIRED"       // 已有事务则加入，否则新建
    PropagationSupports      Propagation = "SUPPORTS"       // 已有事务则加入，否则非事务执行
    PropagationRequiresNew   Propagation = "REQUIRES_NEW"   // 始终新建事务，挂起当前事务
    PropagationNotSupported  Propagation = "NOT_SUPPORTED"  // 非事务执行，挂起当前事务
    PropagationMandatory     Propagation = "MANDATORY"      // 必须在已有事务中，否则报错
    PropagationNever         Propagation = "NEVER"          // 必须不在事务中，否则报错
)
```

### 3.3 嵌套事务 Savepoint 机制

`TXCore` 通过 `transactionCount` 字段跟踪嵌套深度：

```go
// gdb_core_txcore.go
type TXCore struct {
    db              DB
    tx              *sql.Tx
    transactionCount int    // 嵌套层级计数
    isClosed        bool
    // ...
}

// Begin 递增计数并创建 SAVEPOINT
func (tx *TXCore) Begin() error {
    _, err := tx.Exec("SAVEPOINT " + tx.transactionKeyForNestedPoint())
    tx.transactionCount++
    return err
}

// Commit 递减计数，层级 > 0 时 RELEASE SAVEPOINT
func (tx *TXCore) Commit() error {
    if tx.transactionCount > 0 {
        tx.transactionCount--
        _, err := tx.Exec("RELEASE SAVEPOINT " + tx.transactionKeyForNestedPoint())
        return err
    }
    // 最外层：真正 COMMIT
}

// Rollback 递减计数，层级 > 0 时 ROLLBACK TO SAVEPOINT
func (tx *TXCore) Rollback() error {
    if tx.transactionCount > 0 {
        tx.transactionCount--
        _, err := tx.Exec("ROLLBACK TO SAVEPOINT " + tx.transactionKeyForNestedPoint())
        return err
    }
    // 最外层：真正 ROLLBACK
}
```

### 3.4 事务传播实现

`TransactionWithOptions` 根据 `opts.Propagation` 分发处理：

```go
// gdb_core_transaction.go:131
func (c *Core) TransactionWithOptions(ctx, opts, f) error {
    currentTx := TXFromCtx(ctx, group)
    switch opts.Propagation {
    case PropagationRequired:
        if currentTx != nil { return f(ctx, currentTx) }
        return c.createNewTransaction(ctx, opts, f)
    case PropagationNested:
        if currentTx != nil { return currentTx.Transaction(ctx, f) }
        return c.createNewTransaction(ctx, opts, f)
    case PropagationRequiresNew:
        ctx = WithoutTX(ctx, group)  // 清除上下文中的事务
        return c.createNewTransaction(ctx, opts, f)
    // ... 其他传播行为
    }
}
```

### 3.5 使用示例

```go
// 基本闭包事务
g.DB().Transaction(ctx, func(ctx context.Context, tx gdb.TX) error {
    _, err := tx.Exec("INSERT INTO users(name) VALUES(?)", "john")
    if err != nil {
        return err
    }
    _, err = tx.Exec("UPDATE stats SET count = count + 1")
    return err
})

// 指定传播行为
g.DB().TransactionWithOptions(ctx, gdb.TxOptions{
    Propagation: gdb.PropagationRequiresNew,
    Isolation:   sql.LevelReadCommitted,
}, func(ctx context.Context, tx gdb.TX) error {
    // ...
})
```

---

## 4. Model 链式 CRUD

### 4.1 Model 结构体

```go
// gdb_model.go:19
type Model struct {
    db              DB            // 底层 DB 接口
    tx              TX            // 底层 TX 接口（事务时使用）
    rawSql          string        // 原生 SQL
    tablesInit      string        // 初始化表名
    tables          string        // 操作表名
    fields          []any         // 查询字段
    fieldsEx        []any         // 排除字段
    whereBuilder    *WhereBuilder // WHERE 条件构建器
    groupBy         string        // GROUP BY
    orderBy         string        // ORDER BY
    having          []any         // HAVING
    start           int           // LIMIT offset
    limit           int           // LIMIT count
    data            any           // 操作数据
    batch           int           // 批量插入数量
    distinct        string        // DISTINCT
    lockInfo        string        // 锁信息
    cacheEnabled    bool          // 启用缓存
    cacheOption     CacheOption   // 缓存选项
    hookHandler     HookHandler   // Hook 处理器
    unscoped        bool          // 跳过软删除
    safe            bool          // 安全模式（每次操作返回新 Model）
    shardingConfig  ShardingConfig
    shardingValue   any
    // ...
}
```

### 4.2 查询方法

| 方法 | 返回类型 | 说明 |
|------|----------|------|
| `All(where...)` | `Result` | 查询所有行 |
| `One(where...)` | `Record` | 查询单行 |
| `Value(fields...)` | `Value` | 查询单值 |
| `Array(fields...)` | `Array` | 查询列数组 |
| `Count()` | `int` | COUNT 查询 |
| `AllAndCount(useFieldForCount)` | `Result, int, error` | 同时获取结果和总数 |
| `Scan(pointer)` | `error` | 扫描到结构体 |
| `ScanList(pointer, fieldName, with)` | `error` | 扫描到结构体切片的关联字段 |
| `Chunk(size, handler)` | — | 分块迭代处理 |

### 4.3 写入方法

| 方法 | 说明 |
|------|------|
| `Insert()` | INSERT INTO |
| `InsertIgnore()` | INSERT IGNORE INTO |
| `InsertAndGetId()` | INSERT 返回自增 ID |
| `Replace()` | REPLACE INTO |
| `Save()` | INSERT ON DUPLICATE KEY UPDATE |
| `Update()` | UPDATE |
| `Delete()` | DELETE（支持软删除） |

### 4.4 链式条件方法

#### 字段选择

```go
m := db.Model("user")
m.Fields("id", "name")       // 选择字段
m.FieldsEx("password")       // 排除字段
m.Distinct()                  // 去重
m.Select("id", "name")       // 同 Fields（语义更明确）
```

#### WHERE 条件

```go
m.Where("id", 1)
m.Where("name LIKE ?", "john%")
m.Where(g.Map{"status": 1, "age >": 18})
m.WherePri(123)                    // 按主键查询
m.WhereLT("age", 30)               // age < 30
m.WhereLTE("age", 30)              // age <= 30
m.WhereGT("age", 18)               // age > 18
m.WhereGTE("age", 18)              // age >= 18
m.WhereBetween("age", 18, 30)      // BETWEEN 18 AND 30
m.WhereLike("name", "john%")       // LIKE
m.WhereIn("id", g.Slice{1, 2, 3})  // IN (1,2,3)
m.WhereNotIn("status", g.Slice{0}) // NOT IN
m.WhereNull("deleted_at")          // IS NULL
m.WhereNotNull("email")            // IS NOT NULL
m.WhereOr("status", 2)             // OR 条件
m.Wheref("id = %d", 123)           // fmt.Sprintf 风格
```

#### 排序 / 分组 / 分页

```go
m.Order("id DESC")
m.OrderAsc("name")
m.OrderDesc("created_at")
m.OrderRandom()                    // ORDER BY RAND()
m.Group("status")
m.Group("status, department")
m.Having("count > ?", 10)
m.Page(1, 20)                      // 第1页，每页20条
m.Limit(10)                        // 限制10条
m.Offset(5)                        // 偏移5条
```

#### 数据与批量

```go
m.Data(g.Map{"name": "john", "age": 30})
m.Data(&User{Name: "john"})
m.Batch(100)                       // 批量插入每批100条
```

#### 其他操作

```go
m.Ctx(ctx)                         // 绑定上下文
m.Master()                         // 强制走主库
m.Slave()                          // 强制走从库
m.Safe(true)                       // 安全模式
m.Unscoped()                       // 跳过软删除
m.As("u")                          // 表别名
m.Schema("db2")                    // 切换数据库
m.Partition("p1", "p2")            // 分区表
m.Lock(gdb.LockForUpdate)          // 行锁
m.Hook(handler)                    // Hook
m.Cache(option)                    // 缓存
m.OnDuplicateStr("name=VALUES(name)") // ON DUPLICATE KEY UPDATE
```

### 4.5 完整 CRUD 示例

```go
ctx := context.Background()
db := g.DB()

// INSERT
result, err := db.Model("user").Ctx(ctx).Data(g.Map{
    "name": "john",
    "age":  30,
}).Insert()

// INSERT 并获取 ID
id, err := db.Model("user").Ctx(ctx).Data(g.Map{
    "name": "john",
}).InsertAndGetId()

// 批量 INSERT
_, err = db.Model("user").Ctx(ctx).Data(g.List{
    {"name": "john", "age": 30},
    {"name": "smith", "age": 25},
}).Batch(10).Insert()

// UPDATE
result, err := db.Model("user").Ctx(ctx).
    Data(g.Map{"age": 31}).
    Where("id", 1).
    Update()

// UPDATE 使用 Counter
result, err := db.Model("user").Ctx(ctx).
    Data("view_count", gdb.Counter{
        Field: "view_count",
        Value: 1,
    }).
    Where("id", 1).
    Update()

// DELETE（支持软删除）
result, err := db.Model("user").Ctx(ctx).Where("id", 1).Delete()

// SELECT All
var users []User
err := db.Model("user").Ctx(ctx).Scan(&users)

// SELECT One
var user User
err := db.Model("user").Ctx(ctx).Where("id", 1).Scan(&user)

// COUNT
count, err := db.Model("user").Ctx(ctx).Where("status", 1).Count()

// 分页
var users []User
err := db.Model("user").Ctx(ctx).Page(2, 10).OrderAsc("id").Scan(&users)
```

---

## 5. WhereBuilder 条件构建器

### 5.1 概述

`WhereBuilder` 提供独立的条件构建能力，支持复杂的 AND/OR 组合和嵌套。

```go
// 创建 Builder
builder := m.Builder()

// 基本条件
builder = builder.Where("id", 1)
builder = builder.Where("status", "active")

// OR 条件
builder = builder.WhereOr("type", "vip")

// 嵌套子条件
builder = builder.Where(
    m.Builder().Where("age >", 18).WhereOr("level >", 3),
)

// 应用到 Model
m = m.Where(builder)
```

### 5.2 WhereBuilder 支持的 Where 类型

| 参数类型 | 生成的 SQL |
|----------|-----------|
| `string` + `args` | `WHERE name = ?` (带占位符) |
| `string` (无 args) | `WHERE name = 'john'` (直接拼接) |
| `g.Map` | `WHERE key1=val1 AND key2=val2` |
| `*gmap.Map` | 同 g.Map |
| `struct/*struct` | 按字段名生成 |
| `g.Slice` / `[]any` | `WHERE id IN (1,2,3)` |
| `WhereBuilder` | 嵌套子条件 |

### 5.3 WhereHolder 操作类型

```go
const (
    whereHolderOperatorWhere = 1  // WHERE
    whereHolderOperatorAnd   = 2  // AND
    whereHolderOperatorOr    = 3  // OR
    whereHolderTypeDefault   = "Default"
    whereHolderTypeNoArgs    = "NoArgs"
    whereHolderTypeIn        = "In"
)
```

---

## 6. 驱动工厂注册机制

### 6.1 Driver 接口

```go
// gdb.go:589
type Driver interface {
    New(core *Core, node *ConfigNode) (DB, error)
}
```

### 6.2 注册流程

```go
// gdb.go:906
func Register(name string, driver Driver) error {
    driverMap[name] = newDriverWrapper(driver)
    return nil
}
```

每个驱动通过 `init()` 函数自动注册：

```go
// contrib/drivers/mysql/mysql.go
func init() {
    gdb.Register("mysql", &Driver{})
}
```

### 6.3 DriverWrapper

```go
// gdb_driver_wrapper.go
type DriverWrapper struct {
    driver Driver
}

func (d *DriverWrapper) New(core *Core, node *ConfigNode) (DB, error) {
    db, err := d.driver.New(core, node)
    return &DriverWrapperDB{DB: db}, nil  // 包装为 DriverWrapperDB
}
```

`DriverWrapperDB` 内嵌 `DB` 接口，拦截 `DoCommit` 方法以添加日志记录和 OpenTelemetry 追踪。

### 6.4 已注册驱动

| 路径 | Type 名称 | 数据库 |
|------|-----------|--------|
| `contrib/drivers/mysql` | `mysql` | MySQL |
| `contrib/drivers/pgsql` | `pgsql` | PostgreSQL |
| `contrib/drivers/mssql` | `mssql` | SQL Server |
| `contrib/drivers/oracle` | `oracle` | Oracle |
| `contrib/drivers/sqlite` | `sqlite` | SQLite |
| `contrib/drivers/clickhouse` | `clickhouse` | ClickHouse |
| `contrib/drivers/dm` | `dm` | 达梦 |
| `contrib/drivers/gaussdb` | `gaussdb` | GaussDB |
| `contrib/drivers/mariadb` | `mariadb` | MariaDB |
| `contrib/drivers/oceanbase` | `oceanbase` | OceanBase |
| `contrib/drivers/tidb` | `tidb` | TiDB |
| `contrib/drivers/sqlitecgo` | `sqlitecgo` | SQLite (CGO) |

### 6.5 实例创建

```go
// 方式一：通过配置分组名
db, err := gdb.NewByGroup("default")

// 方式二：直接通过配置节点
db, err := gdb.New(gdb.ConfigNode{
    Type: "mysql",
    Link: "mysql:root:123456@tcp(127.0.0.1:3306)/test",
})

// 方式三：单例模式（推荐）
db, err := gdb.Instance("default")
```

---

## 7. Result/Record 数据映射

### 7.1 核心类型

```go
// gdb.go:673
type (
    Raw    string                    // 原生 SQL 片段
    Value  = *gvar.Var               // 字段值（gvar.Var 指针）
    Array  = gvar.Vars               // 字段值数组
    Record = map[string]Value        // 单行记录
    Result = []Record                // 多行结果集
    Map    = map[string]any          // 通用 Map
    List   = []Map                   // 通用 List
)
```

### 7.2 Result 方法列表

| 方法 | 返回类型 | 说明 |
|------|----------|------|
| `IsEmpty()` | `bool` | 是否为空 |
| `Len()` / `Size()` | `int` | 记录数 |
| `Chunk(size int)` | `[]Result` | 分块 |
| `Json()` | `string` | 转 JSON |
| `Xml(rootTag...)` | `string` | 转 XML |
| `List()` | `List` | 转为 `[]map[string]any` |
| `Array(field...)` | `Array` | 提取列值数组 |
| `MapKeyValue(key)` | `map[string]Value` | 按 key 构建映射 |
| `MapKeyStr(key)` | `map[string]Map` | 按 string key 构建 Map |
| `MapKeyInt(key)` | `map[int]Map` | 按 int key 构建 Map |
| `MapKeyUint(key)` | `map[uint]Map` | 按 uint key 构建 Map |
| `RecordKeyStr(key)` | `map[string]Record` | 按 string key 构建 Record |
| `RecordKeyInt(key)` | `map[int]Record` | 按 int key 构建 Record |
| `RecordKeyUint(key)` | `map[uint]Record` | 按 uint key 构建 Record |
| `Structs(pointer)` | `error` | 映射到结构体切片 |

### 7.3 Record 方法

| 方法 | 返回类型 | 说明 |
|------|----------|------|
| `Map()` | `Map` | 转为 `map[string]any` |
| `Struct(pointer)` | `error` | 映射到单个结构体 |
| `IsNil()` | `bool` | 是否为空 |

### 7.4 数据映射示例

```go
// Result → 结构体切片
var users []User
err := db.Model("user").Scan(&users)

// Result → 单个结构体
var user User
err := db.Model("user").Where("id", 1).Scan(&user)

// Result → JSON
result, _ := db.Model("user").All()
jsonStr := result.Json()

// Record 提取字段值
one, _ := db.Model("user").Where("id", 1).One()
name := one["name"].String()
age := one["age"].Int()
```

---

## 8. 查询缓存

### 8.1 CacheOption 结构

```go
// gdb_model_cache.go:17
type CacheOption struct {
    Duration time.Duration // 缓存时间：< 0 清除，= 0 永不过期，> 0 指定 TTL
    Name     string        // 缓存名称（可选，用于手动管理）
    Force    bool          // 强制缓存空结果（防缓存穿透）
}
```

### 8.2 缓存行为说明

| Duration 值 | 行为 |
|-------------|------|
| `> 0` | 设置缓存，指定 TTL |
| `= 0` | 永不过期 |
| `< 0` | 配合 Name 清除指定缓存 |

### 8.3 使用示例

```go
// 查询结果缓存 60 秒
result, err := db.Model("user").
    Cache(gdb.CacheOption{Duration: time.Minute}).
    Where("status", 1).
    All()

// 命名缓存 + 清除
// 写入缓存
result, _ := db.Model("user").
    Cache(gdb.CacheOption{
        Duration: time.Hour,
        Name:     "user_list",
    }).All()

// 写操作后清除缓存
_, err := db.Model("user").
    Cache(gdb.CacheOption{
        Duration: -1,
        Name:     "user_list",
    }).Data(g.Map{"name": "new"}).Insert()

// 分页缓存（分别设置 count 和 data 的缓存）
result, count, err := db.Model("user").
    PageCache(
        gdb.CacheOption{Duration: time.Minute, Name: "user_count"},
        gdb.CacheOption{Duration: time.Minute, Name: "user_data"},
    ).
    Page(1, 10).
    AllAndCount(true)
```

### 8.4 注意事项

- 事务中的 SELECT 不走缓存（`m.tx != nil` 时跳过）
- 缓存键由 `group + schema + table + name + sql + args` 组合生成
- 空结果默认不缓存（`Force: false`），需设置 `Force: true` 防穿透

---

## 9. Hook 链

### 9.1 HookHandler 结构

```go
// gdb_model_hook.go:27
type HookHandler struct {
    Select HookFuncSelect
    Insert HookFuncInsert
    Update HookFuncUpdate
    Delete HookFuncDelete
}
```

### 9.2 Hook 函数签名

```go
type HookFuncSelect func(ctx context.Context, in *HookSelectInput) (Result, error)
type HookFuncInsert func(ctx context.Context, in *HookInsertInput) (sql.Result, error)
type HookFuncUpdate func(ctx context.Context, in *HookUpdateInput) (sql.Result, error)
type HookFuncDelete func(ctx context.Context, in *HookDeleteInput) (sql.Result, error)
```

### 9.3 HookInput 属性

| 属性 | 说明 |
|------|------|
| `Model` | 当前操作的 Model 对象 |
| `Table` | 目标表名（可修改） |
| `Schema` | 目标 schema（可修改） |
| `Sql` | SQL 语句 |
| `Args` | SQL 参数 |
| `Data` | 数据（Insert/Update） |
| `Condition` | WHERE 条件（Update/Delete） |
| `SelectType` | SELECT 类型（Default/Count/Value/Array） |

### 9.4 Hook 执行流程

每个 HookInput 都有 `Next(ctx)` 方法，形成链式调用：

```
Hook 函数调用 → 检查分库分表 → 调用自定义 handler → 调用底层 DoSelect/DoInsert/DoUpdate/DoDelete
```

### 9.5 使用示例

```go
type AuditHook struct{}

func (h *AuditHook) Select(ctx context.Context, in *gdb.HookSelectInput) (gdb.Result, error) {
    log.Printf("SELECT query: %s", in.Sql)
    return in.Next(ctx)
}

func (h *AuditHook) Insert(ctx context.Context, in *gdb.HookInsertInput) (sql.Result, error) {
    log.Printf("INSERT into %s: %v", in.Table, in.Data)
    return in.Next(ctx)
}

// 使用
result, err := db.Model("user").
    Hook(gdb.HookHandler{
        Select: (&AuditHook{}).Select,
        Insert: (&AuditHook{}).Insert,
    }).
    All()
```

---

## 10. 软删除支持

### 10.1 默认字段名

```go
// gdb_model_soft_time.go:77
var (
    createdFieldNames = []string{"created_at", "create_at"}
    updatedFieldNames = []string{"updated_at", "update_at"}
    deletedFieldNames = []string{"deleted_at", "delete_at"}
)
```

### 10.2 SoftTimeType 时间类型

```go
type SoftTimeType int

const (
    SoftTimeTypeAuto           SoftTimeType = 0 // 自动检测
    SoftTimeTypeTime           SoftTimeType = 1 // datetime
    SoftTimeTypeTimestamp      SoftTimeType = 2 // Unix 秒
    SoftTimeTypeTimestampMilli SoftTimeType = 3 // Unix 毫秒
    SoftTimeTypeTimestampMicro SoftTimeType = 4 // Unix 微秒
    SoftTimeTypeTimestampNano  SoftTimeType = 5 // Unix 纳秒
)
```

### 10.3 自动行为

| 操作 | 自动填充字段 |
|------|-------------|
| INSERT | `created_at` / `updated_at` |
| UPDATE | `updated_at` |
| DELETE | `deleted_at`（UPDATE 操作，非真删除） |
| SELECT | 自动添加 `WHERE deleted_at IS NULL` 或 `deleted_at=0` |

### 10.4 配置

```go
// 通过 ConfigNode 自定义字段名
gdb.SetConfig(gdb.Config{
    "default": gdb.ConfigGroup{{
        Type:      "mysql",
        Link:      "mysql:root:123456@tcp(127.0.0.1:3306)/test",
        CreatedAt: "created_time",  // 自定义创建时间字段
        UpdatedAt: "updated_time",  // 自定义更新时间字段
        DeletedAt: "deleted_time",  // 自定义删除时间字段
    }},
})
```

### 10.5 Unscoped 跳过软删除

```go
// 查询包含已删除的记录
var users []User
db.Model("user").Unscoped().Scan(&users)

// 真删除（物理删除）
db.Model("user").Unscoped().Where("id", 1).Delete()
```

### 10.6 SoftTimeOption 自定义时间类型

```go
db.Model("user").
    SoftTime(gdb.SoftTimeOption{SoftTimeType: gdb.SoftTimeTypeTimestamp}).
    Where("id", 1).
    One()
```

---

## 11. 分库分表 / Sharding

### 11.1 配置结构

```go
// gdb_model_sharding.go:21
type ShardingConfig struct {
    Table  ShardingTableConfig   // 分表配置
    Schema ShardingSchemaConfig  // 分库配置
}

type ShardingTableConfig struct {
    Enable bool          // 启用分表
    Prefix string        // 表名前缀，如 "user_"
    Rule   ShardingRule  // 分表规则
}

type ShardingSchemaConfig struct {
    Enable bool          // 启用分库
    Prefix string        // 库名前缀，如 "db_"
    Rule   ShardingRule  // 分库规则
}
```

### 11.2 ShardingRule 接口

```go
type ShardingRule interface {
    SchemaName(ctx, config, value) (string, error)
    TableName(ctx, config, value) (string, error)
}
```

### 11.3 DefaultShardingRule（取模策略）

```go
type DefaultShardingRule struct {
    SchemaCount int  // 分库数量
    TableCount  int  // 分表数量
}

// 路由算法：FNV-1a 哈希 → 取模
// tableIndex = hash(shardingValue) % TableCount
// schemaIndex = hash(shardingValue) % SchemaCount
```

### 11.4 使用示例

```go
rule := &gdb.DefaultShardingRule{
    SchemaCount: 4,
    TableCount:  8,
}

result, err := db.Model("user").
    Sharding(gdb.ShardingConfig{
        Table: gdb.ShardingTableConfig{
            Enable: true,
            Prefix: "user_",
            Rule:   rule,
        },
        Schema: gdb.ShardingSchemaConfig{
            Enable: true,
            Prefix: "db_",
            Rule:   rule,
        },
    }).
    ShardingValue(12345).  // 分片键值
    Data(g.Map{"name": "john"}).
    Insert()
```

---

## 12. With 关联查询

### 12.1 结构体 Tag 定义

```go
type User struct {
    gmeta.Meta `orm:"table:user"`
    Id         int           `json:"id"`
    Name       string        `json:"name"`
    Detail     *UserDetail   `orm:"with:uid=id"`           // 一对一
    Scores     []*UserScore  `orm:"with:uid=id"`           // 一对多
}

type UserDetail struct {
    gmeta.Meta `orm:"table:user_detail"`
    Uid  int    `json:"uid"`
    Info string `json:"info"`
}

type UserScore struct {
    gmeta.Meta `orm:"table:user_score"`
    Uid   int     `json:"uid"`
    Score float64 `json:"score"`
}
```

`orm` Tag 支持：

| Tag | 说明 | 示例 |
|-----|------|------|
| `with` | 关联映射 `源字段=目标字段` | `orm:"with:uid=id"` |
| `with:where` | 关联查询条件 | `orm:"with:uid=id,where:status=1"` |
| `with:order` | 关联查询排序 | `orm:"with:uid=id,order:score desc"` |
| `with:unscoped` | 跳过软删除 | `orm:"with:uid=id,unscoped:true"` |

### 12.2 使用方式

```go
// 方式一：指定关联对象
var user User
db.With(UserDetail{}).With(UserScore{}).
    Where("id", 1).
    Scan(&user)
// user.Detail 和 user.Scores 自动填充

// 方式二：WithAll 自动加载所有带 with tag 的字段
var user User
db.Model("user").WithAll().Where("id", 1).Scan(&user)

// 方式三：批量关联查询
var users []User
db.Model("user").With(UserDetail{}).Scan(&users)
// 内部自动使用 ScanList 批量查询关联数据

// 嵌套 With（递归关联）
type User struct {
    Detail *UserDetail `orm:"with:uid=id"`
}
type UserDetail struct {
    Address *Address `orm:"with:detail_id=id"`
}
```

---

## 13. 行锁（Lock）

### 13.1 预定义常量

```go
// gdb_model_lock.go
const (
    LockForUpdate           = "FOR UPDATE"
    LockForUpdateSkipLocked = "FOR UPDATE SKIP LOCKED"
    LockInShareMode         = "LOCK IN SHARE MODE"
    LockForShare            = "FOR SHARE"
    LockForUpdateNowait     = "FOR UPDATE NOWAIT"
    // PostgreSQL 专用
    LockForNoKeyUpdate           = "FOR NO KEY UPDATE"
    LockForKeyShare              = "FOR KEY SHARE"
    // SQL Server 专用
    LockWithUpdLock = "WITH (UPDLOCK)"
    LockWithNoLock  = "WITH (NOLOCK)"
    // ...
)
```

### 13.2 使用示例

```go
// FOR UPDATE
db.Model("user").Lock(gdb.LockForUpdate).Where("id", 1).One()

// FOR UPDATE SKIP LOCKED（跳过锁定行）
db.Model("user").LockUpdateSkipLocked().Where("status", "active").All()

// 快捷方法
db.Model("user").LockUpdate().Where("id", 1).One()       // FOR UPDATE
db.Model("user").LockShared().Where("id", 1).One()        // LOCK IN SHARE MODE
```

---

## 14. gredis 适配器架构

### 14.1 架构概览

```
Redis (struct)
 ├── localAdapter (Adapter)           — 适配器实例
 └── localGroup                        — 命令分组（内嵌 8 个 Group 接口）
       ├── localGroupGeneric   (IGroupGeneric)
       ├── localGroupHash      (IGroupHash)
       ├── localGroupList      (IGroupList)
       ├── localGroupPubSub    (IGroupPubSub)
       ├── localGroupScript    (IGroupScript)
       ├── localGroupSet       (IGroupSet)
       ├── localGroupSortedSet (IGroupSortedSet)
       └── localGroupString    (IGroupString)

Adapter (interface)
 ├── AdapterGroup   — 命令分组工厂
 └── AdapterOperation — 核心操作

Conn (interface)
 ├── ConnCommand    — 连接级操作（Subscribe/PSubscribe/Receive）
 └── Do/Close       — 连接级命令执行

Pipeliner (interface)
 ├── PipelinerOperation  — Do/Exec/Discard
 └── PipelinerGroup      — 6 个 Pipeline 命令分组
```

### 14.2 Adapter 接口

```go
// gredis_adapter.go:16
type Adapter interface {
    AdapterGroup
    AdapterOperation
}

type AdapterGroup interface {
    GroupGeneric()   IGroupGeneric
    GroupHash()      IGroupHash
    GroupList()      IGroupList
    GroupPubSub()    IGroupPubSub
    GroupScript()    IGroupScript
    GroupSet()       IGroupSet
    GroupSortedSet() IGroupSortedSet
    GroupString()    IGroupString
}

type AdapterOperation interface {
    Do(ctx, command, args...) (*gvar.Var, error)
    Conn(ctx) (Conn, error)
    Close(ctx) error
    Client() RedisRawClient
    Pipeline(ctx) Pipeliner
    TxPipeline(ctx) Pipeliner
    Watch(ctx, fn, keys...) error
}
```

### 14.3 Conn 接口

```go
type Conn interface {
    ConnCommand
    Do(ctx, command, args...) (*gvar.Var, error)
    Close(ctx) error
}

type ConnCommand interface {
    Subscribe(ctx, channel, channels...) ([]*Subscription, error)
    PSubscribe(ctx, pattern, patterns...) ([]*Subscription, error)
    ReceiveMessage(ctx) (*Message, error)
    Receive(ctx) (*gvar.Var, error)
}
```

### 14.4 Config 结构

```go
// gredis_config.go:22
type Config struct {
    Address         string        // 地址，多节点逗号分隔
    Db              int           // 数据库编号
    User            string        // AUTH 用户名
    Pass            string        // AUTH 密码
    SentinelUser    string        // Sentinel 用户名
    SentinelPass    string        // Sentinel 密码
    MinIdle         int           // 最小空闲连接
    MaxIdle         int           // 最大空闲连接（默认 10）
    MaxActive       int           // 最大活跃连接（默认无限制）
    MaxConnLifetime time.Duration // 连接最大生命周期
    IdleTimeout     time.Duration // 空闲超时
    WaitTimeout     time.Duration // 等待超时
    DialTimeout     time.Duration // 拨号超时
    ReadTimeout     time.Duration // 读超时
    WriteTimeout    time.Duration // 写超时
    MasterName      string        // Sentinel 主节点名
    TLS             bool          // 启用 TLS
    TLSSkipVerify   bool          // 跳过 TLS 验证
    TLSConfig       *tls.Config   // TLS 配置
    SlaveOnly       bool          // 只读从节点
    Cluster         bool          // 集群模式
    Protocol        int           // RESP 协议版本（2 或 3，默认 3）
}
```

### 14.5 Pipeliner / Tx / Cmd

```go
// Cmd — Pipeline 命令的未来结果容器
type Cmd struct {
    val *gvar.Var
    err error
}
func (c *Cmd) Result() (*gvar.Var, error)
func (c *Cmd) Val() *gvar.Var
func (c *Cmd) Err() error

// Pipeliner — Pipeline 接口
type Pipeliner interface {
    PipelinerOperation
    PipelinerGroup
}

// Tx — 事务上下文（内嵌 Pipeliner）
type Tx interface {
    Pipeliner
}
```

### 14.6 命令分组接口

#### IGroupString

```go
type IGroupString interface {
    Set(ctx, key, value, option...) (*gvar.Var, error)
    SetNX(ctx, key, value) (bool, error)
    SetEX(ctx, key, value, ttl) error
    Get(ctx, key) (*gvar.Var, error)
    GetDel(ctx, key) (*gvar.Var, error)
    GetEX(ctx, key, option...) (*gvar.Var, error)
    GetSet(ctx, key, value) (*gvar.Var, error)
    StrLen(ctx, key) (int64, error)
    Append(ctx, key, value) (int64, error)
    SetRange(ctx, key, offset, value) (int64, error)
    GetRange(ctx, key, start, end) (string, error)
    Incr(ctx, key) (int64, error)
    IncrBy(ctx, key, increment) (int64, error)
    IncrByFloat(ctx, key, increment) (float64, error)
    Decr(ctx, key) (int64, error)
    DecrBy(ctx, key, decrement) (int64, error)
    MSet(ctx, keyValueMap) error
    MSetNX(ctx, keyValueMap) (bool, error)
    MGet(ctx, keys...) (map[string]*gvar.Var, error)
}
```

其他 Group（Hash/List/Set/SortedSet/Generic/PubSub/Script）接口定义在对应的 `gredis_redis_group_*.go` 文件中，结构类似。

### 14.7 使用示例

```go
// 基本使用
redis, _ := gredis.New(&gredis.Config{
    Address: "127.0.0.1:6379",
    Db:      0,
})

// String 操作
redis.GroupString().Set(ctx, "key", "value")
val, _ := redis.GroupString().Get(ctx, "key")

// Hash 操作
redis.GroupHash().HSet(ctx, "user:1", map[string]any{"name": "john", "age": 30})
val, _ := redis.GroupHash().HGet(ctx, "user:1", "name")
all, _ := redis.GroupHash().HGetAll(ctx, "user:1")

// Pipeline
pipe := redis.Pipeline(ctx)
cmd1 := pipe.PipelineGroupString().Set(ctx, "k1", "v1")
cmd2 := pipe.PipelineGroupString().Get(ctx, "k2")
pipe.Exec(ctx)
val1, _ := cmd1.Result()
val2, _ := cmd2.Result()

// TxPipeline（MULTI/EXEC）
txPipe := redis.TxPipeline(ctx)
cmd := txPipe.PipelineGroupString().Incr(ctx, "counter")
txPipe.Exec(ctx)
count, _ := cmd.Result()

// Watch
err := redis.Watch(ctx, func(tx gredis.Tx) error {
    val, _ := tx.PipelineGroupString().Get(ctx, "key").Result()
    tx.PipelineGroupString().Set(ctx, "key", val.Int()+1)
    return tx.Exec(ctx)
}, "key")

// 单例模式
redis := gredis.Instance("default")
```

### 14.8 适配器注册

```go
// contrib/nosql/redis 中注册适配器
import "github.com/gogf/gf/contrib/nosql/redis/v2"

func init() {
    gredis.RegisterAdapterFunc(func(config *gredis.Config) gredis.Adapter {
        return redis.New(config)
    })
}
```

---

## 15. 与其他模块的依赖关系

### 15.1 gdb 内部依赖

```
database/gdb
 ├── container/gvar        — 字段值类型 Value = *gvar.Var
 ├── container/gmap        — 连接池缓存、实例管理、类型映射
 ├── container/garray      — CatchSQL 管理
 ├── container/gset        — 字段去重
 ├── container/gtype       — 并发安全标志位（debug）
 ├── errors/gcode          — 错误码
 ├── errors/gerror         — 带堆栈的错误
 ├── os/gcache             — SQL 结果缓存、内部缓存
 ├── os/glog               — 日志输出
 ├── os/gctx               — 上下文键管理
 ├── os/gcmd               — 命令行参数（dryrun 开关）
 ├── os/gtime              — 软删除时间戳
 ├── text/gregex           — SQL 解析正则
 ├── text/gstr             — 字符串处理
 ├── util/gconv            — 通用类型转换
 ├── util/grand            — 随机数（负载均衡权重选择）
 ├── util/gutil            — 通用工具
 ├── internal/intlog       — 内部日志
 ├── internal/reflection   — 反射工具
 ├── internal/utils        — 内部工具函数
 ├── encoding/gjson        — Result.JSON() / XML()
 └── os/gstructs           — With 关联查询反射
```

### 15.2 gredis 内部依赖

```
database/gredis
 ├── container/gvar        — 返回值类型
 ├── container/gmap        — 配置管理、实例管理
 ├── errors/gcode          — 错误码
 ├── errors/gerror         — 带堆栈的错误
 └── util/gconv            — 配置解析
```

### 15.3 外部依赖关系

```
gogf/gf/v2/database/gdb           ← contrib/drivers/mysql   (驱动注册)
gogf/gf/v2/database/gdb           ← contrib/drivers/pgsql   (驱动注册)
gogf/gf/v2/database/gdb           ← contrib/drivers/*       (所有驱动)
gogf/gf/v2/database/gredis        ← contrib/nosql/redis     (适配器注册)
frame/gins                        ← database/gdb            (单例管理)
frame/g                           ← database/gdb + gredis   (门面 g.DB() / g.Redis())
```

---

## 附录 A：关键文件索引

| 文件 | 行数 | 职责 |
|------|------|------|
| `gdb.go` | 1174 | 核心类型定义（DB/TX/Core/Driver/ConfigNode/常量） |
| `gdb_core.go` | 841 | Core 实现（CRUD 快捷方法、Union、Ping、日志） |
| `gdb_core_transaction.go` | 295 | 事务管理（7种传播行为、WithTX/WithoutTX） |
| `gdb_core_txcore.go` | 437 | TXCore 实现（嵌套 Savepoint、CRUD） |
| `gdb_core_config.go` | 485 | 配置管理（ConfigNode、SetConfig、Link 解析） |
| `gdb_core_underlying.go` | 533 | 底层 SQL 执行（DoQuery/DoExec/DoCommit） |
| `gdb_core_structure.go` | 512 | SQL 构建（SELECT/INSERT/UPDATE/DELETE 语句生成） |
| `gdb_model.go` | 350 | Model 定义与基础方法 |
| `gdb_model_select.go` | 964 | 查询方法（All/One/Value/Count/Scan/Chunk） |
| `gdb_model_insert.go` | 469 | 插入方法（Insert/Replace/Save） |
| `gdb_model_update.go` | 139 | 更新方法 |
| `gdb_model_delete.go` | ~80 | 删除方法（含软删除处理） |
| `gdb_model_hook.go` | 309 | Hook 链实现 |
| `gdb_model_cache.go` | 172 | 查询缓存 |
| `gdb_model_soft_time.go` | 384 | 软删除时间管理 |
| `gdb_model_with.go` | 349 | With 关联查询 |
| `gdb_model_sharding.go` | 161 | 分库分表 |
| `gdb_model_lock.go` | 129 | 行锁常量与方法 |
| `gdb_model_builder_where.go` | 171 | WhereBuilder 条件构建 |
| `gdb_driver_default.go` | 46 | 默认驱动 |
| `gdb_driver_wrapper.go` | 31 | 驱动包装器 |
| `gdb_driver_wrapper_db.go` | 131 | DB 包装器（日志/追踪拦截） |
| `gdb_type_result.go` | 214 | Result 方法集 |
| `gdb_type_record.go` | ~100 | Record 方法集 |
| `gdb_func.go` | 1017 | 工具函数 |
| `gredis.go` | 78 | Redis 客户端创建与适配器注册 |
| `gredis_adapter.go` | 109 | Adapter/Conn 接口定义 |
| `gredis_redis.go` | 177 | Redis 结构体与方法 |
| `gredis_config.go` | 144 | 配置管理 |
| `gredis_cmd.go` | 55 | Cmd 类型定义 |
| `gredis_pipeline.go` | 196 | Pipeliner/Tx 接口定义 |

## 附录 B：InsertOption 枚举

```go
type InsertOption int

const (
    InsertOptionDefault InsertOption = iota  // INSERT
    InsertOptionReplace                      // REPLACE
    InsertOptionSave                         // INSERT ON DUPLICATE KEY UPDATE
    InsertOptionIgnore                       // INSERT IGNORE
)
```

## 附录 C：SqlType 枚举

```go
type SqlType string

const (
    SqlTypeBegin               SqlType = "DB.Begin"
    SqlTypeTXCommit            SqlType = "TX.Commit"
    SqlTypeTXRollback          SqlType = "TX.Rollback"
    SqlTypeExecContext         SqlType = "DB.ExecContext"
    SqlTypeQueryContext        SqlType = "DB.QueryContext"
    SqlTypePrepareContext      SqlType = "DB.PrepareContext"
    SqlTypeStmtExecContext     SqlType = "DB.Statement.ExecContext"
    SqlTypeStmtQueryContext    SqlType = "DB.Statement.QueryContext"
    SqlTypeStmtQueryRowContext SqlType = "DB.Statement.QueryRowContext"
)
```

## 附录 D：LocalType 字段类型映射

```go
type LocalType string

const (
    LocalTypeString       LocalType = "string"
    LocalTypeTime         LocalType = "time"
    LocalTypeDate         LocalType = "date"
    LocalTypeDatetime     LocalType = "datetime"
    LocalTypeInt          LocalType = "int"
    LocalTypeInt64        LocalType = "int64"
    LocalTypeUint64       LocalType = "uint64"
    LocalTypeFloat32      LocalType = "float32"
    LocalTypeFloat64      LocalType = "float64"
    LocalTypeBool         LocalType = "bool"
    LocalTypeBytes        LocalType = "[]byte"
    LocalTypeJson         LocalType = "json"
    LocalTypeJsonb        LocalType = "jsonb"
    LocalTypeUUID         LocalType = "uuid.UUID"
    // ... 更多类型见 gdb.go:791
)
```

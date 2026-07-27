# 第五部分：工具与基础设施层（util/、encoding/、container/、errors/、text/、crypto/、test/）

## 目录

- [1. util/gconv — 万能类型转换（框架基石）](#1-utilgconv--万能类型转换框架基石)
- [2. util/gvalid — 数据校验引擎](#2-utilgvalid--数据校验引擎)
- [3. util/gutil — 通用工具函数](#3-utilgutil--通用工具函数)
- [4. util/grand — 高性能随机数](#4-utilgrand--高性能随机数)
- [5. util/guid — 全局唯一 ID](#5-utilguid--全局唯一-id)
- [6. util/gtag — 结构体标签常量](#6-utilgtag--结构体标签常量)
- [7. util/gmeta — 结构体元数据](#7-utilgmeta--结构体元数据)
- [8. util/gpage — 分页](#8-utilgpage--分页)
- [9. util/gmode — 运行模式](#9-utilgmode--运行模式)
- [10. encoding/gjson — JSON 增强](#10-encodinggjson--json-增强)
- [11. encoding 多格式解析（gyaml/gxml/gtoml/gini/gproperties）](#11-encoding-多格式解析)
- [12. encoding/gcompress — 压缩](#12-encodinggcompress--压缩)
- [13. encoding/gcharset — 字符集转换](#13-encodinggcharset--字符集转换)
- [14. encoding/ghash — 哈希函数](#14-encodingghash--哈希函数)
- [15. encoding/ghtml、gurl、gbase64、gbinary](#15-encodingghtmlgurlgbase64gbinary)
- [16. container/garray — 并发安全数组](#16-containergarray--并发安全数组)
- [17. container/gmap — 并发安全映射](#17-containergmap--并发安全映射)
- [18. container/gset — 并发安全集合](#18-containergset--并发安全集合)
- [19. container/glist — 双向链表](#19-containerglist--双向链表)
- [20. container/gtree — 树容器](#20-containergtree--树容器)
- [21. container/gqueue、gring — 队列与环形缓冲](#21-containergqueuegring--队列与环形缓冲)
- [22. container/gpool — 对象池](#22-containergpool--对象池)
- [23. container/gtype — 原子类型包装](#23-containergtype--原子类型包装)
- [24. container/gvar — 泛型变量](#24-containergvar--泛型变量)
- [25. errors/gerror — 带堆栈的错误链](#25-errorsgerror--带堆栈的错误链)
- [26. errors/gcode — 错误码注册](#26-errorsgcode--错误码注册)
- [27. text/gstr — 字符串操作](#27-textgstr--字符串操作)
- [28. text/gregex — 正则表达式封装](#28-textgregex--正则表达式封装)
- [29. crypto — 加密哈希](#29-crypto--加密哈希)
- [30. test/gtest — 测试辅助框架](#30-testgtest--测试辅助框架)

---

## 1. util/gconv — 万能类型转换（框架基石）

### 1.1 包概述

`gconv` 是 GoFrame 框架的基石包，实现了强大的任意类型转换功能。几乎所有公共 API 接受 `any` 参数时都依赖 `gconv` 进行类型强制。设计理念是：**让开发者无需关心类型兼容性，由框架自动完成转换**。

源码位置：`util/gconv/`

包注释见 `util/gconv/gconv.go:7`：
> Package gconv implements powerful and convenient converting functionality for any types of variables.

关键设计约束（`util/gconv/gconv.go:9`）：
> This package should keep much fewer dependencies with other packages.

### 1.2 架构分层

`gconv` 采用三层架构：

```
gconv/                        — 公共 API 层（薄封装，委托给内部 converter）
├── gconv.go                  — Converter 接口定义 + 类型别名 + 全局 defaultConverter
├── gconv_basic.go            — 基础类型快捷函数（String/Bool/Byte 等）
├── gconv_convert.go          — Convert / ConvertWithRefer
├── gconv_int.go / gconv_uint.go / gconv_float.go — 数值类型公共函数
├── gconv_map.go / gconv_maps.go — Map 转换公共函数
├── gconv_struct.go / gconv_structs.go — Struct 转换公共函数
├── gconv_scan.go             — Scan / ScanWithOptions（统一入口）
├── gconv_slice_*.go          — 各类型切片转换
├── gconv_time.go / gconv_ptr.go / gconv_unsafe.go
├── internal/converter/       — 核心实现层
│   ├── converter.go          — Converter 结构体 + 注册机制 + 内置转换器
│   ├── converter_int.go      — 整数转换策略
│   ├── converter_string.go   — 字符串转换策略
│   ├── converter_struct.go   — 结构体转换核心（690 行）
│   ├── converter_map.go      — Map 转换核心（654 行）
│   └── ...
├── internal/structcache/     — 结构体字段缓存（性能优化核心）
│   ├── structcache.go        — Converter + AnyConvertFunc 定义
│   ├── structcache_cached.go — 结构体解析缓存逻辑
│   ├── structcache_cached_struct_info.go — CachedStructInfo
│   ├── structcache_cached_field_info.go  — CachedFieldInfo / CachedFieldInfoBase
│   └── structcache_pool.go   — 对象池
└── internal/localinterface/  — 轻量接口定义（避免循环导入）
    └── localinterface.go     — IString, IInt64, IUnmarshalValue 等
```

### 1.3 核心类型和接口

#### Converter 接口体系（`util/gconv/gconv.go:23-114`）

```go
// 顶层组合接口
type Converter interface {
    ConverterForConvert
    ConverterForRegister
    ConverterForInt
    ConverterForUint
    ConverterForTime
    ConverterForFloat
    ConverterForMap
    ConverterForSlice
    ConverterForStruct
    ConverterForBasic
}
```

每个子接口按转换目标类型分组，例如 `ConverterForInt`（`util/gconv/gconv.go:52-58`）定义了 `Int/Int8/Int16/Int32/Int64` 方法。

#### 内部 Converter 结构体（`util/gconv/internal/converter/converter.go:38-41`）

```go
type Converter struct {
    internalConverter    *structcache.Converter         // 结构体缓存 + AnyConvertFunc 注册
    typeConverterFuncMap map[converterInType]map[converterOutType]converterFunc // 自定义类型转换器
}
```

#### 全局默认转换器（`util/gconv/gconv.go:148`）

```go
var defaultConverter = converter.NewConverter()
```

所有公共函数（`gconv.String()`、`gconv.Int()` 等）最终都委托给 `defaultConverter`。

### 1.4 关键方法列表

| 公共函数 | 内部委托 | 用途 |
|---------|---------|------|
| `String(any)` | `defaultConverter.String()` | 任意类型转字符串 |
| `Bool(any)` | `defaultConverter.Bool()` | 任意类型转布尔 |
| `Int/Int64(any)` | `defaultConverter.Int/Int64()` | 任意类型转整数 |
| `Float64(any)` | `defaultConverter.Float64()` | 任意类型转浮点 |
| `Bytes(any)` | `defaultConverter.Bytes()` | 任意类型转字节切片 |
| `Map(any, ...MapOption)` | `defaultConverter.Map()` | 转 `map[string]any` |
| `Struct(params, pointer, ...)` | → `Scan()` | 参数映射到结构体 |
| `Scan(srcValue, dstPointer, ...)` | `defaultConverter.Scan()` | 统一转换入口 |
| `SliceAny/SliceInt/SliceStr(any)` | `defaultConverter.Slice*()` | 切片转换 |
| `Convert(fromValue, toTypeName)` | `defaultConverter.ConvertWithTypeName()` | 按类型名转换 |
| `RegisterTypeConverterFunc(fn)` | `defaultConverter.RegisterTypeConverterFunc()` | 注册自定义转换器 |

### 1.5 转换策略深度分析

#### 1.5.1 整数转换策略（`util/gconv/internal/converter/converter_int.go:70-158`）

`Int64` 是所有整数转换的基础，采用多级优先级策略：

1. **nil 检查**（`:71`）：`empty.IsNil()` → 返回 0
2. **类型断言快速路径**（`:74`）：`int64` 直接返回
3. **reflect.Kind 分发**（`:78-152`）：
   - 整数族 → `rv.Int()`
   - 无符号整数族 → `int64(rv.Uint())`
   - 浮点族 → `int64(rv.Float())`
   - 布尔 → `true=1, false=0`
   - 指针 → 递归解引用
   - `[]byte` → `gbinary.DecodeToInt64()`
   - 字符串 → 多策略解析（见下）
4. **字符串解析子策略**（`:106-147`）：
   - 符号处理（`+/-`）
   - 十六进制（`0x/0X` 前缀）→ `strconv.ParseInt(s, 16, 64)`
   - 十进制 → `strconv.ParseInt(s, 10, 64)`
   - 回退到 Float64 解析再截断

#### 1.5.2 字符串转换策略（`util/gconv/internal/converter/converter_string.go:23-134`）

采用 `type switch` + reflect 回退：

1. **基础类型直接转换**（`:28-79`）：`int/uint/float/bool/string/[]byte/time.Time/gtime.Time`
2. **接口断言**（`:81-90`）：优先检查 `IString` 和 `IError` 接口
3. **reflect.Kind 回退**（`:96-125`）：处理通过反射访问的自定义类型
4. **JSON 序列化兜底**（`:127-133`）：`json.Marshal(value)` 作为最终手段

#### 1.5.3 结构体转换核心（`util/gconv/internal/converter/converter_struct.go:52-241`）

`Struct` 方法是框架最复杂的转换逻辑，处理流程：

```
1. nil/参数检查
2. JSON 内容快速检测（doConvertWithJSONCheck）
3. 同类型直接赋值（doConvertWithTypeCheck）— 性能优化
4. 自定义转换器检查（callCustomConverter）
5. 标准接口检测（bindVarToReflectValueWithInterfaceCheck）
   — IUnmarshalValue > IUnmarshalText > IUnmarshalJSON > ISet
6. 参数转换为 map[string]any]
7. 获取结构体缓存（GetCachedStructInfo）★ 性能核心
8. 自定义映射优先（ParamKeyToAttrMap）
9. 循环字段信息匹配（bindStructWithLoopFieldInfos）
   — 精确匹配 → 模糊匹配（大小写不敏感、去符号）
```

字段匹配优先级（通过 `gtag.StructTagPriority`，`util/gtag/gtag.go:58-60`）：
```go
var StructTagPriority = []string{
    GConv, Param, GConvShort, ParamShort, Json,
}
// 即: "gconv" > "param" > "c" > "p" > "json" > 字段名
```

### 1.6 性能优化机制

#### 1.6.1 结构体字段缓存（structcache）

**CachedFieldInfoBase**（`util/gconv/internal/structcache/structcache_cached_field_info.go:37-80`）：

```go
type CachedFieldInfoBase struct {
    FieldIndexes            []int           // 全局索引（支持嵌套）
    PriorityTagAndFieldName []string         // 所有标签名 + 字段名
    IsCommonInterface       bool             // 是否实现通用接口
    HasCustomConvert        bool             // 是否有自定义转换器
    StructField             reflect.StructField
    OtherSameNameField      []*CachedFieldInfo // 同名字段
    ConvertFunc             AnyConvertFunc   // 快速赋值函数
    LastFuzzyKey            atomic.Value     // 模糊匹配缓存
}
```

缓存通过 `sync.Map` 存储（`structcache.go:28`），首次解析结构体后永久缓存，避免重复 reflect 操作。

#### 1.6.2 快速路径

- **类型断言快速返回**：`converter_int.go:74` 的 `anyInput.(int64)` 直接返回
- **同类型直接赋值**：`converter_struct.go:120` 的 `doConvertWithTypeCheck`
- **内置类型 AnyConvertFunc**：`converter.go:136-179` 为所有基础类型注册了批量转换函数
- **对象池复用**：`structcache_pool.go` 通过 `sync.Pool` 复用临时 map

### 1.7 自定义类型转换器注册

```go
// 注册模式: func(T1) (T2, error)
// T1 不能是指针，T2 必须是指针
gconv.RegisterTypeConverterFunc(func(src string) (*MyType, error) {
    return &MyType{Value: src}, nil
})
```

验证逻辑见 `converter.go:72-127`：检查函数签名、输入非指针、输出为指针、防止重复注册。

### 1.8 使用规范和代码示例

```go
package main

import (
    "context"
    "github.com/gogf/gf/v2/frame/g"
    "github.com/gogf/gf/v2/util/gconv"
)

type User struct {
    Id       int    `json:"id" c:"user_id"`  // c 标签优先级高于 json
    Name     string `json:"name"`
    Age      int    `json:"age"`
    Email    string `json:"email"`
}

func main() {
    // 1. 基本类型转换
    g.Dump(gconv.Int("123"))      // 123
    g.Dump(gconv.String(456))     // "456"
    g.Dump(gconv.Bool("true"))    // true
    g.Dump(gconv.Float64("3.14")) // 3.14

    // 2. Map 转结构体（Struct 是 Scan 的别名）
    params := map[string]any{
        "user_id": 1001,
        "name":    "John",
        "age":     "30",    // 字符串自动转 int
    }
    user := User{}
    gconv.Struct(params, &user)
    g.Dump(user) // {Id:1001, Name:"John", Age:30, Email:""}

    // 3. ScanWithOptions — OmitEmpty / OmitNil
    dst := &User{Name: "Alice", Email: "alice@example.com"}
    src := map[string]any{"Name": "", "Email": nil}
    gconv.ScanWithOptions(src, dst, gconv.ScanOption{
        OmitEmpty: true,
        OmitNil:   true,
    })
    // dst 仍为 {Name:"Alice", Email:"alice@example.com"}

    // 4. StructTag — 指定优先标签
    data := map[string]any{"user_id": 2002}
    user2 := User{}
    gconv.StructTag(data, &user2, "c")
    g.Dump(user2) // Id: 2002

    // 5. 自定义转换器
    gconv.RegisterTypeConverterFunc(func(s string) (*User, error) {
        return &User{Name: s}, nil
    })

    // 6. 切片转换
    g.Dump(gconv.SliceInt("1,2,3")) // [1,2,3]
}
```

### 1.9 与其他模块的依赖关系

| 依赖方向 | 说明 |
|---------|------|
| → `os/gtime` | 时间类型转换 |
| → `encoding/gbinary` | `[]byte` ↔ 整数转换 |
| → `errors/gerror`, `errors/gcode` | 错误包装 |
| → `internal/json` | JSON 序列化兜底 |
| → `internal/empty` | nil/空值判断 |
| → `internal/utils` | RemoveSymbols 等工具 |
| → `util/gtag` | 结构体标签优先级常量 |

---

## 2. util/gvalid — 数据校验引擎

### 2.1 包概述

`gvalid` 实现了强大的数据校验功能，参考 Laravel 验证规则设计。支持 70+ 内置校验规则、自定义规则、链式调用、i18n 错误消息、联合校验等。

源码位置：`util/gvalid/`

### 2.2 核心类型

#### Validator（`util/gvalid/gvalid_validator.go:22-33`）

```go
type Validator struct {
    i18nManager                       *gi18n.Manager
    data                              any
    assoc                             any
    rules                             any
    messages                          any
    ruleFuncMap                       map[string]RuleFunc
    useAssocInsteadOfObjectAttributes bool
    bail                              bool
    foreach                           bool
    caseInsensitive                   bool
}
```

**不可变性设计**：所有链式方法（`Data()`、`Rules()`、`Bail()` 等）都通过 `Clone()` 创建新实例（`:88-93`），确保线程安全。

#### Error 接口（`util/gvalid/gvalid_error.go:19-31`）

```go
type Error interface {
    Code() gcode.Code
    Current() error
    Error() string
    FirstItem() (key string, messages map[string]error)
    FirstRule() (rule string, err error)
    FirstError() (err error)
    Items() (items []map[string]map[string]error)
    Map() map[string]error
    Maps() map[string]map[string]error
    String() string
    Strings() (errs []string)
}
```

#### validationError 实现（`util/gvalid/gvalid_error.go:34-40`）

```go
type validationError struct {
    code      gcode.Code
    rules     []fieldRule                          // 顺序规则（保持错误序列）
    errors    map[string]map[string]error          // field → rule → error
    firstKey  string
    firstItem map[string]error
}
```

#### RuleFunc 类型（`util/gvalid/gvalid_register.go:20`）

```go
type RuleFunc func(ctx context.Context, in RuleFuncInput) error
```

#### 内置规则接口（`util/gvalid/internal/builtin/builtin.go:19-28`）

```go
type Rule interface {
    Name() string
    Message() string
    Run(in RunInput) error
}
```

### 2.3 校验流程调用链

```
Validator.Run(ctx)
  ├── data.Kind == Map  → doCheckMap(ctx, data)
  ├── data.Kind == Struct → doCheckStruct(ctx, data)
  └── 其他                → doCheckValue(ctx, input)
```

#### doCheckValue 核心逻辑（`util/gvalid/gvalid_validator_check_value.go:36-224`）

```
1. 规则解析：按 "|" 分割为多个规则
2. 规则合法性检查：builtin.GetRule() + getCustomRuleFunc()
3. 特殊规则处理：regex/not-regex（合并含 ":" 的模式）
4. 装饰规则检测：bail / foreach / ci
5. 遍历每个规则：
   a. 解析 ruleKey + rulePattern（ruleRegex 正则）
   b. 获取错误消息（getErrorMessageByRule）
   c. 优先调用自定义规则（customRuleFunc）
   d. 回退到内置规则（builtinRule.Run）
   e. foreach：将值转为切片，逐元素校验
   f. bail：首个错误立即返回
6. 错误消息变量替换：{field} / {value} / {pattern}
7. 构造 validationError 返回
```

### 2.4 内置规则列表（70+）

所有规则定义在 `util/gvalid/internal/builtin/` 目录下，每个规则一个文件，通过 `init()` 自动注册。

| 类别 | 规则 |
|------|------|
| **必填** | `required`, `required-if`, `required-unless`, `required-with`, `required-without`, `required-if-all`, `required-with-all`, `required-without-all` |
| **比较** | `same`, `different`, `eq`, `not-eq`, `gt`, `gte`, `lt`, `lte` |
| **范围** | `in`, `not-in`, `between`, `min`, `max` |
| **长度** | `length`, `min-length`, `max-length`, `size` |
| **格式** | `regex`, `not-regex`, `date`, `date-format`, `datetime`, `email`, `url`, `domain`, `ip`, `ipv4`, `ipv6`, `mac` |
| **类型** | `integer`, `float`, `boolean`, `numeric`, `alpha`, `alpha-num`, `alpha-dash`, `json` |
| **业务** | `phone`, `telephone`, `postcode`, `resident-id`, `bank-card`, `qq`, `passport`, `password`, `password2`, `password3` |
| **装饰** | `bail`, `foreach`, `ci` |
| **其他** | `array`, `enums`, `before`, `after`, `before-equal`, `after-equal`, `uppercase`, `lowercase` |

### 2.5 自定义规则注册

#### 全局注册（`util/gvalid/gvalid_register.go:51-61`）

```go
gvalid.RegisterRule("my-rule", func(ctx context.Context, in gvalid.RuleFuncInput) error {
    val := in.Value.String()
    if val == "" {
        return errors.New("值不能为空")
    }
    return nil
})
```

#### 实例级注册（`util/gvalid/gvalid_validator.go:173-177`）

```go
validator.RuleFunc("custom-rule", func(ctx context.Context, in gvalid.RuleFuncInput) error {
    // 仅对当前 Validator 实例生效
    return nil
})
```

### 2.6 使用规范和代码示例

```go
package main

import (
    "context"
    "fmt"
    "github.com/gogf/gf/v2/util/gvalid"
)

// 1. 结构体标签校验
type RegisterReq struct {
    Name     string `v:"required|length:3,20#请输入用户名|用户名长度需要在3到20之间" json:"name"`
    Email    string `v:"required|email" json:"email"`
    Password string `v:"required|password3" json:"password"`
    Age      int    `v:"required|between:18,120" json:"age"`
}

func validateStruct() {
    req := RegisterReq{
        Name:     "ab",      // 太短
        Email:    "invalid", // 非邮箱
        Password: "123",     // 弱密码
        Age:      15,        // 未成年
    }
    err := gvalid.New().Data(req).Run(context.Background())
    if err != nil {
        // 获取所有错误
        fmt.Println(err.Strings())
        // 获取第一个错误
        fmt.Println(err.FirstError())
    }
}

// 2. 链式单值校验
func validateValue() {
    err := gvalid.New().
        Rules("required|between:1,100").
        Data(150).
        Run(context.Background())
    if err != nil {
        fmt.Println(err.FirstError()) // The Data value `150` must be between 1 and 100
    }
}

// 3. 联合校验（Assoc）
func validateWithAssoc() {
    data := map[string]any{
        "password":        "abc123",
        "password_confirm": "abc456",
    }
    err := gvalid.New().
        Data("abc123").
        Assoc(data).
        Rules("required|same:password_confirm").
        Run(context.Background())
    // 两个密码不一致
}

// 4. bail + foreach
func validateForeach() {
    err := gvalid.New().
        Bail().           // 首个错误即返回
        Foreach().        // 逐元素校验
        Rules("required|email").
        Data([]string{"a@b.com", "invalid", "c@d.com"}).
        Run(context.Background())
}

// 5. 自定义校验规则
func customRuleExample() {
    gvalid.RegisterRule("phone-cn", func(ctx context.Context, in gvalid.RuleFuncInput) error {
        phone := in.Value.String()
        if len(phone) != 11 {
            return fmt.Errorf("手机号 %s 格式不正确", phone)
        }
        return nil
    })

    err := gvalid.New().
        Rules("required|phone-cn").
        Data("13800138000").
        Run(context.Background())
    _ = err
}
```

### 2.7 与其他模块的依赖关系

| 依赖 | 说明 |
|------|------|
| → `i18n/gi18n` | 错误消息国际化 |
| → `container/gvar` | RuleFuncInput 中的 Value/Data 类型 |
| → `container/gset` | 规则去重 |
| → `encoding/gjson` | JSON 数据校验 |
| → `errors/gerror`, `errors/gcode` | 错误包装 |
| → `text/gstr`, `text/gregex` | 字符串匹配和正则 |
| → `util/gconv` | 类型转换 |
| → `util/gtag` | 标签常量（Valid, Param, NoValidation） |
| → `util/gmeta` | 结构体元数据 |
| → `os/gstructs` | 结构体字段反射 |

---

## 3. util/gutil — 通用工具函数

### 3.1 包概述

提供通用辅助函数，包括 map/struct 的 Keys/Values 提取、SliceDefault、MapCopy、Dump、TryCatch、反射工具等。

源码位置：`util/gutil/`

### 3.2 关键方法列表

| 方法 | 文件 | 用途 |
|------|------|------|
| `Keys(mapOrStruct)` | `gutil.go:21` | 提取 map 或 struct 的键名 |
| `Values(mapOrStruct)` | `gutil.go:72` | 提取 map 或 struct 的值 |
| `MapCopy(dst, src)` | `gutil_map.go` | map 浅拷贝 |
| `MapContains(map, key)` | `gutil_map.go` | 检查 map 是否包含键 |
| `SliceContains(slice, value)` | `gutil_slice.go` | 检查切片是否包含值 |
| `Dump(values...)` | `gutil_dump.go` | 格式化打印变量 |
| `TryCatch(try, catch)` | `gutil_try_catch.go` | try-catch 模式 |
| `CopyValue(value)` | `gutil_copy.go` | 深拷贝 |
| `Comparator[T]()` | `gutil_comparator.go` | 通用比较器 |
| `OriginValueAndKind(value)` | `gutil_reflect.go` | 获取原始值和 Kind |

### 3.3 使用示例

```go
// 提取 struct 字段名
type User struct { Name string; Age int }
fields := gutil.Keys(&User{}) // ["Name", "Age"]

// Try-Catch
gutil.TryCatch(func(ctx context.Context) {
    panic("something wrong")
}, func(ctx context.Context, exception error) {
    log.Println("recovered:", exception)
})
```

---

## 4. util/grand — 高性能随机数

### 4.1 包概述

基于 `crypto/rand` 的高性能随机数生成器，通过异步 goroutine + buffer channel 实现高性能。

源码位置：`util/grand/`

### 4.2 核心设计：异步缓冲管道（`util/grand/grand_buffer.go`）

```go
var bufferChan = make(chan []byte, 10000)  // 缓冲大小 10000

func init() {
    go asyncProducingRandomBufferBytesLoop()
}

func asyncProducingRandomBufferBytesLoop() {
    for {
        buffer := make([]byte, 1024)
        rand.Read(buffer)
        for _, step = range []int{4} {
            for i := 0; i <= n-4; i += step {
                bufferChan <- buffer[i : i+4]
            }
        }
    }
}
```

系统随机数生成成本很高，通过预生成 + channel 缓冲大幅提升性能。

### 4.3 关键方法

| 方法 | 说明 |
|------|------|
| `Intn(max)` | `[0, max)` 随机整数（32位） |
| `N(min, max)` | `[min, max]` 随机整数（支持负数） |
| `B(n)` | 随机字节切片 |
| `S(n, symbols...)` | 随机字符串（字母+数字+可选符号） |
| `Str(s, n)` | 从指定字符集生成随机字符串 |

### 4.4 使用示例

```go
grand.N(1, 100)         // [1, 100] 随机数
grand.S(16)             // 16位随机字符串（字母+数字）
grand.S(8, true)        // 8位随机字符串（含特殊符号）
grand.B(32)             // 32字节随机数据
```

---

## 5. util/guid — 全局唯一 ID

### 5.1 包概述

提供简单高性能的全局唯一 ID 生成，格式为 32 字节字符串，由 MAC 地址哈希 + 进程 ID + 时间戳 + 序列号 + 随机数组成。

源码位置：`util/guid/guid.go`

### 5.2 ID 结构（`util/guid/guid.go:57-100`）

```
[MAC哈希7字节][进程ID4字节][时间戳13字节][序列号3字节][随机数5字节]
共 32 字符
```

- `sequenceMax = uint32(46655)` — 序列号最大值（"zzz"，36进制）
- `macAddrStr` — init() 中通过 `gipv4.GetMacArray()` + `ghash.SDBM` 生成
- `processIdStr` — `os.Getpid()` 36进制编码
- `sequence` — `gtype.Uint32` 原子递增

### 5.3 关键方法

| 方法 | 说明 |
|------|------|
| `S(data...)` | 生成 32 字节唯一字符串 |
| `New()` | 返回 `*guid.Guid` 对象 |

### 5.4 使用示例

```go
id := guid.S() // "0000000a3b2c5f7e8d9a0b1c2d3e4f5a"
// 带数据的唯一 ID（对数据做哈希）
dataId := guid.S([]byte("some data"))
```

---

## 6. util/gtag — 结构体标签常量

### 6.1 包概述

集中定义所有结构体标签常量，供 `gconv`、`gvalid`、`ghttp`、`goai` 等模块统一引用。

源码位置：`util/gtag/gtag.go`

注意（`:11`）：此包非并发安全，仅在启动阶段调用。

### 6.2 标签常量一览（`util/gtag/gtag.go:14-53`）

| 常量 | 值 | 用途 |
|------|------|------|
| `Default` / `DefaultShort` | `"default"` / `"d"` | HTTP 参数默认值 |
| `Param` / `ParamShort` | `"param"` / `"p"` | 参数名映射 |
| `Valid` / `ValidShort` | `"valid"` / `"v"` | 校验规则 |
| `NoValidation` | `"nv"` | 跳过校验 |
| `GConv` / `GConvShort` | `"gconv"` / `"c"` | 类型转换目标名 |
| `Json` | `"json"` | JSON 标签 |
| `ORM` | `"orm"` | ORM 标签 |
| `Path`, `Method`, `Domain` | — | 路由标签 |
| `Summary`, `Description` | — | OpenAPI 文档标签 |

### 6.3 StructTagPriority（`util/gtag/gtag.go:58-60`）

```go
var StructTagPriority = []string{
    GConv, Param, GConvShort, ParamShort, Json,
}
```

定义了 `gconv` 的默认标签优先级。

---

## 7. util/gmeta — 结构体元数据

### 7.1 包概述

通过在结构体中嵌入 `gmeta.Meta` 空结构体，利用 struct tag 存储元数据。

源码位置：`util/gmeta/gmeta.go`

### 7.2 核心设计

```go
type Meta struct{}  // 空结构体，零开销

const metaAttributeName = "Meta"
var metaType = reflect.TypeOf(Meta{})
```

`Data()` 方法（`:27-38`）通过反射查找 `Meta` 字段，解析其 struct tag 返回 `map[string]string`。

### 7.3 使用示例

```go
type User struct {
    gmeta.Meta `type:"entity" table:"user" description:"用户表"`
    Id   int    `json:"id"`
    Name string `json:"name"`
}

meta := gmeta.Data(&User{})
// map["type"]="entity", map["table"]="user", map["description"]="用户表"

table := gmeta.Get(&User{}, "table")
fmt.Println(table.String()) // "user"
```

---

## 8. util/gpage — 分页

### 8.1 包概述

提供 Web 分页 HTML 生成功能。

> **注意**：此包已标记为 Deprecated（`util/gpage/gpage.go:9`），建议在业务层处理分页 HTML，将在 3.0 版本移除。

### 8.2 Page 结构体（`util/gpage/gpage.go:26-42`）

```go
type Page struct {
    TotalSize      int
    TotalPage      int    // 自动计算
    CurrentPage    int    // >= 1
    UrlTemplate    string // URL 模板，如 "/user/list/{.page}"
    LinkStyle      string
    SpanStyle      string
    PageBarNum     int
    AjaxActionName string
}
```

### 8.3 使用示例

```go
page := gpage.New(1000, 20, 3, "/user/list/{.page}")
html := page.GetContent() // 生成分页 HTML
```

---

## 9. util/gmode — 运行模式

### 9.1 包概述

管理应用运行模式（开发/测试/预发布/生产），使用字符串而非整数便于配置。

源码位置：`util/gmode/gmode.go`

### 9.2 模式常量（`:18-25`）

```go
NOT_SET   = "not-set"
DEVELOP   = "develop"
TESTING   = "testing"
STAGING   = "staging"
PRODUCT   = "product"
```

### 9.3 自动检测逻辑（`Mode()` :58-74）

1. 如果已手动设置，直接返回
2. 检查环境变量/命令行参数 `gf.gmode`
3. 如果存在源码文件 → `DEVELOP`，否则 → `PRODUCT`

> **注意**（`:28`）：`currentMode` 非并发安全，应在程序启动时设置。

### 9.4 使用示例

```go
gmode.SetDevelop()     // 设置开发模式
if gmode.IsDevelop() {
    // 开发模式逻辑
}
gmode.SetProduct()     // 设置生产模式
```

---

## 10. encoding/gjson — JSON 增强

### 10.1 包概述

`gjson` 不仅仅处理 JSON，它是一个通用的配置数据容器，支持 JSON/XML/INI/YAML/TOML/Properties 格式的解析、操作和序列化。

源码位置：`encoding/gjson/`

### 10.2 核心类型

#### Json 结构体（`encoding/gjson/gjson.go:43-55`）

```go
type Json struct {
    mu rwmutex.RWMutex
    p  *any   // 层级数据根指针
    c  byte   // 分隔符（默认 '.'）
    vc bool   // Violence Check 模式
}
```

#### Options（`encoding/gjson/gjson.go:58-72`）

```go
type Options struct {
    Safe     bool         // 并发安全
    Tags     string       // 自定义优先标签
    Type     ContentType  // 内容类型
    StrNumber bool        // 数字解析为字符串
}
```

#### ContentType 常量（`:27-35`）

```go
ContentTypeJSON       = `json`
ContentTypeXML        = `xml`
ContentTypeIni        = `ini`
ContentTypeYaml       = `yaml`
ContentTypeYml        = `yml`
ContentTypeToml       = `toml`
ContentTypeProperties = `properties`
```

### 10.3 层级数据访问

通过 `.` 分隔符访问嵌套数据：

```go
json.Get("user.address.city")       // 嵌套路径
json.Get("list.10")                  // 数组索引
json.Get("array.0.name")             // 混合路径
```

支持两种查找模式（`gjson.go:377-386`）：

- **普通模式**（默认）：按 `.` 简单分割
- **Violence Check 模式**（`vc=true`）：当 key 本身包含 `.` 时启用

### 10.4 关键方法列表

| 方法 | 说明 |
|------|------|
| `New(value, safe...)` | 创建 Json 对象 |
| `NewContent(content, options...)` | 从内容创建 |
| `Load(path)` / `LoadContent()` | 从文件/内容加载 |
| `Get(pattern, def...)` | 按路径获取值（返回 `*gvar.Var`） |
| `GetJson(pattern)` | 获取子 Json 对象 |
| `Set(pattern, value)` | 设置值 |
| `Remove(pattern)` | 删除节点 |
| `GetVar(pattern)` | 获取 `*gvar.Var` |
| `Map()` / `Maps()` | 转为 map |
| `Array()` | 转为数组 |
| `ToJSON()` / `ToXML()` / `ToYAML()` / `ToTOML()` / `ToINI()` | 格式转换 |
| `Dump()` | 格式化打印 |
| `Merge(json)` | 合并 |
| `Contains(pattern)` | 检查路径是否存在 |
| `Len(pattern)` | 获取节点长度 |

### 10.5 使用示例

```go
// 从 JSON 字符串创建
j, _ := gjson.LoadContent(`{"name":"John","age":30,"address":{"city":"Beijing"}}`)
j.Get("name").String()            // "John"
j.Get("age").Int()                // 30
j.Get("address.city").String()    // "Beijing"

// 修改数据
j.Set("address.zipcode", "100000")
j.Remove("age")

// 格式转换
yaml, _ := j.ToYAML()
xml, _ := j.ToXML()

// 从 YAML 文件加载
j2, _ := gjson.Load("config.yaml")

// 合并
j.Merge(j2)
```

---

## 11. encoding 多格式解析

### 11.1 gyaml

封装 `gopkg.in/yaml.v3`，提供 YAML 编解码。

```go
data, err := gyaml.Encode(map[string]any{"key": "value"})
v, err := gyaml.Decode([]byte("key: value"))
v2, err := gyaml.DecodeToMap([]byte("key: value"))
```

### 11.2 gxml

封装标准库 `encoding/xml`，提供 XML ↔ map 互转（弥补标准库仅支持 struct 的不足）。

```go
m, err := gxml.Decode([]byte(`<root><name>John</name></root>`))
xmlBytes, err := gxml.Encode(map[string]any{"root": map[string]any{"name": "John"}})
```

### 11.3 gtoml

封装 `github.com/BurntSushi/toml`，提供 TOML 编解码。

```go
data, _ := gtoml.Encode(map[string]any{"title": "TOML"})
v, _ := gtoml.Decode([]byte("title = \"TOML\""))
```

### 11.4 gini

INI 格式解析，返回 `map[string]map[string]string`。

```go
data, err := gini.Encode(map[string]map[string]string{
    "database": {"host": "localhost", "port": "3306"},
})
config, err := gini.Decode(data)
```

### 11.5 gproperties

Java Properties 格式解析。

```go
data, err := gproperties.Encode(map[string]string{"key": "value"})
config, err := gproperties.Decode(data)
```

---

## 12. encoding/gcompress — 压缩

### 12.1 包概述

提供 Gzip、Zlib、Zip 三种压缩/解压功能。

源码位置：`encoding/gcompress/`

### 12.2 关键方法

| 方法 | 说明 |
|------|------|
| `Gzip(data)` / `UnGzip(data)` | Gzip 压缩/解压 |
| `Gzlib(data)` / `UnZlib(data)` | Zlib 压缩/解压 |
| `ZipPath(path, prefix)` | 压缩目录/文件为 zip |
| `UnZipFile(zipPath, dstPath)` | 解压 zip 文件 |

---

## 13. encoding/gcharset — 字符集转换

### 13.1 包概述

基于 `golang.org/x/text` 实现字符集转换。

源码位置：`encoding/gcharset/gcharset.go`

支持的字符集（`gcharset.go:9-19`）：GBK、GB18030、GB2312、Big5、EUCJP、ShiftJIS、EUCKR、UTF-8/16、macintosh、IBM*、Windows*、ISO-*。

### 13.2 使用示例

```go
// GBK 转 UTF-8
utf8Str, _ := gcharset.Convert("UTF-8", "GB18030", gbkBytes)

// 检查字符集是否支持
if gcharset.Supported("GBK") {
    // 支持
}
```

---

## 14. encoding/ghash — 哈希函数

### 14.1 包概述

提供多种经典非加密哈希算法，返回 `uint32` 或 `uint64`。

源码位置：`encoding/ghash/`

### 14.2 算法实现

| 文件 | 算法 | 返回类型 |
|------|------|---------|
| `ghash_ap.go` | AP Hash | uint32 |
| `ghash_bkdr.go` | BKDR Hash | uint32 |
| `ghash_djb.go` | DJB Hash | uint32 |
| `ghash_elf.go` | ELF Hash | uint32 |
| `ghash_jshash.go` | JS Hash | uint32 |
| `ghash_pjw.go` | PJW Hash | uint32 |
| `ghash_rs.go` | RS Hash | uint32 |
| `ghash_sdbm.go` | SDBM Hash | uint32 |

```go
h := ghash.BKDR([]byte("hello world"))
```

---

## 15. encoding/ghtml、gurl、gbase64、gbinary

### 15.1 ghtml（`encoding/ghtml/ghtml.go`）

| 方法 | 说明 |
|------|------|
| `StripTags(s)` | 移除 HTML 标签 |
| `Entities(s)` | HTML 实体编码（`html.EscapeString`） |
| `EntitiesDecode(s)` | HTML 实体解码 |
| `SpecialChars(s)` | 特殊字符编码（`&<>"'`） |

### 15.2 gurl（`encoding/gurl/url.go`）

| 方法 | 说明 |
|------|------|
| `UrlEncode(s)` / `UrlDecode(s)` | URL 编解码 |
| `RawUrlEncode(s)` / `RawUrlDecode(s)` | RFC 3986 编解码 |
| `BuildQuery(queryMap)` | 构建查询字符串 |

### 15.3 gbase64

标准 Base64 编解码封装。

### 15.4 gbinary

提供大小端编码、位操作等二进制工具。

| 方法 | 说明 |
|------|------|
| `BeInt32/beInt32` | 大端 32位编码/解码 |
| `LeInt32/leInt32` | 小端 32位编码/解码 |
| `EncodeBits/DecodeBits` | 位级编码 |

---

## 16. container/garray — 并发安全数组

### 16.1 包概述

提供普通数组和排序数组两种实现，均支持通过 `safe` 参数切换并发安全模式。

源码位置：`container/garray/`

### 16.2 类型体系

```
garray/
├── Array        = *TArray[any]      // 普通数组（泛型包装）
├── StrArray     = *TArray[string]   // 字符串数组
├── IntArray     = *TArray[int]      // 整数数组
├── SortedArray  = *TSortedArray[any] // 排序数组
├── SortedStrArray / SortedIntArray  // 类型特化排序数组
```

内部使用 `internal/rwmutex.RWMutex` 实现并发安全，该锁在 `safe=false` 时为空操作。

### 16.3 关键方法（以 Array 为例）

| 方法 | 说明 |
|------|------|
| `New(safe...)` | 创建数组 |
| `Append(v)` / `PushBack(v)` | 追加元素 |
| `Get(index)` | 按索引获取 |
| `Set(index, v)` | 按索引设置 |
| `Insert(index, v)` | 插入 |
| `Remove(index)` | 移除 |
| `Pop()` / `PopLeft()` | 弹出 |
| `Contains(v)` | 包含检查 |
| `Search(v)` | 搜索（返回索引） |
| `Len()` | 长度 |
| `Slice()` | 转为原生切片 |
| `Unique()` | 去重 |
| `Reverse()` | 反转 |
| `Sort()` | 排序 |
| `Merge(array)` | 合并 |
| `Chunk(size)` | 分块 |
| `Iterator(f)` | 遍历 |
| `Filter(filter)` | 过滤 |
| `Map(f)` | 映射 |

### 16.4 SortedArray

排序数组在插入时自动维护有序性，基于二分查找实现高效操作。需要提供比较函数 `ComparatorFunc`。

```go
arr := garray.NewSortedArray(func(a, b any) int {
    return gutil.Comparator(gconv.Int(a), gconv.Int(b))
})
arr.Add(3).Add(1).Add(2)
// 内部保持 [1, 2, 3] 有序
```

---

## 17. container/gmap — 并发安全映射

### 17.1 包概述

提供多种 map 实现：HashMap、ListMap、TreeMap，均支持并发安全切换。

源码位置：`container/gmap/`

### 17.2 类型体系

```
gmap/
├── Map / HashMap = AnyAnyMap     // 标准 hash map
├── IntAnyMap, StrAnyMap          // 类型特化 map
├── IntIntMap, StrStrMap 等       // 值类型特化
├── ListMap                       // 保持插入顺序的 map
├── TreeMap                       // 有序 map（基于红黑树）
```

类型别名的定义见 `container/gmap/gmap.go:10-13`：
```go
type (
    Map     = AnyAnyMap
    HashMap = AnyAnyMap
)
```

### 17.3 关键方法

| 方法 | 说明 |
|------|------|
| `New(safe...)` | 创建 map |
| `Set(k, v)` / `Sets(map)` | 设置键值对 |
| `Get(k)` / `GetVar(k)` | 获取值 |
| `GetOrSet(k, v)` | 获取或设置 |
| `GetOrSetFunc(k, f)` | 获取或通过函数设置 |
| `Search(k)` | 搜索（返回 found） |
| `Contains(k)` | 包含检查 |
| `Remove(k)` / `Removes(keys)` | 删除 |
| `Keys()` / `Values()` | 键/值切片 |
| `Iterator(f)` | 遍历 |
| `Flip()` | 键值交换 |
| `Merge(map)` | 合并 |
| `Clone()` | 克隆 |

### 17.4 ListMap vs TreeMap

- **ListMap**：底层使用双向链表 + map，保持插入顺序
- **TreeMap**：底层使用红黑树（来自 `gtree`），按 key 排序

---

## 18. container/gset — 并发安全集合

### 18.1 包概述

提供去重集合，底层基于 map 实现。

源码位置：`container/gset/`

### 18.2 类型体系

```
gset/
├── Set     = *TSet[any]      // 任意类型集合
├── StrSet  = *TSet[string]   // 字符串集合
├── IntSet  = *TSet[int]      // 整数集合
```

`Set` 嵌入 `*TSet[any]`（`gset_any_set.go:17-19`）：
```go
type Set struct {
    *TSet[any]
    once sync.Once
}
```

### 18.3 关键方法

| 方法 | 说明 |
|------|------|
| `New(safe...)` / `NewSet(safe...)` | 创建集合 |
| `NewFrom(items)` | 从切片创建 |
| `Add(v)` / `AddIfNotExist(v)` | 添加 |
| `Contains(v)` | 包含检查 |
| `Remove(v)` | 删除 |
| `Size()` | 大小 |
| `Slice()` | 转切片 |
| `Union(set)` | 并集 |
| `Inter(set)` | 交集 |
| `Diff(set)` | 差集 |
| `Complement(full)` | 补集 |
| `IsSubsetOf(set)` | 子集判断 |
| `Iterator(f)` | 遍历 |
| `Pop()` | 弹出随机元素 |

---

## 19. container/glist — 双向链表

### 19.1 包概述

基于标准库 `container/list` 的双向链表封装，支持并发安全切换。

源码位置：`container/glist/glist.go`

### 19.2 核心结构（`glist.go:25-31`）

```go
type List struct {
    mu   rwmutex.RWMutex
    list *list.List  // 标准库双向链表
}
type Element = list.Element
```

### 19.3 关键方法

| 方法 | 说明 |
|------|------|
| `PushFront(v)` / `PushBack(v)` | 头部/尾部插入 |
| `PushFronts(values)` / `PushBacks(values)` | 批量插入 |
| `PopFront()` / `PopBack()` | 弹出 |
| `PushBefore(mark, v)` / `PushAfter(mark, v)` | 定点插入 |
| `Front()` / `Back()` | 获取首/尾元素 |
| `Len()` | 长度 |
| `Remove(e)` | 删除 |
| `InsertsBefore(mark, values)` | 定点批量插入 |
| `IteratorAsc(f)` / `IteratorDesc(f)` | 正序/逆序遍历 |
| `Join(glue)` | 连接为字符串 |
| `Slice()` | 转切片 |

---

## 20. container/gtree — 树容器

### 20.1 包概述

提供 AVL 树、红黑树、B 树三种有序映射实现，均支持并发安全。

源码位置：`container/gtree/`

### 20.2 类型体系

```
gtree/
├── AvlTree            — AVL 自平衡二叉搜索树
├── RedBlackTree       — 红黑树（Java TreeMap 同构）
├── BTree              — B 树
├── TreeMap            = RedBlackTree  // 别名
```

### 20.3 使用示例

```go
// TreeMap（红黑树，按 key 排序）
tree := gtree.NewRedBlackTree(gutil.Comparator, true)
tree.Set("c", 3).Set("a", 1).Set("b", 2)
tree.IteratorAsc(func(k, v any) bool {
    fmt.Println(k, v) // a 1, b 2, c 3（有序输出）
    return true
})

// B 树（适合磁盘存储）
btree := gtree.NewBTree(3, gutil.Comparator)
```

---

## 21. container/gqueue、gring — 队列与环形缓冲

### 21.1 gqueue

**FIFO 队列**，底层由链表 + channel 实现。

特点：
- FIFO 顺序（data → list → chan）
- 支持动态队列大小（无限制）
- Pop 时阻塞等待

```go
q := gqueue.New(1000) // 有界队列
q.Push("item")
val := q.Pop() // 阻塞直到有数据
q.Close()      // 关闭队列，通知所有阻塞的 Pop
```

### 21.2 gring

**环形缓冲区**。

> **注意**：已标记 Deprecated。

---

## 22. container/gpool — 对象池

### 22.1 包概述

提供带 TTL 的可复用对象池，比标准库 `sync.Pool` 多了过期和销毁回调功能。

源码位置：`container/gpool/gpool.go`

### 22.2 TTL 逻辑

```go
// ttl = 0  : 永不过期
// ttl < 0  : 使用后立即过期
// ttl > 0  : 超时过期
func New(ttl time.Duration, newFunc NewFunc, expireFunc ...ExpireFunc) *Pool
```

### 22.3 使用示例

```go
pool := gpool.New(
    30*time.Second,
    func() (any, error) {
        return &bytes.Buffer{}, nil
    },
    func(i any) {
        // 过期回调（资源清理）
    },
)
obj, _ := pool.Get()
pool.Put(obj) // 放回池中复用
```

---

## 23. container/gtype — 原子类型包装

### 23.1 包概述

基于 `sync/atomic` 的高性能并发安全基础类型包装器。

源码位置：`container/gtype/`

### 23.2 类型列表

| 类型 | 底层存储 |
|------|---------|
| `Int` / `Uint` | `int64` / `uint64` |
| `Int32` / `Int64` | 对应原子类型 |
| `Float32` / `Float64` | `uint64`（通过 `math.Float64bits`） |
| `Bool` | `int32` |
| `String` / `Bytes` | `atomic.Value` |
| `Any` / `Interface` | `atomic.Value` |

### 23.3 统一 API（以 Int 为例）

```go
func NewInt(value ...int) *Int
func (v *Int) Set(value int) (old int)       // atomic.SwapInt64
func (v *Int) Val() int                       // atomic.LoadInt64
func (v *Int) Add(delta int) (new int)        // atomic.AddInt64
func (v *Int) Cas(old, new int) (swapped bool) // atomic.CompareAndSwapInt64
```

所有类型都实现了 `UnmarshalValue`、`MarshalJSON`、`DeepCopy` 接口，可与 `gconv` 无缝集成。

---

## 24. container/gvar — 泛型变量

### 24.1 包概述

`gvar` 是运行时泛型变量类型，持有任意类型的值并提供丰富的类型转换方法。它是 `gjson`、`gvalid`、`gdb` 等模块的值传递桥梁。

源码位置：`container/gvar/`

### 24.2 核心结构（`container/gvar/gvar.go:16-19`）

```go
type Var struct {
    value any
    safe  bool  // 并发安全标志
}
```

### 24.3 并发安全设计（`gvar.go:24-34`）

```go
func New(value any, safe ...bool) *Var {
    if len(safe) > 0 && safe[0] {
        return &Var{
            value: gtype.NewInterface(value),  // 包装为原子类型
            safe:  true,
        }
    }
    return &Var{value: value}
}
```

并发安全模式下，内部使用 `gtype.Interface` 存储值。

### 24.4 类型转换方法

`gvar` 通过委托 `gconv` 提供了全套类型转换方法：

```go
v := gvar.New("123")
v.Int()      // 123 (via gconv.Int)
v.String()   // "123" (via gconv.String)
v.Bool()     // true
v.Float64()  // 123.0
v.Bytes()    // [49, 50, 51]
```

### 24.5 完整方法列表

| 方法 | 说明 |
|------|------|
| `Val()` / `Interface()` | 获取原始值 |
| `Is*()` | 类型判断（IsEmpty, IsNil, IsSlice 等） |
| `Set(value)` | 设置值 |
| `Map()` / `Maps()` / `Slice()` / `Struct()` | 复合类型转换 |
| `Scan(pointer)` | 转换到结构体指针 |
| `Time()` / `Duration()` / `GTime()` | 时间转换 |
| `MarshalJSON()` / `UnmarshalJSON()` | JSON 序列化 |
| `Copy()` | 深拷贝 |
| `List()` | 转为列表 |

---

## 25. errors/gerror — 带堆栈的错误链

### 25.1 包概述

`gerror` 提供带堆栈追踪、错误码、错误链的增强错误系统。

设计约束（`errors/gerror/gerror.go:10-12`）：
> this package is quite a basic package, which SHOULD NOT import extra packages except standard packages and internal packages, to avoid cycle imports.

### 25.2 接口体系（`gerror.go:18-60`）

```go
type IEqual interface { error; Equal(target error) bool }
type ICode interface { error; Code() gcode.Code }
type IStack interface { error; Stack() string }
type ICause interface { error; Cause() error }
type ICurrent interface { error; Current() error }
type IUnwrap interface { error; Unwrap() error }
type ITextArgs interface { error; Text() string; Args() []any }
```

### 25.3 Error 结构体（`gerror_error.go:19-25`）

```go
type Error struct {
    error error       // 包装的内部错误（形成错误链）
    stack stack       // 堆栈信息
    text  string      // 自定义错误文本
    args  []any       // 格式化参数（用于 i18n）
    code  gcode.Code  // 错误码
}
```

### 25.4 错误链机制

```
Wrap/Wrapf → 创建新 Error，error 字段指向被包装的错误
     ↓
Error()  → 拼接 text + error.Error()
Cause()  → 沿 error 链找到根因
Current()→ 返回当前层级的错误（不含链）
Unwrap() → 返回下一层错误（支持 errors.Is/errors.As）
```

### 25.5 堆栈追踪（`gerror_error_stack.go`）

`Stack()` 方法遍历错误链，收集每一层的堆栈帧，过滤去重框架内部路径和 GOROOT，格式化输出。

支持两种模式：
- **Brief 模式**：过滤所有 GoFrame 包路径
- **Full 模式**：仅过滤 `gerror` 包路径

### 25.6 创建函数体系

| 函数 | 说明 |
|------|------|
| `New(text)` | 创建无错误码的错误 |
| `Newf(format, args...)` | 格式化创建 |
| `NewSkip(skip, text)` | 指定堆栈跳过层数 |
| `Wrap(err, text)` | 包装错误（继承错误码） |
| `Wrapf(err, format, args...)` | 格式化包装 |
| `NewCode(code, text...)` | 创建带错误码的错误 |
| `NewCodef(code, format, args...)` | 格式化带错误码 |
| `WrapCode(code, err, text...)` | 带错误码包装 |

### 25.7 使用示例

```go
// 创建带错误码的错误
err := gerror.NewCode(gcode.CodeInvalidParameter, "参数无效")

// 包装错误（保留错误码 + 堆栈）
if err != nil {
    err = gerror.Wrap(err, "处理用户请求失败")
}

// 提取错误信息
fmt.Println(gerror.Code(err))    // CodeInvalidParameter
fmt.Println(err.Error())         // 处理用户请求失败: 参数无效
fmt.Println(gerror.Current(err)) // 当前层错误
fmt.Println(gerror.Cause(err))   // 根因错误

// 检查错误码
if gerror.HasCode(err, gcode.CodeInvalidParameter) {
    // 处理特定错误码
}
```

---

## 26. errors/gcode — 错误码注册

### 26.1 Code 接口（`gcode.go:11-21`）

```go
type Code interface {
    Code() int        // 错误码数字
    Message() string  // 简短描述
    Detail() any      // 扩展详情
}
```

### 26.2 内置错误码

框架保留码 < 1000：

| 变量 | 码值 | 消息 |
|------|------|------|
| `CodeNil` | -1 | （无错误码） |
| `CodeOK` | 0 | OK |
| `CodeInternalError` | 50 | Internal Error |
| `CodeValidationFailed` | 51 | Validation Failed |
| `CodeDbOperationError` | 52 | Database Operation Error |
| `CodeInvalidParameter` | 53 | Invalid Parameter |
| `CodeNotAuthorized` | 61 | Not Authorized |
| `CodeNotFound` | 65 | Not Found |
| `CodeBusinessValidationFailed` | 300 | Business Validation Failed |

---

## 27. text/gstr — 字符串操作

### 27.1 包概述

提供丰富的字符串操作函数，涵盖查找、替换、分割、大小写转换、比较、格式化等。

源码位置：`text/gstr/`（20+ 文件，按功能分类）

### 27.2 常用方法示例

```go
gstr.Contains("hello world", "world")      // true
gstr.ContainsI("Hello World", "world")     // true（不区分大小写）
gstr.Replace("hello", "l", "L")            // "heLLo"
gstr.ToUpper("hello")                       // "HELLO"
gstr.CamelCase("hello_world")               // "HelloWorld"
gstr.SnakeCase("HelloWorld")                // "hello_world"
gstr.Trim("  hello  ")                      // "hello"
gstr.SubStr("hello world", 0, 5)            // "hello"
gstr.Split("a,b,c", ",")                    // ["a", "b", "c"]
gstr.IsNumeric("12345")                     // true
gstr.CompareVersion("1.2.0", "1.2.1")       // -1
```

---

## 28. text/gregex — 正则表达式封装

### 28.1 包概述

在标准库 `regexp` 基础上添加了**编译缓存**，避免重复编译相同正则表达式。

源码位置：`text/gregex/`

### 28.2 缓存机制（`gregex_cache.go:16-51`）

```go
var (
    regexMu  = sync.RWMutex{}
    regexMap = make(map[string]*regexp.Regexp) // 永久缓存
)
```

### 28.3 关键方法

| 方法 | 说明 |
|------|------|
| `Quote(s)` | 转义正则特殊字符 |
| `Validate(pattern)` | 验证正则合法性 |
| `IsMatch(pattern, src)` | 是否匹配 |
| `MatchString(pattern, src)` | 返回匹配的字符串 |
| `MatchAll(pattern, src)` | 返回所有匹配 |
| `Replace(pattern, replace, src)` | 替换 |
| `ReplaceFunc(pattern, src, f)` | 函数替换 |
| `Split(pattern, src)` | 按正则分割 |

---

## 29. crypto — 加密哈希

### 29.1 gaes — AES 对称加密

| 方法 | 模式 | 说明 |
|------|------|------|
| `EncryptCBC` / `DecryptCBC` | CBC | CBC 模式加解密 |
| `EncryptCFB` / `DecryptCFB` | CFB | CFB 模式加解密 |
| `EncryptGCM` / `DecryptGCM` | GCM | GCM 模式（推荐） |

密钥长度要求：16/24/32 字节（AES-128/192/256）。

```go
key := []byte("1234567890123456")
cipher, _ := gaes.Encrypt([]byte("plaintext"), key)
plain, _ := gaes.Decrypt(cipher, key)
```

### 29.2 grsa — RSA 非对称加密

支持 PKCS#1 和 PKCS#8 格式密钥：

| 方法族 | 填充方案 | 安全性 |
|--------|---------|--------|
| `Encrypt` / `Decrypt` | PKCS#1 v1.5 | 兼容性 |
| `EncryptOAEP` / `DecryptOAEP` | OAEP | **推荐** |

### 29.3 哈希函数

```go
gmd5.Encrypt("hello")                    // "5d41402abc4b2a76b9719d911017c592"
gsha256.Encrypt("hello")                 // SHA-256 哈希
gcrc32.Encrypt("hello")                  // CRC32 校验码
```

---

## 30. test/gtest — 测试辅助框架

### 30.1 包概述

`gtest` 是 GoFrame 的测试辅助框架，提供 `gtest.C` + `gtest.T` 范式，在标准库 `testing.T` 基础上添加了丰富的断言方法。

源码位置：`test/gtest/`

### 30.2 核心范式：gtest.C（`test/gtest/gtest_util.go:29-37`）

```go
func C(t *testing.T, f func(t *T)) {
    defer func() {
        if err := recover(); err != nil {
            t.Fail()
        }
    }()
    f(&T{t})
}
```

### 30.3 断言方法

| 方法 | 说明 |
|------|------|
| `Assert(value, expect)` | 断言相等（值比较） |
| `AssertEQ(value, expect)` | 断言相等（值 + 类型比较） |
| `AssertNE(value, expect)` | 断言不等 |
| `AssertGT/GE/LT/LE(value, expect)` | 大小比较 |
| `AssertIN(value, expect)` | 断言包含 |
| `AssertNil(value)` | 断言 nil |
| `Error(message...)` | 触发测试失败 |
| `Fatal(message...)` | 致命错误 |

### 30.4 使用规范

```go
func TestAdd(t *testing.T) {
    gtest.C(t, func() {
        gtest.Assert(Add(1, 2), 3)
        gtest.AssertEQ(Add(1, 2), 3)
        gtest.AssertNE(Add(1, 2), 4)
    })
}
```

---

## 附录：模块间依赖关系总览

```
test/gtest ─────────────────────────────────┐
    │                                        │
    ├──→ util/gconv ──→ os/gtime            │
    ├──→ text/gstr                          │
    └──→ internal/empty                     │
                                             │
container/gvar ──→ util/gconv ──→ encoding/gbinary
    │              ├──→ errors/gerror ──→ errors/gcode
    │              ├──→ os/gtime
    │              └──→ internal/json
    ├──→ container/gtype
    └──→ internal/json

util/gvalid ──→ container/gvar
    │          ├──→ container/gset
    │          ├──→ encoding/gjson
    │          ├──→ text/gstr, gregex
    │          ├──→ util/gconv, gtag, gmeta
    │          ├──→ errors/gerror
    │          └──→ i18n/gi18n

errors/gerror ──→ errors/gcode
    │           └──→ internal/errors, internal/consts

text/gregex ──→ errors/gerror (仅用于缓存错误)

crypto/* ──→ errors/gerror, errors/gcode
encoding/gjson ──→ util/gconv, container/gvar, text/gstr, errors/gerror
```

**依赖层级原则**：
1. `errors/gcode` 是最底层，无外部依赖
2. `errors/gerror` 仅依赖 `gcode`
3. `util/gconv` 依赖 `gerror`、`gtime`、`gbinary`
4. `container/gtype` 仅依赖 `gconv`
5. `container/gvar` 依赖 `gtype`、`gconv`
6. `text/*` 依赖 `gerror`
7. `util/gvalid` 是上层用户工具，依赖较多基础设施

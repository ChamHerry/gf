# GoFrame v2 深度源码分析与开发规范文档

> 基于 GoFrame 源码的完整调用链路分析，涵盖所有核心模块和扩展模块。

## 文档结构

| 部分 | 文件 | 内容 |
|------|------|------|
| 第一部分 | [part1-facade-frame.md](part1-facade-frame.md) | **Facade 层** — `frame/g` 门面包、`frame/gins` 单例容器、64 段分段锁、类型别名体系、三级配置查找策略 |
| 第二部分 | [part2-ghttp.md](part2-ghttp.md) | **HTTP 服务** — `net/ghttp` Server 生命周期、多层哈希路由树、洋葱模型中间件、Request/Response、分组路由、WebSocket、OpenAPI、静态文件服务、`net/gclient` HTTP Client |
| 第三部分 | [part3-gdb.md](part3-gdb.md) | **数据库 ORM** — `database/gdb` DB/TX 接口、Model 链式 CRUD、WhereBuilder、7 种事务传播行为 + 嵌套 Savepoint、Result/Record 映射、查询缓存、Hook 链、软删除、分库分表、With 关联查询、`database/gredis` 适配器架构 |
| 第四部分 | [part4-os.md](part4-os.md) | **OS 抽象层** — `os/gcfg`(适配器模式)、`os/glog`(Handler 链+轮转)、`os/gcache`(适配器+LRU)、`os/gsession`(Manager+Session+Storage)、`os/gview`(模板引擎)、`os/gcmd`(命令树)、`os/gfile`、`os/gctx`(OTel)、`os/gproc`(IPC)、`os/gtimer`、`os/gcron`、`os/gfsnotify`、`os/gmetric`、`os/gres`、`os/gstructs`、`os/gtime` |
| 第五部分 | [part5-util-encoding-container-errors-text-crypto-test.md](part5-util-encoding-container-errors-text-crypto-test.md) | **工具基础设施** — `util/gconv`(万能转换)、`util/gvalid`(70+ 校验规则)、`util/gutil`、`util/grand`、`util/guid`、`util/gtag`、`util/gmeta`、`encoding/gjson`(JSON 增强)、`container/*`(garray/gmap/gset/glist/gtree/gqueue/gpool/gtype/gvar)、`errors/gerror`(错误链)、`errors/gcode`、`text/gstr`、`text/gregex`、`crypto/*`、`test/gtest` |
| 第六部分 | [part6-contrib-cmd.md](part6-contrib-cmd.md) | **扩展模块与 CLI** — `contrib/drivers/*`(MySQL/PostgreSQL/ClickHouse/DM 等)、`contrib/nosql/redis`、`contrib/registry/*`(etcd/Consul/Nacos 等)、`contrib/config/*`(Apollo/Nacos)、`contrib/trace/*`(OTLP gRPC/HTTP)、`contrib/metric/otelmetric`、`contrib/rpc/grpcx`、`contrib/sdk/httpclient`、`cmd/gf` CLI 工具 |

## 框架架构分层

```
┌─────────────────────────────────────────────────────────┐
│                     用户应用代码                         │
├─────────────────────────────────────────────────────────┤
│                  frame/g (Facade 门面)                    │
│           g.Server() / g.DB() / g.Cfg() / g.Log()       │
├─────────────────────────────────────────────────────────┤
│              frame/gins (单例容器 + 懒加载)               │
├──────┬──────┬──────┬──────┬──────┬──────┬───────────────┤
│ghttp │ gdb  │ gcfg │ glog │gcache│gview │   gservice    │
│gclient│gredis│gi18n │gsession│gtimer│gcron│   gsvc       │
├──────┴──────┴──────┴──────┴──────┴──────┴───────────────┤
│    util/gconv  │  container/gvar  │  errors/gerror      │
│    util/gvalid │  container/gmap  │  text/gstr          │
├─────────────────────────────────────────────────────────┤
│            internal/ (框架内部，不对外暴露)               │
├─────────────────────────────────────────────────────────┤
│          contrib/ (扩展插件，按需导入)                    │
│  drivers/* │ registry/* │ config/* │ trace/* │ metric/* │
└─────────────────────────────────────────────────────────┘
```

## 核心设计模式速查

| 模式 | 应用模块 | 说明 |
|------|---------|------|
| **门面模式** | `frame/g` | 通过类型别名和单例访问器统一入口 |
| **工厂+注册表** | `gdb.Driver`, `gredis.Adapter` | `init()` 自注册，用户空导入激活 |
| **适配器模式** | `gcfg.Adapter`, `gcache.Adapter`, `gredis.Adapter` | 抽象接口 + 可替换实现 |
| **责任链** | `glog.Handler`, `ghttp.Middleware`, `gdb.Hook` | 链式处理，`Next()` 传递 |
| **分段锁** | `internal/instance` | 64 段锁降低并发竞争 |
| **懒加载** | `gins.*()` | 首次访问时创建单例 |
| **结构体缓存** | `gconv/internal/structcache` | 反射结果永久缓存 |
| **缓冲管道** | `util/grand` | 预生成随机数 + channel 缓冲 |

## 关键约定

- `g.DB()`/`g.Redis()` 配置缺失时会 panic，其他组件静默使用默认值
- `gconv` 标签优先级：`gconv` > `param` > `c` > `p` > `json` > 字段名
- 所有 contrib 模块通过 `_ import` 空导入激活
- 测试使用 `gtest.C(t, func(t *gtest.T){...})` 范式
- `gmetric` 在无 Provider 时使用 noop 实现
- 框架保留错误码 < 1000，自定义错误码应 >= 1000

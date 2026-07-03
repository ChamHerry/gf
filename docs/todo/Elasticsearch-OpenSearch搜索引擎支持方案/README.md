# Elasticsearch/OpenSearch 搜索引擎支持方案

> **模板定位：** LLM 持续实施控制台  
> **功能 ID：** `elasticsearch-opensearch-search-support`  
> **对应 SDD Spec：** `openspec/changes/elasticsearch-opensearch-search-support/`  
> **对应需求：** 为 GoFrame 增加 Elasticsearch 与 OpenSearch 搜索/文档引擎支持方案  
> **功能分支：** 待创建  
> **优先级：** P1  
> **负责人：** wangxc  
> **最新代码校准日期：** 2026-07-03

---

## 0. LLM-STATE（每轮实施前必须读取）

```yaml
schema_version: "1.0"
feature_id: "elasticsearch-opensearch-search-support"
feature_name: "Elasticsearch/OpenSearch 搜索引擎支持方案"
status: "done"
current_task: "none"
last_completed_task: "T-009"
next_action: "Docker E2E 验证已补齐，等待用户确认后续可按 OpenSpec 流程归档"
verification_status: "pass"
research_status: "pass"
call_chain_status: "pass"
code_calibration_status: "pass"
last_verification_command: "GF_SEARCH_E2E_OPENSEARCH_URL=http://127.0.0.1:19201 go test ./... -count=1 -race -run Test_E2EOpenSearchDocker -v"
last_updated: "2026-07-03T17:05:18+08:00"
resume_from:
  - "README.md#0-LLM-STATE每轮实施前必须读取"
  - "01-实施计划.md#6-LLM任务队列"
  - "02-实施状态与验证.md#3-执行日志"
allowed_scope:
  - "openspec/changes/<target-change>/"
  - "database/gsearch/"
  - "frame/g/"
  - "frame/gins/"
  - "internal/consts/"
  - "contrib/nosql/elasticsearch/"
  - "contrib/nosql/opensearch/"
  - "docs/todo/Elasticsearch-OpenSearch搜索引擎支持方案/"
resolved_blockers:
  - id: "B-001"
    task: "T-001"
    resolution: "Used `npm exec --yes --package=@fission-ai/openspec -- openspec ...` to run the required validate/status commands without installing a repo dependency"
    resolved_at: "2026-07-03T15:51:57+08:00"
forbidden_scope:
  - "database/gdb/"
  - "contrib/drivers/"
  - "cmd/gf/internal/cmd/gendao/"
  - "root go.mod external SDK dependency additions"
stop_condition: "所有 required=true 的 T-### 均为 done，验证报告 VERDICT: PASS"
blocker_policy: "遇到 blocked 时记录 B-###、失败证据、已尝试方案和下一恢复动作"
```

> **状态更新规则：** LLM 每完成一个任务，必须同步更新本块的 `current_task`、`last_completed_task`、`next_action`、`verification_status`、`code_calibration_status` 和 `last_updated`。生成实施方案时还必须更新 `research_status` 和 `call_chain_status`。

---

## 1. 一句话结论

> 当前最佳方案是新增 `database/gsearch` 轻量搜索抽象，并在 `contrib/nosql/elasticsearch` 与 `contrib/nosql/opensearch` 分别注册官方客户端适配器；不要把 Elasticsearch/OpenSearch 注册为 `gdb` SQL driver，也不要用一个混合 adapter 同时兼容两者。

---

## 2. 功能目标与非目标

### 2.1 目标

| R-ID | 目标 | 验收方式 | 状态 |
|---|---|---|---|
| R-001 | Root module 新增不依赖官方 ES/OpenSearch SDK 的 `database/gsearch` 抽象 | `go test ./database/gsearch/... -count=1 -race` | done |
| R-002 | 新增 `contrib/nosql/elasticsearch` 模块，依赖 Elastic 官方 Go client | `cd contrib/nosql/elasticsearch && go test ./... -count=1 -race` | done |
| R-003 | 新增 `contrib/nosql/opensearch` 模块，依赖 OpenSearch 官方 Go client | `cd contrib/nosql/opensearch && go test ./... -count=1 -race` | done |
| R-004 | 支持基础操作：Ping/Info、Index、Get、Delete、Search、Bulk、Index Management、Raw Perform | fake adapter + `httptest.Server` 覆盖 | done |
| R-005 | 支持配置分组与 facade：`search.default.type` 选择 adapter，可选 `g.Search()` | `frame/gins` 配置测试 | done |
| R-006 | 保留原生客户端逃生口，避免 root 绑定 typed API 或 DSL | API review + adapter tests | done |
| R-007 | 使用 Docker 启动真实 Elasticsearch/OpenSearch 服务完成端到端验证 | Docker E2E tests | done |

### 2.2 非目标

| 项 | 原因 | 后续处理 |
|---|---|---|
| 不实现为 `gdb.Driver` | `gdb` 强绑定 SQL、`database/sql`、事务、表字段、DAO 生成 | 无 |
| 不在 root `go.mod` 引入 ES/OpenSearch SDK | GoFrame root 保持最小依赖，Elastic v9 还要求 Go 1.24+ | 仅 contrib 引入 |
| 不实现完整 Query DSL/typed API 代理 | 两个官方客户端 typed API 分化且变化快 | NEXT-001 |
| 不实现 ES/OpenSearch 兼容服务端、自研搜索引擎、向量插件 | 专利与实现复杂度显著上升 | NEXT-002 |
| 不承诺 ES client 访问 OpenSearch 或反向访问 | 官方兼容性已分叉 | 无 |

---

## 3. 当前运行时链路 / 目标调用链

### 3.1 调研与调用链证据摘要

| 项 | 状态 | 证据位置 | 结论摘要 |
|---|---|---|---|
| 官方文档调研 | pass | `01-实施计划.md §2.4` | Elastic/OpenSearch 官方客户端均为 HTTP JSON REST，但 typed API、认证、版本兼容分叉 |
| GitHub 开源实现调研 | pass | `01-实施计划.md §2.4` | `elastic/go-elasticsearch` 与 `opensearch-go` 应分别适配；`olivere/elastic` 仅作历史 fluent API 参考 |
| DeepWiki 解析 | pass | `01-实施计划.md §2.4` | 两个官方客户端均分为 client/transport/API 层，但生成 API 与兼容矩阵不同 |
| subagent/GitNexus 调用链分析 | pass | `01-实施计划.md §3` | `gdb.Register` 只服务 SQL drivers；`gredis` 的 root 抽象 + contrib adapter 更适配本需求 |
| 当前代码实时校准 | pass | `02-实施状态与验证.md §8` | 当前仓库没有 ES/OpenSearch 实现；root/contrib 模块边界明确 |

### 3.2 目标调用链

```text
用户代码：g.Search("default") 或 gsearch.Instance("default")
  -> frame/gins.Search() 读取 search.<group> 配置
  -> database/gsearch.ConfigFromMap()
  -> database/gsearch.New(config)
  -> adapter registry 按 config.Type 选择 elasticsearch 或 opensearch
  -> contrib/nosql/<engine>.New(config)
  -> 官方 Go client 执行 HTTP JSON REST
  -> 返回 gsearch.Response / SearchResponse / BulkResponse
```

关键约束：

1. `database/gdb`、`contrib/drivers/*` 和 DAO 生成链路不得参与本功能。
2. Root `database/gsearch` 不暴露 Elastic/OpenSearch 官方 typed request/response 类型。
3. 同一个进程必须能同时导入 Elasticsearch 与 OpenSearch adapter，因此注册机制需要按 `Type` 区分，而不是 Redis 当前的单默认 adapter function。
4. Bulk/Search 不能只按 HTTP status 判断成功，必须暴露 per-item error、shard failure、timeout 与 partial result 信息。

---

## 4. 当前实施状态总览

| T-ID | 任务 | Status | DependsOn | Evidence | Next |
|---|---|---|---|---|---|
| T-001 | 创建或补充 OpenSpec 变更文档 | done | none | `npm exec --yes --package=@fission-ai/openspec -- openspec validate elasticsearch-opensearch-search-support --strict` PASS | 已完成 |
| T-002 | 实现 `database/gsearch` 配置、注册、实例缓存与 fake adapter 测试 | done | T-001 | `go test ./database/gsearch/... -count=1 -race` PASS | 进入 T-003 |
| T-003 | 实现 root 通用请求/响应与基础操作接口 | done | T-002 | `go test ./database/gsearch/... -count=1 -race -cover` PASS，coverage 87.8% | 进入 T-004 |
| T-004 | 实现 `frame/gins.Search` 与 `g.Search` facade | done | T-002 | `go test ./frame/g/... ./frame/gins/... -count=1 -race` PASS | 进入 T-005 |
| T-005 | 新增 Elasticsearch contrib adapter | done | T-003 | `cd contrib/nosql/elasticsearch && go test ./... -count=1 -race` PASS | 进入 T-006 |
| T-006 | 新增 OpenSearch contrib adapter | done | T-003 | `cd contrib/nosql/opensearch && go test ./... -count=1 -race` PASS，覆盖 adapter-local signer | 进入 T-007 |
| T-007 | 补充文档、README 双语说明与示例 | done | T-005,T-006 | README 双语文件存在；凭证占位检查与 `git diff --check` PASS | 进入 T-008 |
| T-008 | 集成验证、tidy、lint、review | done | T-007 | root/frame/ES/OS tests、`make tidy`、`make lint`、OpenSpec validate、GF Review PASS | 已完成 |
| T-009 | Docker 真实 Elasticsearch/OpenSearch 端到端验证 | done | T-008 | ES `8.19.6` 与 OpenSearch `2.11.0` 容器 E2E tests PASS；默认 contrib tests、`make tidy`、`make lint` PASS | 已完成 |

状态值只允许：`todo`、`in_progress`、`blocked`、`done`、`skipped`。

---

## 5. 交付物清单

| 类型 | 路径/对象 | 预期变更 | 状态 | 证据 |
|---|---|---|---|---|
| Spec | `openspec/changes/<change>/` | 新增或补充 proposal/design/tasks/spec | done | strict validate PASS |
| Code | `database/gsearch/` | 新增 root 抽象、配置、注册、实例缓存、fake tests | done | `go test ./database/gsearch/... -count=1 -race` PASS |
| Code | `frame/gins/gins_search.go` | 新增配置读取和实例 facade | done | `go test ./frame/g/... ./frame/gins/... -count=1 -race` PASS |
| Code | `frame/g/g_object.go` | 新增 `Search` facade | done | `go test ./frame/g/... ./frame/gins/... -count=1 -race` PASS |
| Code | `internal/consts/` | 新增 `ConfigNodeNameSearch` 常量 | done | `go test ./frame/g/... ./frame/gins/... -count=1 -race` PASS |
| Contrib | `contrib/nosql/elasticsearch/` | 新增独立 Go module | done | `go test ./... -count=1 -race` PASS |
| Contrib | `contrib/nosql/opensearch/` | 新增独立 Go module | done | `cd contrib/nosql/opensearch && go test ./... -count=1 -race` PASS |
| Test | `database/gsearch/*_test.go` | fake adapter 单元测试 | done | `go test ./database/gsearch/... -count=1 -race` PASS |
| Test | `contrib/nosql/*/*_test.go` | HTTP adapter 单元测试 | done | ES/OS contrib tests PASS |
| Test | `contrib/nosql/*/*_e2e_test.go` | Docker 真实服务 E2E 测试 | done | ES/OS Docker E2E tests PASS |
| Doc | `README.md` / `README.zh_CN.md` | 新目录文档双语说明 | done | `find contrib/nosql/elasticsearch contrib/nosql/opensearch -maxdepth 1 -type f \\( -name 'README.md' -o -name 'README.zh_CN.md' \\)` |

---

## 6. 数据模型 / 配置 / 接口

### 6.1 核心结构体草案

```go
type EngineType string

const (
    EngineTypeElasticsearch EngineType = "elasticsearch"
    EngineTypeOpenSearch    EngineType = "opensearch"
)

type Config struct {
    Type                   EngineType
    Addresses              []string
    Username               string
    Password               string
    APIKey                 string
    ServiceToken           string
    CloudID                string
    Headers                map[string]string
    CACert                 []byte
    CertificateFingerprint string
    TLS                    bool
    TLSSkipVerify          bool
    RetryOnStatus          []int
    MaxRetries             int
    CompressRequestBody    bool
    DiscoverNodesOnStart   bool
    Extra                  map[string]any
}

type Adapter interface {
    Ping(ctx context.Context) error
    Info(ctx context.Context) (*InfoResponse, error)
    Perform(ctx context.Context, req *Request) (*Response, error)
    Search(ctx context.Context, req *SearchRequest) (*SearchResponse, error)
    Bulk(ctx context.Context, req *BulkRequest) (*BulkResponse, error)
    Close(ctx context.Context) error
    Client() any
}
```

### 6.2 配置项

| 配置项 | 环境变量 | 默认值 | 行为 | 兼容性 |
|---|---|---|---|---|
| `search.<group>.type` | 无 | 无 | 选择 `elasticsearch` 或 `opensearch` adapter | 新增，不影响旧行为 |
| `search.<group>.addresses` | 无 | adapter 默认 | 节点地址列表 | 新增 |
| `search.<group>.username/password` | 无 | 空 | Basic Auth | 新增 |
| `search.<group>.apiKey` | 无 | 空 | API key auth | 新增 |
| `search.<group>.cloudId` | 无 | 空 | Elastic Cloud 专用 | Elasticsearch adapter 处理 |
| `search.<group>.aws.*` | 无 | 空 | AWS OpenSearch SigV4 | OpenSearch adapter 扩展处理 |

示例：

```yaml
search:
  default:
    type: elasticsearch
    addresses:
      - "https://localhost:9200"
    username: "elastic"
    password: "changeme"
    certificateFingerprint: "..."
  ops:
    type: opensearch
    addresses:
      - "https://localhost:9200"
    username: "admin"
    password: "admin"
```

### 6.3 API / 对外接口

| 接口 | 变更 | 兼容性 | 验证 |
|---|---|---|---|
| `gsearch.New(config)` | 新增 | 兼容 | root tests |
| `gsearch.Instance(name...)` | 新增 | 兼容 | instance tests |
| `g.Search(name...)` | 新增 facade | 兼容 | frame/gins tests |
| `contrib/nosql/elasticsearch` blank import | 新增 adapter | 兼容 | contrib tests |
| `contrib/nosql/opensearch` blank import | 新增 adapter | 兼容 | contrib tests |

---

## 7. 验收标准

| AC-ID | 验收项 | 覆盖任务 | 验证命令/证据 | Status |
|---|---|---|---|---|
| AC-001 | 未导入 contrib adapter 时返回明确必要包未导入错误 | T-002 | `go test ./database/gsearch/... -count=1 -race` | PASS |
| AC-002 | 同时导入 ES/OpenSearch adapter 时按 `type` 正确分派 | T-005,T-006 | contrib tests | todo |
| AC-003 | Root module 不新增官方 ES/OpenSearch SDK 依赖 | T-002 | `rg -n "^go \|github.com/elastic\|opensearch" go.mod` + `go test ./database/gsearch/...` | PASS |
| AC-004 | Search/Bulk 响应暴露 partial/shard/per-item error | T-003,T-005,T-006 | root fake adapter tests、ES/OS contrib `httptest.Server` tests 已覆盖 | PASS |
| AC-005 | `g.Search("group")` 能从 `search.<group>` 读取配置 | T-004 | `go test ./frame/g/... ./frame/gins/... -count=1 -race` | PASS |
| AC-006 | 文档说明版本兼容、许可、专利风险边界 | T-007 | README review | PASS |
| AC-007 | 多模块测试通过 | T-008 | root + contrib `go test` | PASS |

---

## 8. 验证入口

```bash
go test ./database/gsearch/... -count=1 -race
go test ./frame/g/... ./frame/gins/... -count=1 -race
cd contrib/nosql/elasticsearch && go test ./... -count=1 -race
cd contrib/nosql/opensearch && go test ./... -count=1 -race
make tidy
make lint
```

| C-ID | Command | 何时运行 | 预期 |
|---|---|---|---|
| C-001 | `go test ./database/gsearch/... -count=1 -race` | T-002/T-003 后 | PASS |
| C-002 | `go test ./frame/g/... ./frame/gins/... -count=1 -race` | T-004 后 | PASS |
| C-003 | `cd contrib/nosql/elasticsearch && go test ./... -count=1 -race` | T-005 后 | PASS |
| C-004 | `cd contrib/nosql/opensearch && go test ./... -count=1 -race` | T-006 后 | PASS |
| C-999 | `make tidy && make lint` | 全部任务完成后 | PASS |

---

## 9. 风险与回滚摘要

| RK-ID | 风险 | 影响 | 缓解措施 | 回滚方式 |
|---|---|---|---|---|
| RK-001 | Elastic v9 要求 Go 1.24+，GoFrame root 是 Go 1.23 | 高 | Root 不依赖 v9；contrib 选择版本并记录 CI 边界 | 回退 ES contrib 依赖版本或延后 v9 |
| RK-002 | ES/OpenSearch typed API 分叉 | 高 | Root 只暴露通用 raw/基础接口，adapter 暴露 `Client()` 逃生口 | 回退 typed wrapper |
| RK-003 | 单默认 adapter 注册无法同时支持两者 | 中 | 设计 `RegisterAdapterFunc(type, factory)` | 回退为按 type registry |
| RK-004 | Bulk/Search partial failure 被误判成功 | 高 | 响应模型暴露 per-item/shard/timeout | 修正 adapter 错误映射 |
| RK-005 | 专利/许可误用 | 中 | 只调用公开 REST API，不复制服务端实现 | 移除高风险高级功能 |

---

## 10. 文档索引

| 文档 | 用途 |
|---|---|
| [`00-使用说明.md`](00-使用说明.md) | LLM 执行协议、状态枚举、证据规范 |
| [`01-实施计划.md`](01-实施计划.md) | 详细任务队列、外部调研、调用链、设计决策 |
| [`02-实施状态与验证.md`](02-实施状态与验证.md) | 调研与命令证据、验收覆盖、当前 VERDICT |
| [`03-复盘与交接.md`](03-复盘与交接.md) | 决策、踩坑、回滚、维护与交接 |

---

## 11. 变更日志

| 时间 | 变更 | 作者/Agent | 证据 |
|---|---|---|---|
| 2026-07-03T15:20:46+08:00 | 创建实施控制台和方案文档 | Codex | official docs + DeepWiki + subagent + GitNexus |
| 2026-07-03T15:51:57+08:00 | 完成 OpenSpec strict validate，进入代码任务 | Codex | `npm exec --yes --package=@fission-ai/openspec -- openspec validate elasticsearch-opensearch-search-support --strict` |
| 2026-07-03T15:58:51+08:00 | 完成 T-002 root `gsearch` 配置、注册、实例缓存与 fake tests | Codex | `go test ./database/gsearch/... -count=1 -race` |
| 2026-07-03T16:05:42+08:00 | 完成 T-003 root 请求/响应模型和操作转发接口 | Codex | `go test ./database/gsearch/... -count=1 -race -cover`，coverage 87.8% |
| 2026-07-03T16:10:53+08:00 | 完成 T-004 `gins.Search` / `g.Search` facade 与配置分组测试 | Codex | `go test ./frame/g/... ./frame/gins/... -count=1 -race` |
| 2026-07-03T16:19:18+08:00 | 完成 T-005 Elasticsearch contrib adapter | Codex | `cd contrib/nosql/elasticsearch && go test ./... -count=1 -race` |
| 2026-07-03T16:31:22+08:00 | 完成 T-006 OpenSearch contrib adapter | Codex | `cd contrib/nosql/opensearch && go test ./... -count=1 -race` |
| 2026-07-03T16:35:23+08:00 | 补齐 T-006 OpenSearch adapter-local signer 映射测试 | Codex | `cd contrib/nosql/opensearch && go test ./... -count=1 -race` |
| 2026-07-03T16:39:10+08:00 | 完成 T-007 双语 README、示例、兼容性与风险说明 | Codex | `git diff --check -- contrib/nosql/elasticsearch contrib/nosql/opensearch docs/todo/Elasticsearch-OpenSearch搜索引擎支持方案 openspec/changes/elasticsearch-opensearch-search-support` |
| 2026-07-03T16:45:27+08:00 | 完成 T-008 最终测试、tidy、lint、OpenSpec validate 与 GF Review | Codex | `make lint` -> `0 issues.` |

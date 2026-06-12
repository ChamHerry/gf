# gf check — 项目规范检查 CLI 命令

> **模板定位：** LLM 持续实施控制台  
> **功能 ID：** `gf-check`  
> **对应需求：** 为 `gf` CLI 工具添加 `check` 子命令，用于检查 GoFrame 项目是否符合框架规范  
> **功能分支：** `feature/gf-check`  
> **优先级：** P1  
> **负责人：** wangxc  
> **最新代码校准日期：** 2026-06-12

---

## 0. LLM-STATE（每轮实施前必须读取）

```yaml
schema_version: "1.0"
feature_id: "gf-check"
feature_name: "gf check — 项目规范检查 CLI 命令"
status: "in_progress"
current_task: "T-002,T-003,T-004,T-005,T-006"
last_completed_task: "T-001"
next_action: "执行 T-002~T-006（可并行）：实现 8 个规则文件"
verification_status: "partial_pass"
research_status: "pass"
call_chain_status: "pass"
code_calibration_status: "pass"
last_verification_command: "cd cmd/gf && go build ./internal/cmd/check/"
last_updated: "2026-06-12T19:30:00+08:00"
resume_from:
  - "README.md#0-LLM-STATE每轮实施前必须读取"
  - "01-实施计划.md#6-LLM任务队列"
  - "02-实施状态与验证.md#3-执行日志"
allowed_scope:
  - "cmd/gf/internal/cmd/cmd_check.go"
  - "cmd/gf/internal/cmd/check/"
  - "cmd/gf/internal/cmd/cmd_z_unit_check_test.go"
  - "cmd/gf/internal/cmd/testdata/check/"
  - "cmd/gf/gfcmd/gfcmd.go (仅添加 cmd.Check 注册)"
forbidden_scope:
  - "os/gcmd/"
  - "database/"
  - "net/"
  - "frame/"
  - "cmd/gf/internal/cmd/cmd.go"
  - "cmd/gf/internal/cmd/cmd_fix.go"
stop_condition: "所有 required=true 的 T-### 均为 done，验证报告 VERDICT: PASS"
blocker_policy: "遇到 blocked 时记录 B-###、失败证据、已尝试方案和下一恢复动作"
```

> **状态更新规则：** LLM 每完成一个任务，必须同步更新本块的 `current_task`、`last_completed_task`、`next_action`、`verification_status`、`code_calibration_status` 和 `last_updated`。

---

## 1. 一句话结论

> gf check 处于 `in_progress`，T-001（引擎核心包）已完成并通过编译验证；下一步并行执行 T-002~T-006（8 个规则文件），全部完成后运行 `cd cmd/gf && go test ./internal/cmd/ -run Test_Check -count=1 -race`。

---

## 2. 功能目标与非目标

### 2.1 目标

| R-ID | 目标 | 验收方式 | 状态 |
|---|---|---|---|
| R-001 | `gf check` 命令可通过 `gf check`、`gf check -p <path>`、`gf check -s` 运行 | `cd cmd/gf && go run . check -h` 显示帮助 | todo |
| R-002 | 检查 GoFrame 项目目录结构规范（main.go、go.mod、api/、internal/、manifest/ 等） | 测试用例覆盖单仓和 mono-repo 两种模板 | todo |
| R-003 | 检查 API 定义规范（struct 命名 Req/Res 后缀、g.Meta path/method 标签） | AST 解析测试用例 | todo |
| R-004 | 检查 Controller 规范（ControllerV{N} 命名、方法签名、NewV{N} 工厂函数） | AST 解析测试用例 | todo |
| R-005 | 检查分层依赖规则（controller 不直接调用 dao、api 不引用 internal/model） | 导入路径分析测试用例 | todo |
| R-006 | 检查自动生成文件保护（model/do、model/entity、dao/internal 的 DO NOT EDIT 标记） | 文件头注释检查测试 | todo |
| R-007 | 检查 go.mod 规范（依赖 gf/v2、Go 版本 >= 1.23） | go.mod 解析测试用例 | todo |
| R-008 | 输出彩色格式化的检查报告，exit code 0=通过/1=有违规 | 命令行输出验证 | todo |
| R-009 | 支持 `--strict` 模式将 warning 升级为 error | CLI 选项测试 | todo |
| R-010 | 支持通过 `hack/config.yaml` 配置忽略规则 | 配置解析测试 | todo |

### 2.2 非目标

| 项 | 原因 | 后续处理 |
|---|---|---|
| 不集成 `go/analysis` 框架 | gf CLI 是独立模块，引入 `golang.org/x/tools` 会增加依赖；用 `go/parser` + `go/ast` 足够 | NEXT-001 |
| 不做 golangci-lint 集成 | 范围外，golangci-lint 有自己的插件体系 | NEXT-002 |
| 不检查业务逻辑正确性 | 语义分析超出静态检查范围 | NEXT-003 |
| 不支持自定义规则插件 | MVP 版本聚焦内置规则；后续可扩展 | NEXT-004 |
| 不检查 protobuf/gRPC 代码规范 | gRPC 规范检查需要额外工具链 | NEXT-005 |

---

## 3. 当前运行时链路 / 目标调用链

### 3.1 调研与调用链证据摘要

| 项 | 状态 | 证据位置 | 结论摘要 |
|---|---|---|---|
| 官方文档调研 | pass | `01-实施计划.md §2.4` | GoFrame 项目结构、API/controller/DAO 规范已完整调研 |
| GitHub 开源实现调研 | pass | `01-实施计划.md §2.4` | 10+ 个项目已调研，采纳 revive 的 Rule 接口模式和 go-arch-lint 的分层检查思路 |
| subagent/GitNexus 调用链分析 | pass | `01-实施计划.md §3` | gf CLI 命令注册链完整分析，新命令只需 3 处修改 |
| 当前代码实时校准 | not_run | `02-实施状态与验证.md §8` | 待 T-001 执行前校准 |

### 3.2 目标调用链

```text
入口：gf check [-p path] [-s/--strict] [--skip rule-id,...]
  -> cmd_check.go: cCheck.Index(ctx, in)
    -> check.NewEngine(path, options)
      -> engine.Scan()  // 扫描项目文件
        -> gfile.ScanDir + go/parser.ParseFile
      -> engine.Run(ctx)
        -> rules_dir.go: CheckDirectoryStructure()
        -> rules_api.go: CheckAPIDefinitions()
        -> rules_controller.go: CheckControllers()
        -> rules_dao.go: CheckDAOModel()
        -> rules_layer.go: CheckLayerDependencies()
        -> rules_config.go: CheckConfigurations()
        -> rules_module.go: CheckModule()
        -> rules_gen.go: CheckGeneratedFiles()
    -> engine.Report()
      -> result.Formatter.Format(results)
      -> mlog.Print(report)
    -> exit code 0 (pass) or 1 (violations found)
```

关键约束：

1. 不改变任何现有 `gf` 命令的行为。
2. check 命令本身不修改任何项目文件（只读检查）。
3. 失败时返回非零 exit code，但不 panic。
4. 支持 context 取消。

---

## 4. 当前实施状态总览

| T-ID | 任务 | Status | DependsOn | Evidence | Next |
|---|---|---|---|---|---|
| T-001 | 创建 check 引擎核心包（rule 接口 + result 类型 + engine 编排器） | done | none | go build + go vet PASS | 执行 T-002~T-006 |
| T-002 | 实现目录结构检查规则（rules_dir.go） | todo | T-001 | none | 执行 T-003 |
| T-003 | 实现 API 定义检查规则（rules_api.go） | todo | T-001 | none | 执行 T-004 |
| T-004 | 实现 Controller 检查规则（rules_controller.go） | todo | T-001 | none | 执行 T-005 |
| T-005 | 实现分层依赖检查规则（rules_layer.go） | todo | T-001 | none | 执行 T-006 |
| T-006 | 实现模块/go.mod/配置检查规则（rules_module.go + rules_config.go + rules_gen.go） | todo | T-001 | none | 执行 T-007 |
| T-007 | 创建 cmd_check.go 命令文件并注册到 gfcmd.go | todo | T-002,T-003,T-004,T-005,T-006 | none | 执行 T-008 |
| T-008 | 创建测试数据和编写单元测试 | todo | T-007 | none | 执行 T-009 |
| T-009 | 集成测试：对 template-single 和 template-mono 运行 check | todo | T-008 | none | 最终验证 |

状态值只允许：`todo`、`in_progress`、`blocked`、`done`、`skipped`。

---

## 5. 交付物清单

| 类型 | 路径/对象 | 预期变更 | 状态 | 证据 |
|---|---|---|---|---|
| Code | `cmd/gf/internal/cmd/cmd_check.go` | 新增命令定义 | todo | none |
| Code | `cmd/gf/internal/cmd/check/check.go` | 新增 engine 入口 | done | go build PASS |
| Code | `cmd/gf/internal/cmd/check/check_rule.go` | 新增 rule 接口和类型 | done | go build PASS |
| Code | `cmd/gf/internal/cmd/check/check_result.go` | 新增 result 类型和 formatter | done | go build PASS |
| Code | `cmd/gf/internal/cmd/check/check_project.go` | 新增 project scanner | done | go build PASS |
| Code | `cmd/gf/internal/cmd/check/rules_dir.go` | 新增目录结构检查规则 | todo | none |
| Code | `cmd/gf/internal/cmd/check/rules_api.go` | 新增 API 定义检查规则 | todo | none |
| Code | `cmd/gf/internal/cmd/check/rules_controller.go` | 新增 controller 检查规则 | todo | none |
| Code | `cmd/gf/internal/cmd/check/rules_dao.go` | 新增 DAO/model 检查规则 | todo | none |
| Code | `cmd/gf/internal/cmd/check/rules_layer.go` | 新增分层依赖检查规则 | todo | none |
| Code | `cmd/gf/internal/cmd/check/rules_config.go` | 新增配置检查规则 | todo | none |
| Code | `cmd/gf/internal/cmd/check/rules_module.go` | 新增 go.mod 检查规则 | todo | none |
| Code | `cmd/gf/internal/cmd/check/rules_gen.go` | 新增生成文件检查规则 | todo | none |
| Modify | `cmd/gf/gfcmd/gfcmd.go:78-92` | 在 AddObject 中添加 `cmd.Check` | todo | none |
| Test | `cmd/gf/internal/cmd/cmd_z_unit_check_test.go` | 新增单元测试 | todo | none |
| Test | `cmd/gf/internal/cmd/testdata/check/` | 新增测试数据（合规 + 违规样例） | todo | none |
| Doc | `cmd/gf/README.MD` | 更新命令列表 | todo | none |
| Doc | `cmd/gf/README.zh_CN.MD` | 更新命令列表 | todo | none |

---

## 6. 数据模型 / 配置 / 接口

### 6.1 数据模型

```go
// Severity 定义规则违规严重级别
type Severity string

const (
    SeverityError   Severity = "error"
    SeverityWarning Severity = "warning"
    SeverityInfo    Severity = "info"
)

// Rule 接口 — 每个检查规则实现此接口
type Rule interface {
    ID() string          // 规则 ID，如 "DIR-001"
    Name() string        // 规则名称
    Description() string // 规则描述
    Severity() Severity  // 默认严重级别
    Run(ctx context.Context, project *Project) []*Violation
}

// Violation 表示一次规则违规
type Violation struct {
    RuleID    string   `json:"ruleId"`
    Severity  Severity `json:"severity"`
    Message   string   `json:"message"`
    FilePath  string   `json:"filePath,omitempty"`
    Line      int      `json:"line,omitempty"`
    Suggestion string  `json:"suggestion,omitempty"`
}

// Project 表示被检查的项目
type Project struct {
    RootPath    string
    GoModPath   string
    ModuleName  string
    IsMono      bool
    AppDirs     []string           // mono-repo: app/ 下的服务目录
    GoFiles     map[string]*ast.File // path -> parsed AST
    DirEntries  map[string]bool     // 目录存在性缓存
}

// Engine 检查引擎
type Engine struct {
    project    *Project
    rules      []Rule
    options    *Options
    violations []*Violation
}

// Options 检查选项
type Options struct {
    Strict     bool     // 将 warning 升级为 error
    SkipRules  []string // 跳过的规则 ID 列表
    OutputJSON bool     // JSON 输出格式
}
```

### 6.2 配置项

| 配置项 | 环境变量 | 默认值 | 行为 | 兼容性 |
|---|---|---|---|---|
| `gfcli.check.skip` | 无 | `[]` | 在 `hack/config.yaml` 中配置跳过的规则 ID | 默认不改变旧行为 |
| `gfcli.check.strict` | 无 | `false` | 在 `hack/config.yaml` 中配置是否启用严格模式 | 默认不改变旧行为 |

### 6.3 API / 对外接口

| 接口 | 变更 | 兼容性 | 验证 |
|---|---|---|---|
| `gf check` | 新增 CLI 命令 | 兼容（不影响现有命令） | `cd cmd/gf && go run . check -h` |
| `gf check -p <path>` | 新增 `-p` 选项 | 兼容 | `cd cmd/gf && go run . check -p testdata/check/sample` |
| `gf check -s/--strict` | 新增 `-s` 选项 | 兼容 | `cd cmd/gf && go run . check -s` |
| `gf check --skip DIR-001,CODE-ERR-002` | 新增 `--skip` 选项 | 兼容 | `cd cmd/gf && go run . check --skip DIR-001` |

---

## 7. 验收标准

| AC-ID | 验收项 | 覆盖任务 | 验证命令/证据 | Status |
|---|---|---|---|---|
| AC-001 | `gf check` 在合规项目上 exit code=0 | T-008,T-009 | `cd cmd/gf && go run . check -p testdata/check/valid-project` | todo |
| AC-002 | `gf check` 在违规项目上 exit code=1 并输出违规列表 | T-008,T-009 | `cd cmd/gf && go run . check -p testdata/check/invalid-project` | todo |
| AC-003 | `gf check -s` 将 warning 升级为 error | T-008 | `cd cmd/gf && go run . check -s -p testdata/check/warning-only` | todo |
| AC-004 | `gf check --skip` 跳过指定规则 | T-008 | `cd cmd/gf && go run . check --skip DIR-005 -p testdata/check/valid-project` | todo |
| AC-005 | 对 `template-single` 模板检查通过 | T-009 | `cd /tmp/gf-template-single && go run /Users/wangxc/Code/gf/cmd/gf . check` | todo |
| AC-006 | 单元测试覆盖率 ≥ 80% | T-008 | `cd cmd/gf && go test ./internal/cmd/ -run Test_Check -count=1 -race -cover` | todo |
| AC-007 | `gf -h` 输出包含 `check` 命令 | T-007 | `cd cmd/gf && go run . -h` | todo |

---

## 8. 验证入口

```bash
# 单元测试
cd cmd/gf && go test ./internal/cmd/ -run Test_Check -count=1 -race

# 覆盖率
cd cmd/gf && go test ./internal/cmd/ -run Test_Check -count=1 -race -cover

# 功能测试
cd cmd/gf && go run . check -p testdata/check/valid-project
cd cmd/gf && go run . check -p testdata/check/invalid-project

# 构建验证
cd cmd/gf && go build ./...

# Lint
cd cmd/gf && golangci-lint run -c ../../.golangci.yml
```

| C-ID | Command | 何时运行 | 预期 |
|---|---|---|---|
| C-001 | `cd cmd/gf && go build ./internal/cmd/check/` | T-001 后 | PASS |
| C-002 | `cd cmd/gf && go test ./internal/cmd/ -run Test_Check_Rules_Dir -count=1 -race` | T-002 后 | PASS |
| C-003 | `cd cmd/gf && go test ./internal/cmd/ -run Test_Check_Rules_API -count=1 -race` | T-003 后 | PASS |
| C-004 | `cd cmd/gf && go test ./internal/cmd/ -run Test_Check_Rules_Controller -count=1 -race` | T-004 后 | PASS |
| C-005 | `cd cmd/gf && go test ./internal/cmd/ -run Test_Check_Rules_Layer -count=1 -race` | T-005 后 | PASS |
| C-006 | `cd cmd/gf && go build ./...` | T-007 后 | PASS |
| C-007 | `cd cmd/gf && go run . -h` | T-007 后 | 输出包含 check |
| C-008 | `cd cmd/gf && go run . check -p testdata/check/valid-project` | T-008 后 | exit code 0 |
| C-009 | `cd cmd/gf && go run . check -p testdata/check/invalid-project` | T-008 后 | exit code 1 + 违规列表 |
| C-999 | `cd cmd/gf && go test ./internal/cmd/ -run Test_Check -count=1 -race -cover` | 全部完成后 | PASS + cover ≥ 80% |

---

## 9. 风险与回滚摘要

| RK-ID | 风险 | 影响 | 缓解措施 | 回滚方式 |
|---|---|---|---|---|
| RK-001 | AST 解析在大型项目上性能较慢 | 中 | 限制扫描范围；使用文件缓存 | 添加 `-v/verbose` 超时选项 |
| RK-002 | go.mod 版本检测误判（间接依赖） | 低 | 只检查 require 块的直接依赖 | 添加 `--no-mod-check` 选项 |
| RK-003 | mono-repo 检查逻辑复杂度过高 | 中 | MVP 版本只检查 app/ 下每个子目录 | NEXT-006 分阶段增强 |
| RK-004 | 新增 check 包引入循环依赖 | 高 | check 包不导入 cmd 包；只被 cmd_check.go 引用 | 检查包放在 internal/cmd/check/ 独立子包 |
| RK-005 | 测试数据文件过多导致维护成本高 | 低 | 使用最简测试数据；分层组织 | 限制 testdata 规模 |

---

## 10. 文档索引

| 文档 | 用途 |
|---|---|
| [`00-使用说明.md`](00-使用说明.md) | LLM 执行协议、状态枚举、证据规范 |
| [`01-实施计划.md`](01-实施计划.md) | 详细任务队列、依赖、步骤、文件范围、调研证据 |
| [`02-实施状态与验证.md`](02-实施状态与验证.md) | 执行日志、命令结果、验收覆盖、最终 VERDICT |
| [`03-复盘与交接.md`](03-复盘与交接.md) | 决策、踩坑、回滚、维护与交接 |

---

## 11. 变更日志

| 时间 | 变更 | 作者/Agent | 证据 |
|---|---|---|---|
| 2026-06-12T18:20:00+08:00 | 创建实施控制台；完成调研、调用链分析、开源项目调研 | opencode (glm-5.1) | deepwiki + subagent + web research |
| 2026-06-12T19:30:00+08:00 | T-001 完成：创建 check 引擎核心包（4 文件），go build + go vet PASS | opencode (glm-5.1) | `cd cmd/gf && go build ./internal/cmd/check/` exit 0 |

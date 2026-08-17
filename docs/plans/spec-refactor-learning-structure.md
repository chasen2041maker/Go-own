---
title: '重构为 Go 实战项目集'
type: 'refactor'
created: '2026-08-17'
status: 'done'
baseline_commit: '298c93970bddda9721cea9a6f465d1b7084e4426'
review_loop_iteration: 0
context:
  - '{project-root}/README.md'
---

<frozen-after-approval reason="human-owned intent — do not modify unless human renegotiates">

## Intent

**Problem:** 当前仓库按语法章节和零散示例组织，多个 `main()` 不能组成一个可持续开发的应用，空占位目录也让学习重点停留在“看章节”而不是“做需求”。

**Approach:** 把仓库重构成单 Module、多项目的实战项目集：所有用户手写 Go 代码迁入 `practice/` 并按独立程序保存；`projects/` 包含 Task API、并发网址检测器和日志分析器，当前只为 Task API 提供可运行起点，其余项目提供有验收标准的需求说明。

## Boundaries & Constraints

**Always:** `go test ./...` 可验证整个仓库；每个项目是独立 `package main`；只使用 Go 标准库；Task API 执行 `go run ./projects/01-task-api` 即可启动；README 把后续需求写成可验收里程碑；每一份用户手写 `.go` 文件都保留在 `practice/` 并可独立运行。

**Ask First:** 加入第三方依赖、数据库或 Web 框架；提前实现本应由用户亲手完成的里程碑；在项目增长前引入多层架构。

**Never:** 删除用户手写 Go 代码；让多个含 `main()` 的练习处于同一个 Go 包；让项目互相导入业务代码；继续把练习目录作为主学习路线；创建无需求内容的空项目；加入框架、数据库、泛型或单实现接口；一次性替用户完成整个 CRUD 项目。

## I/O & Edge-Case Matrix

| Scenario | Input / State | Expected Output / Behavior | Error Handling |
|----------|---------------|---------------------------|----------------|
| 启动项目 | `go run ./projects/01-task-api` | HTTP 服务监听 README 声明的本地端口 | 启动失败记录错误并非零退出 |
| 健康检查 | `GET /health` | 返回 200 和合法 JSON | 不支持的方法返回 405 |
| 验证项目 | `go test ./...` | 健康检查测试通过，项目完整编译 | 任一失败使命令非零退出 |

</frozen-after-approval>

## Code Map

- `projects/01-task-api/` -- 当前主项目的入口、任务模型、健康检查和测试。
- `projects/02-url-checker/`、`projects/03-log-analyzer/` -- 后续项目的需求与验收说明，不提前实现。
- `README.md`、`projects/README.md` -- 项目集入口、项目状态和操作方式。
- `practice/01-hello` 至 `practice/11-http-server` -- 集中保存全部用户手写程序；每个含 `main()` 的程序拥有独立子目录，保留已有 hello 测试和说明。
- `learn/` -- 其中的 Go 文件迁入 `practice/` 后删除旧章节结构；空章节 README 不迁移。
- `cmd/`、`internal/`、`pkg/` -- 只有未来用途说明的空占位结构，删除；原 `projects/README.md` 由实战项目入口替换。
- `docs/plans/spec-go-learning-foundation.md` -- 已完成且与新方向冲突的旧结构计划，删除。

## Tasks & Acceptance

**Execution:**
- [x] `projects/01-task-api` -- 建立可运行但不过度完成的 Task API 起步项目，只交付健康检查、任务数据模型和测试示范。
- [x] `projects/02-url-checker/README.md`、`projects/03-log-analyzer/README.md` -- 写明两个后续真实项目的目标、递进需求和完成条件，不创建空实现。
- [x] `projects/README.md` -- 展示三个项目的状态和统一操作方式。
- [x] `practice/` -- 使用文件移动完整保留 `learn/01-基础语法` 下的全部手写 Go 程序、hello 测试和说明；按独立 `main` 包拆分，并修正 HTTP 示例的 JSON 标签。
- [x] `learn/`、`cmd/`、`internal/`、`pkg/`、`docs/plans/spec-go-learning-foundation.md` -- 完成代码迁移后删除旧章节结构、空占位文件和过时计划。
- [x] `README.md`、`docs/README.md` -- 改成项目入口；列出内存存储、CRUD、测试、并发安全、JSON 持久化和工程化里程碑，每项包含完成条件但不提供答案。
- [x] 全部 Go 文件 -- 执行 `gofmt`，通过仓库级测试、静态检查和一次真实健康检查。

**Acceptance Criteria:**
- Given 仓库根目录，when 执行 `go run ./projects/01-task-api`，then Task API 成功启动且 `/health` 可访问。
- Given 仓库根目录，when 执行 `go test ./...` 和 `go vet ./...`，then 项目通过测试和静态检查。
- Given 用户打开根 README 或 `projects/README.md`，when 选择项目，then 能看到三个独立项目的目标、状态和进入方式。
- Given 用户需要查看旧练习，when 打开 `practice/`，then 每份原有 Go 代码仍存在且可按子目录单独运行。
- Given 重构后的文件树，when 检查仓库，then 只保留当前项目代码、必要配置和当前设计记录。

## Spec Change Log

## Design Notes

仓库使用一个根 `go.mod`，三个项目各自是独立 `main` 包，避免 `go.work` 和多 Module 的额外认知负担。Task API 初期直接使用少量同包文件，不创建 `cmd/internal/domain/repository`。`practice/` 只是历史练习收藏，不决定学习顺序；当某个项目真的出现职责或测试隔离问题时，再拆包并把重构作为学习任务。

## Verification

**Commands:**
- `gofmt -l $(rg --files -g '*.go')` -- expected: 不输出文件名。
- `go test ./...` -- expected: 所有包编译，已有测试通过。
- `go vet ./...` -- expected: 不报告问题。
- `go run ./projects/01-task-api` 后请求 `http://localhost:8080/health` -- expected: 200 与合法 JSON。
- `git status --short` -- expected: 只包含本次结构迁移、明确删除和文档更新。

## Suggested Review Order

**项目入口与学习路线**

- 从仓库首页理解多项目结构和当前学习主线。
  [`README.md:1`](../../README.md#L1)

- 三个项目按能力递进，同时保持业务代码互不依赖。
  [`projects/README.md:1`](../../projects/README.md#L1)

- Task API 里程碑只定义需求和验收，不提前给答案。
  [`01-task-api/README.md:23`](../../projects/01-task-api/README.md#L23)

- 并发项目从串行版本逐步演进到受控并发。
  [`02-url-checker/README.md:1`](../../projects/02-url-checker/README.md#L1)

- 日志项目先保证正确解析，再用基准决定并发。
  [`03-log-analyzer/README.md:1`](../../projects/03-log-analyzer/README.md#L1)

**最小可运行 Task API**

- 单一入口监听 README 声明的端口并复用同一 Handler。
  [`main.go:8`](../../projects/01-task-api/main.go#L8)

- 标准库路由只实现当前需要的健康检查。
  [`handler.go:8`](../../projects/01-task-api/handler.go#L8)

- 任务模型为后续手写 CRUD 提供最小数据边界。
  [`task.go:5`](../../projects/01-task-api/task.go#L5)

- HTTP 测试覆盖成功、合法 JSON 与错误方法。
  [`handler_test.go:11`](../../projects/01-task-api/handler_test.go#L11)

**旧代码保存**

- 11 个独立练习目录保留全部手写 Go 文件。
  [`practice/README.md:1`](../../practice/README.md#L1)

- 原 HTTP 示例仅修正无效 JSON 标签并完成格式化。
  [`11-http-server/main.go:13`](../../practice/11-http-server/main.go#L13)

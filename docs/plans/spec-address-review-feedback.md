---
title: '落实项目集审查反馈'
type: 'chore'
created: '2026-08-17'
status: 'done'
baseline_commit: 'a6daad1024daba3899c624e3c9f7593076222184'
review_loop_iteration: 2
context:
  - '{project-root}/docs/plans/spec-refactor-learning-structure.md'
---

<frozen-after-approval reason="human-owned intent — do not modify unless human renegotiates">

## Intent

**Problem:** 外部审查确认项目集可用，但发现本地监听范围、现有 HTTP 契约测试和三个项目的学习验收口径仍有歧义；Windows 当前 `CGO_ENABLED=0`，README 直接要求 `-race` 也会让初学者误判环境错误。

**Approach:** 落实所有适用于当前仓库的审查意见：收紧 Task API 本地监听并补测试断言，补全 Go 版本、任务 API、并发测试、日志报告和项目推进顺序的文档契约；仍不提前实现 CRUD、持久化或后续项目代码。

## Boundaries & Constraints

**Always:** 保持单 `go.mod`、三项目结构和标准库边界；HTTP 服务只监听本机回环地址；每个里程碑同步写本阶段测试；未来需求必须给出唯一、可测试的边界行为；说明 Windows Race Detector 的 CGO/C 编译器或 WSL 前置条件。

**Ask First:** 修改 Go Module 路径；加入第三方依赖；实现 Store、CRUD、文件持久化、并发 Worker Pool 或工程化功能；删除现有项目或练习。

**Never:** 为审查建议引入 DTO 目录、接口、分层或配置系统；声称当前 Windows 环境已通过 `-race`；把未来里程碑说明误写成已经实现的功能。

## I/O & Edge-Case Matrix

| Scenario | Input / State | Expected Output / Behavior | Error Handling |
|----------|---------------|---------------------------|----------------|
| 启动 Task API | `go run ./projects/01-task-api` | 仅监听 `127.0.0.1:8080` | 启动失败记录错误并退出 |
| 错误健康检查方法 | `POST /health` | 返回 `405` 且 `Allow: GET` | 测试覆盖状态码与 Header |
| 运行 Race Detector | Windows 且 `CGO_ENABLED=0` | 文档提前说明所需环境 | 不把工具链错误归因于业务代码 |

</frozen-after-approval>

## Code Map

- `README.md` -- Go 1.26 最低版本、本地启动地址和常规验证入口。
- `projects/01-task-api/main.go`、`main_test.go`、`handler_test.go` -- 精确回环监听、启动成功语义与 `Allow` Header 回归保护。
- `projects/01-task-api/README.md` -- 列表顺序、服务器管理字段、逐阶段测试和细分后的持久化/工程化任务。
- `projects/02-url-checker/README.md` -- 本地 `httptest.Server`、取消结果和 Race Detector 前置条件。
- `projects/03-log-analyzer/README.md` -- `invalid_lines` 结构、平均耗时类型/舍入和空输入规则。
- `projects/README.md` -- 明确 Task API 1～5、URL Checker、Log Analyzer、再回到 Task API 6～7 的顺序。
- `docs/README.md`、`.repo-wiki/wiki-plan.toml` -- 登记本次审查落实规格，保持文档检查通过。

## Tasks & Acceptance

**Execution:**
- [x] `projects/01-task-api/handler_test.go` -- 增加 `Allow: GET` 断言，并通过临时移除 Header 的变异检查证明测试能失败。
- [x] `projects/01-task-api/main.go` -- 把固定监听地址收紧为 `127.0.0.1:8080`，不引入配置抽象。
- [x] `README.md`、`projects/README.md`、三个项目 README -- 落实全部当前和未来需求边界，拆小过大的后期里程碑。
- [x] `docs/README.md`、`.repo-wiki/wiki-plan.toml` -- 登记规格并保持文档入口一致。
- [x] 第二轮审查 -- 通过地址测试和预绑定收紧启动语义，并覆盖五种不支持的健康检查方法。
- [x] 第二轮审查 -- 明确 Task API 输入、持久化恢复、配置、超时与强制关闭契约。
- [x] 第二轮审查 -- 明确 URL Checker CLI/结果分类和 Log Analyzer 解析/错误/溢出边界。
- [x] 最终审查 -- 收紧精确监听地址测试及 Task API HTTP、路径、持久化恢复和退出契约。
- [x] 最终审查 -- 固定 URL Checker 与 Log Analyzer 的 CLI、输出、取消、错误和溢出行为。

**Acceptance Criteria:**
- Given Task API 启动，when 检查监听端点并请求 `/health`，then 仅回环地址可见且 GET 返回合法 JSON。
- Given `POST /health`，when 运行 Handler 测试，then 同时验证 `405` 与 `Allow: GET`。
- Given 初学者按任一项目 README 开发，when 遇到列表、服务器字段、取消、无效日志或 `-race`，then 文档提供唯一且可测试的预期。
- Given 全部修改完成，when 执行格式、测试、Vet 和 Repo Wiki 检查，then 所有命令通过且未新增第三方依赖。

## Spec Change Log

- 2026-08-17：完成代码、测试与文档修改；验证回环监听、HTTP 回归测试、仓库测试、Vet 和 Repo Wiki 检查。
- 2026-08-17：根据第二轮审查收紧启动成功语义、HTTP 方法覆盖及三个项目的边界契约；聚焦测试、运行时检查和仓库门禁通过后恢复完成状态。
- 2026-08-17：根据最终审查固定精确监听地址，并补齐 HTTP/持久化、并发 CLI 和日志分析的确定性边界；规格保持 `in-review` 等待最终复核。

## Design Notes

`module go-own`、单 Module、多项目、`.repo-wiki` 和当前扁平 Task API 均保留：审查没有证明它们妨碍当前学习。审查中明确要求以后处理的功能只补契约，不提前实现。

## Verification

**Commands:**
- `$goFiles = @('projects/01-task-api/main.go', 'projects/01-task-api/main_test.go', 'projects/01-task-api/handler_test.go'); gofmt -l $goFiles` -- expected: 无输出。
- `go test ./...` -- expected: 全部通过。
- `go vet ./...` -- expected: 无报告。
- `go test -race ./...` -- expected now: 若 `CGO_ENABLED=0`，仅核对 README 已解释前置条件，不宣称通过。
- `python C:/Users/15234/.codex/skills/maintain-repo-wiki/scripts/repo_wiki.py check --root .` -- expected: structural check passed。

## Suggested Review Order

**运行边界与回归保护**

- 先绑定成功再记录日志，并固定本机回环地址。
  [`main.go:9`](../../projects/01-task-api/main.go#L9)

- 精确地址测试防止以后重新暴露到局域网。
  [`main_test.go:5`](../../projects/01-task-api/main_test.go#L5)

- 健康检查覆盖成功和五种不支持的方法。
  [`handler_test.go:11`](../../projects/01-task-api/handler_test.go#L11)

**项目契约与学习顺序**

- Task API 固定请求、响应、持久化和退出边界。
  [`01-task-api/README.md:25`](../../projects/01-task-api/README.md#L25)

- URL Checker 固定 CLI、结果索引和取消语义。
  [`02-url-checker/README.md:5`](../../projects/02-url-checker/README.md#L5)

- Log Analyzer 固定输入校验、输出和错误规则。
  [`03-log-analyzer/README.md:32`](../../projects/03-log-analyzer/README.md#L32)

- 项目入口明确完整的往返学习路线。
  [`projects/README.md:5`](../../projects/README.md#L5)

**环境与文档治理**

- 根入口说明 Go 版本和本地监听范围。
  [`README.md:23`](../../README.md#L23)

- 文档路由登记本次审查落实规格。
  [`wiki-plan.toml:29`](../../.repo-wiki/wiki-plan.toml#L29)

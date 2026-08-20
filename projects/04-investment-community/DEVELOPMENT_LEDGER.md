# 原创投资内容社区开发总账

> 快照：2026-08-20（Asia/Taipei）
>
> 状态：已安全暂停；chunk-01～06 完成，chunk-07 保存为可编译、已通过核心 MySQL 测试的 WIP，chunk-08 未开始。
> 原创边界：只借鉴成熟社区的功能范围；Go 代码、接口、表、测试和虚构数据均独立设计，不复制公司源码、真实数据、密钥或品牌资产。

## 1. 唯一续接位置

| 项目 | 当前值 |
| --- | --- |
| Git 仓库 | `C:\company\own\Go-own` |
| 开发 worktree | `C:\Users\15234\.config\superpowers\worktrees\Go-own\stock-community-governance` |
| 分支 | `codex/stock-community-governance` |
| 完成态 HEAD | `421458e`（chunk-06） |
| 当前 WIP | 续接时以 `git log -1 --oneline` 为准，提交说明含 `wip(chunk-07)` |
| 临时执行状态 | 根目录 `HANDOFF.md` |
| 规格 | `docs/plans/spec-investment-community.md` |
| 实施计划 | `docs/plans/2026-08-19-investment-community-implementation.md` |
| API 事实源 | `projects/04-investment-community/contracts/openapi.yaml` |

新对话必须切到上述 worktree，先读本总账、`HANDOFF.md`、规格、计划、阶段 07 教材和 OpenAPI；不要在 `main` 重做已完成代码。

## 2. 已完成并提交

| Chunk | Commit | Tag | 能力 |
| --- | --- | --- | --- |
| 01 工程地基 | `3e5b096` | `stock-v1/s01-done` | 双轨骨架、配置、HTTP 生命周期、错误信封、Request ID、MySQL Migration 与并发迁移 |
| 02 认证 | `6a36e66` | `stock-v1/s02-done` | 注册、登录、`/me`、bcrypt、JWT、数据库角色与停用状态回查 |
| 03 证券与圈子 | `2884112` | `stock-v1/s03-done` | 虚构证券、圈子、签名游标、加入/退出、并发事务、原子 Seed |
| 04 帖子 | `52c04cb` | `stock-v1/s04-done` | 帖子 CRUD、标签、创建幂等、筛选分页、乐观锁、软删除与举报收口 |
| 05 评论与通知 | `1bfe56e` | `stock-v1/s05-done` | 评论/一级回复、幂等、评论/回复通知、已读、作者删除收口 |
| 06 举报受理 | `421458e` | `stock-v1/s06-done` | 举报创建、查重、自举报拒绝、管理员举报队列 |

文档、契约、八阶段教材及本总账的基线提交为 `d798a8b`。`stock-v1/starter` 暂指阶段一绿色提交，最终教学交付时再校正。

## 3. chunk-07 当前 WIP（不得标记完成）

已经写入：

- `domain/audit.go`：审计动作、审计分页、治理冲突和恢复结果模型；
- `usecase/governance.go`：管理员前置授权、决策/恢复输入校验、审计分页；
- `httpapi/governance.go`：决策、恢复、审计 Handler 及稳定错误映射；
- `store/mysql/governance.go`：目标优先锁序、ignore/hide、同目标举报收口、治理通知、同事务审计、恢复 CAS/ABA、有界 1213/1205 整事务重试；
- 单元、HTTP 和真实 MySQL 集成测试。

已验证：

```powershell
go test ./projects/04-investment-community/reference/... -count=1

$env:COMMUNITY_TEST_DSN='<本机临时测试 DSN>'
go test -tags=integration ./projects/04-investment-community/reference/internal/store/mysql `
  -run 'Test(ConcurrentDecision|AuditInsertFailure|GovernanceRetry)' -count=1
```

两条命令均通过。真实 MySQL 覆盖：两个管理员不同决策只有一个改变状态、审计插入失败完整回滚、相同 hide/restore 重试不重复通知与审计、旧 restore 被后续 hide 的治理版本阻止。

仍缺：

1. 把 `GovernanceApplication` 接入 `router.go` 的完整构造器；
2. 在 `cmd/api/main.go` 创建 `GovernanceService` 并注入；
3. 补审计游标签名绑定与治理错误响应的专门回归测试；
4. 跑真实 HTTP+MySQL 三条管理员接口；
5. 跑项目 test/vet/build、OpenAPI/验收覆盖、文档影响检查和独立复审；
6. 复审通过后提交完成态并打 `stock-v1/s07-done`。当前 WIP 不能打完成 Tag。

## 4. 下一步严格顺序

1. 从上述 6 个缺口续完 chunk-07；不得重写已经通过的事务核心。
2. chunk-08：Docker Compose、Dockerfile、Swagger UI、黑盒 acceptance、CI、演示脚本和教学入口收口。
3. 全量验证与最终复审；更新总账，删除临时 `HANDOFF.md`，运行 Repo Wiki 门禁。
4. 未经用户要求，不推送、不合并 `main`、不删除 MySQL 测试数据。

## 5. 已知环境与基线

- MySQL 测试容器：`go-own-community-integration`，`mysql:8.0.46`，本机端口 `13385`；DSN 只临时注入环境变量，不写入 Git。
- Reference API 当前未运行。
- 全仓 `go vet ./...` 有一个与本项目无关的既有失败：`practice/07-functions/answers/exercise.go:70` 不可达代码；项目范围 vet 仍需在每阶段验证。

## 6. 续接提示词

> 在 `C:\Users\15234\.config\superpowers\worktrees\Go-own\stock-community-governance` 的 `codex/stock-community-governance` 分支继续原创 Go 投资内容社区。先读 `projects/04-investment-community/DEVELOPMENT_LEDGER.md` 和根 `HANDOFF.md`。chunk-01～06 已完成；chunk-07 已保存为通过默认 reference 测试和三项真实 MySQL 治理测试的 WIP，但尚未接入 router/main，也未完成 HTTP 实测、全量验证、复审和 Tag。严格从总账列出的 6 个缺口继续，保持 TDD、中文 why 注释和原创边界。

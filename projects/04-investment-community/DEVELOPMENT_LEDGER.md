# 原创投资内容社区开发总账

> 快照：2026-08-20 13:40:49 +08:00（Asia/Taipei）
>
> 状态：开发中；chunk-01～04 已完成并复审，chunk-05 仅有未提交的 Domain/Usecase 半成品。
>
> 权威原则：只参考成熟社区常见功能范围，全部 Go 代码、接口、表、测试和虚构数据独立设计；不得复制公司源码、真实数据、密钥或品牌资产。

## 1. 续接位置

| 项目 | 当前值 |
| --- | --- |
| Git 仓库 | `C:\company\own\Go-own` |
| 开发 worktree | `C:\Users\15234\.config\superpowers\worktrees\Go-own\stock-community-governance` |
| 分支 | `codex/stock-community-governance` |
| 当前分支 HEAD | 本总账已提交；续接时以 `git rev-parse HEAD` 的实际输出为准 |
| 最后完成的业务代码 | `52c04cbcfc9949ae3589def15769e38e096d5a9b`（chunk-04） |
| 基线 | `origin/main@45d8a53500d049eb14d3f39cf4fc4c923c2ef96b` |
| 临时执行状态 | 根目录 `HANDOFF.md` |
| 实施计划 | `docs/plans/2026-08-19-investment-community-implementation.md` |
| 分块分析 | `docs/plans/2026-08-19-investment-community-implementation.chunked-analysis.md` |
| 开发规格 | `docs/plans/spec-investment-community.md` |
| API 事实源 | `projects/04-investment-community/contracts/openapi.yaml` |

另一个对话开始后必须先切到上述 worktree，完整阅读本总账、`HANDOFF.md`、规格、计划和 OpenAPI，再执行任何修改。禁止在 `main` 上重做已完成代码。

## 2. 已完成并提交

| Chunk | 结果 | Commit | Tag | 主要能力 |
| --- | --- | --- | --- | --- |
| 01 工程地基 | APPROVED | `3e5b096` | `stock-v1/s01-done` | reference/starter、配置、HTTP 生命周期、统一错误、Request ID、MySQL Migration、checksum、advisory lock、结构 marker、真实并发迁移 |
| 02 认证 | APPROVED | `6a36e66` | `stock-v1/s02-done` | 注册、登录、`/me`、bcrypt、JWT、数据库角色/停用状态回查、严格 JSON、MySQL 用户仓储 |
| 03 证券与圈子 | APPROVED | `2884112` | `stock-v1/s03-done` | 虚构证券、圈子、签名游标、`joined=true/false` 成员状态、并发事务、原子 Seed |
| 04 帖子 | APPROVED | `52c04cb` | `stock-v1/s04-done` | 帖子 CRUD、证券标签、创建幂等、筛选游标、version 乐观锁、软删除、`author_deleted` 举报收口 |

`stock-v1/starter` 当前仍指向阶段一绿色提交；最终交付前需根据学习工作区方案决定是否移动到只含 starter+教材的专用提交。

## 3. 已有强验证证据

- 默认项目测试、项目范围 `go vet` 和 reference/starter 构建在 chunk-01～04 每阶段提交前通过。
- MySQL 8.0.46 真实执行通过：并发 Migration、兼容部分 DDL 恢复、不兼容旧表拒绝、checksum/marker、用户仓储、证券/圈子分页、并发加入/退出、外键、Seed 重跑与回滚、帖子/标签事务、并发幂等、version 竞争和删除举报收口。
- 真实 HTTP+MySQL 已跑通：注册→登录→`/me`；证券/圈子→重复加入→重复退出；发帖→幂等重放→更新→旧 version 409→筛选列表→删除→删除后 404。
- HTTP 响应检查确认公开作者不包含邮箱。
- 全仓 `go vet ./...` 有一个与本项目无关的既有基线失败：`practice/07-functions/answers/exercise.go:70` 不可达代码；新项目自身 vet 已通过。

## 4. 当前未提交状态（不要误判为完成）

chunk-05 正在实现“评论、一级回复、站内通知”，中断时只有以下文件存在：

```text
projects/04-investment-community/reference/internal/domain/comment.go
projects/04-investment-community/reference/internal/domain/notification.go
projects/04-investment-community/reference/internal/usecase/interactions.go
projects/04-investment-community/reference/internal/usecase/interactions_test.go
```

聚焦命令已通过：

```powershell
go test ./projects/04-investment-community/reference/internal/domain `
        ./projects/04-investment-community/reference/internal/usecase -count=1
```

但 chunk-05 仍缺 HTTP Handler、MySQL Store、真实事务/并发测试、装配、完整验证、独立复审、提交和 Tag。不得把当前半成品标记为 GREEN 或提交为完成阶段。

## 5. 未提交但应保留的设计/教学资产

- `projects/04-investment-community/contracts/`：21 个 OpenAPI 操作与验收场景。
- `projects/04-investment-community/docs/`：原创声明、架构、数据模型、治理说明、八阶段教学文档。
- `projects/04-investment-community/README.md`、`.env.example`。
- `docs/plans/*investment-community*` 与 `spec-investment-community.md`。
- 根 `README.md`、`projects/README.md`、`docs/README.md`、`.repo-wiki/wiki-plan.toml`、`.gitignore` 的项目入口更新。

这些文件尚未提交，不能删除或重新生成覆盖。Repo Wiki 当前只因临时 `HANDOFF.md` 未登记而失败；最终删除 HANDOFF 后应再次运行检查。

## 6. 下一步严格顺序

1. 完成 chunk-05：评论/回复、四类通知中的 `comment/reply`、通知列表/已读、删除评论与 `author_deleted` 举报收口；真实 MySQL 验证、复审、提交并打 `stock-v1/s05-done`。
2. chunk-06：举报创建、重复举报先查重、自举报拒绝、管理员待办、作者删除收口回归。
3. chunk-07：目标优先锁序、ignore/hide、四类通知中的 `content_hidden/content_restored`、审计、`moderation_version` 防 ABA 恢复、deadlock/timeout 有界整事务重试。
4. chunk-08：Docker Compose、Swagger、黑盒 acceptance、CI、演示脚本、教材/门户最终校对、Repo Wiki 门禁。
5. 全量验证、最终复审、删除 `HANDOFF.md`、更新本总账状态；未获用户要求不得推送或合并 main。

## 7. 当前本机运行状态

- MySQL 容器：`go-own-community-integration`，镜像 `mysql:8.0.46`，端口 `127.0.0.1:13385 → 3306`，用于专用测试 Schema。
- Reference API：未运行；`127.0.0.1:18084` 当前无监听。
- 测试 DSN 仅在本机命令中临时注入，不得写入 Git 或总账。

## 8. 续接提示词

可把下面内容直接发给另一个 Codex 对话：

> 在 `C:\Users\15234\.config\superpowers\worktrees\Go-own\stock-community-governance` 的 `codex/stock-community-governance` 分支继续开发原创 Go 投资内容社区。先完整阅读 `projects/04-investment-community/DEVELOPMENT_LEDGER.md`、根 `HANDOFF.md`、规格、实施计划和 OpenAPI。保护当前 chunk-05 半成品，不得重置或覆盖。chunk-01～04 已完成复审，最后业务提交为 `52c04cb`，其后还有本总账文档提交；从 chunk-05 评论/通知继续，严格 TDD、真实 MySQL、独立复审和阶段 Tag。实现必须原创，只参考功能范围，关键并发/权限/事务写中文 why 注释，并保持 starter 与八阶段教材可供我亲手重写。

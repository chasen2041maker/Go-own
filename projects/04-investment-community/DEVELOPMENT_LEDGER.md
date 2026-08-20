# 原创投资内容社区开发总账

> 快照：2026-08-20（Asia/Taipei）
>
> 状态：V1 全部完成；本地门禁、真实 MySQL、21 接口黑盒验收、Linux Compose 冷构建、Swagger/OpenAPI 探测与 CI 均已验证。
> 原创边界：只借鉴成熟社区的功能范围；Go 代码、接口、表、测试和虚构数据均独立设计，不复制公司源码、真实数据、密钥或品牌资产。

## 1. 唯一续接位置

| 项目 | 当前值 |
| --- | --- |
| Git 仓库 | `https://github.com/chasen2041maker/Go-own` |
| 本机完整交付副本 | `C:\company\own\Go-own\dist\stock-community-final-clone` |
| 分支 | `codex/stock-community-governance` |
| 完成态 HEAD | `stock-v1/s08-done`（以 `git show -s --oneline stock-v1/s08-done` 为准） |
| 当前 WIP | 无 |
| 临时执行状态 | 根目录 `HANDOFF.md` |
| 规格 | `docs/plans/spec-investment-community.md` |
| 实施计划 | `docs/plans/2026-08-19-investment-community-implementation.md` |
| API 事实源 | `projects/04-investment-community/contracts/openapi.yaml` |

新对话在任意克隆中获取并切到 `codex/stock-community-governance`，先读本总账、`HANDOFF.md`、规格、计划、阶段 07 教材和 OpenAPI；不要在 `main` 重做已完成代码。本机也保留了上表所列的干净完整副本和 `dist/stock-community-governance-final.bundle`。

## 2. 已完成并提交

| Chunk | Commit | Tag | 能力 |
| --- | --- | --- | --- |
| 01 工程地基 | `3e5b096` | `stock-v1/s01-done` | 双轨骨架、配置、HTTP 生命周期、错误信封、Request ID、MySQL Migration 与并发迁移 |
| 02 认证 | `6a36e66` | `stock-v1/s02-done` | 注册、登录、`/me`、bcrypt、JWT、数据库角色与停用状态回查 |
| 03 证券与圈子 | `2884112` | `stock-v1/s03-done` | 虚构证券、圈子、签名游标、加入/退出、并发事务、原子 Seed |
| 04 帖子 | `52c04cb` | `stock-v1/s04-done` | 帖子 CRUD、标签、创建幂等、筛选分页、乐观锁、软删除与举报收口 |
| 05 评论与通知 | `1bfe56e` | `stock-v1/s05-done` | 评论/一级回复、幂等、评论/回复通知、已读、作者删除收口 |
| 06 举报受理 | `421458e` | `stock-v1/s06-done` | 举报创建、查重、自举报拒绝、管理员举报队列 |
| 07 治理闭环 | `bc20705` | `stock-v1/s07-done` | 目标优先锁序、隐藏/恢复、治理版本、通知、审计、并发与 ABA 防护 |

文档、契约、八阶段教材及本总账的基线提交为 `d798a8b`；chunk-08 工程交付为 `fae8988`，提交后真实 MySQL 审计修复为 `c140790`。`stock-v1/starter` 永久保留为阶段一绿色历史快照，不再移动；最终不可变 `stock-v1/learner-start` 已创建，供学习者获得完整教材/契约和最小 starter 起点。

## 3. chunk-08 最终交付

已经写入：

- `acceptance/api_test.go`：带 `acceptance` build tag、只走 HTTP 的完整治理旅程；
- `delivery_contract_test.go`：OpenAPI 恰好 21 个操作及路由字面量、starter 隔离门禁；
- `Dockerfile`、`compose.yaml`、`swagger-nginx.conf`：MySQL → migrate → seed → API → Swagger；
- `scripts/demo.ps1` 与 `scripts/create-integration-schema.ps1`：黑盒演示及独立测试 schema 创建；
- `.github/workflows/investment-community.yml`：默认门禁、真实 MySQL、HTTP acceptance 与 Linux Compose 冷构建；
- 非 root 容器、回环端口、构建秘密排除、API 安全响应头、结构化脱敏访问日志；
- 确定性 Repo Wiki 文档目录/路由页，以及项目、学习路线、根入口与规格状态同步。

本轮新鲜验证：

```powershell
go test ./... -count=1
go vet ./projects/04-investment-community/...
go build ./projects/04-investment-community/reference/cmd/... ./projects/04-investment-community/starter/cmd/...
docker compose -f projects/04-investment-community/compose.yaml config
go test -p=1 -tags=integration ./projects/04-investment-community/reference/... -count=1 -v
go test -tags=acceptance ./projects/04-investment-community/acceptance -count=1 -v
python $wikiScript inventory --root .
python $wikiScript generate --root . --check
python $wikiScript check --root .
```

上述本地命令均通过；integration 使用独立 `investment_community_test` 且无 SKIP。acceptance 使用演示库与独立 API 端口，显式调用全部 21 个 operationId，并覆盖注册、入圈、发帖/更新、评论/回复通知、举报、隐藏、审计、恢复及删除。

完整历史提交 `5529de3ce2feac41f49dffd916152af7cfa6c66d` 已发布到远端功能分支，Git tree SHA 为 `959190d50cec42cef19bd1cad92c2f78a8aacdba`。GitHub Actions [run 32353166640](https://github.com/chasen2041maker/Go-own/actions/runs/32353166640) 三个 job 全部通过：

- `default`：Linux gofmt、项目默认测试、项目 vet/build；
- `mysql-and-acceptance`：真实 MySQL 全量 integration 无 SKIP，迁移、Seed、API 就绪和 21 操作 HTTP acceptance；
- `compose-cold-start`：从空环境 build/start 完整 Compose 栈，探测 API、Swagger 与 OpenAPI，执行 HTTP-only journey，并清理数据卷。

提交后总审计又补齐三项真实 MySQL 证据：通知插入故障时回复整笔回滚、8 个同 key 评论并发只产生一行/一条通知、8 个重复举报并发只产生一个 receipt。并发评论测试曾稳定复现 1213，根因已通过统一“帖子目标 → 幂等键 → 可选父评论”锁序修复；三项聚焦测试连续 5 次通过，全量 integration 随后再次无 SKIP 通过。

最终证据提交后，`stock-v1/s08-done` 与 `stock-v1/learner-start` 两个不可变 Tag 指向同一完成态。代码、契约、教学和交付门禁均无剩余必做事项。

## 4. 完成后的使用顺序

1. 学习者执行 `git switch -c learn-investment-community stock-v1/learner-start`；
2. 按 `docs/learning/stage-01.md` 到 `stage-08.md` 的 RED → GREEN 顺序在 `starter/` 亲手重写；
3. 维护者若规划 V2，另建规格、分支和迁移，不改写 V1 的阶段 Tag；
4. `main` 尚未合并，本次交付保留在 `codex/stock-community-governance`，由用户决定何时合并。

## 5. 已知环境与基线

- MySQL 测试容器：`go-own-community-integration`，`mysql:8.0.46`，本机端口 `13385`；DSN 只临时注入环境变量，不写入 Git。
- Reference API 当前未运行。
- 全仓 `go vet ./...` 有一个与本项目无关的既有失败：`practice/07-functions/answers/exercise.go:70` 不可达代码；项目范围 vet 仍需在每阶段验证。

## 6. 续接提示词

> 原创 Go 投资内容社区 V1 已完成，chunk-01～08、真实 MySQL、21 接口黑盒验收、Linux Compose 冷构建、Swagger/OpenAPI 与 GitHub CI 均有绿色证据。学习请从 `stock-v1/learner-start` 新建分支；扩展功能请另开 V2 规格，不修改 V1 不可变 Tag。

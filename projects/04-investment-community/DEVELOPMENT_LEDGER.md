# 原创投资内容社区开发总账

> 快照：2026-08-20（Asia/Taipei）
>
> 状态：实现与本地可执行门禁完成；Docker 冷启动/CI 待具备 Docker daemon 权限或推送后的 Linux CI 验证。
> 原创边界：只借鉴成熟社区的功能范围；Go 代码、接口、表、测试和虚构数据均独立设计，不复制公司源码、真实数据、密钥或品牌资产。

## 1. 唯一续接位置

| 项目 | 当前值 |
| --- | --- |
| Git 仓库 | `C:\company\own\Go-own` |
| 开发 worktree | `C:\Users\15234\.codex\visualizations\2026\08\19\01a0186e-78ea-7573-86a7-ef1dd016550d\stock-community-governance-3` |
| 分支 | `codex/stock-community-governance` |
| 完成态 HEAD | chunk-08 最终提交（以 `git log -1 --oneline` 为准） |
| 当前 WIP | 无业务 WIP；仅保留 Docker 冷启动/CI 外部验证 |
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
| 07 治理闭环 | `bc20705` | `stock-v1/s07-done` | 目标优先锁序、隐藏/恢复、治理版本、通知、审计、并发与 ABA 防护 |

文档、契约、八阶段教材及本总账的基线提交为 `d798a8b`。`stock-v1/starter` 永久保留为阶段一绿色历史快照，不再移动；主流程在最终 chunk-08 提交后创建新的不可变 `stock-v1/learner-start`，供学习者获得完整教材/契约和最小 starter 起点。

## 3. chunk-08 当前交付

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

上述命令均通过；integration 使用独立 `investment_community_test` 且无 SKIP。acceptance 使用演示库与独立 API 端口，显式调用全部 21 个 operationId，并覆盖注册、入圈、发帖/更新、评论/回复通知、举报、隐藏、审计、恢复及删除。Docker 镜像实际构建未作为成功证据：当前沙箱拒绝 Docker 命名管道/Buildx 锁访问；Compose 展开已通过，工作流会在 Linux 冷构建后检查 Swagger 镜像非 root、API/Swagger/OpenAPI 和完整 HTTP 旅程。

本分支最终提交后创建 `stock-v1/s08-done` 与 `stock-v1/learner-start` 两个不可变 Tag。剩余外部证据只有 Docker 冷启动与 GitHub CI；必须在可访问 Docker daemon 的环境复验，不能用静态配置成功替代。

## 4. 下一步严格顺序

1. 把本轮生成的 Git bundle 导入原仓库，使原 `codex/stock-community-governance` 快进到最终提交；
2. 在具备 Docker daemon 权限的环境执行 `docker compose up -d --build --wait`、Swagger/API 探测和 acceptance；
3. 推送或手动触发 GitHub Actions，确认 default、mysql-and-acceptance、compose-cold-start 三个 job；
4. 未经用户要求，不推送、不合并 `main`、不删除 MySQL 测试数据。

## 5. 已知环境与基线

- MySQL 测试容器：`go-own-community-integration`，`mysql:8.0.46`，本机端口 `13385`；DSN 只临时注入环境变量，不写入 Git。
- Reference API 当前未运行。
- 全仓 `go vet ./...` 有一个与本项目无关的既有失败：`practice/07-functions/answers/exercise.go:70` 不可达代码；项目范围 vet 仍需在每阶段验证。

## 6. 续接提示词

> 导入最终 Git bundle 后，在 `codex/stock-community-governance` 继续。chunk-01～08 的实现、复审与本地非 Docker 门禁已完成；只需在有 Docker daemon 权限的环境补跑冷构建/Swagger/acceptance，并确认 GitHub Actions。未经用户要求不要推送或合并。

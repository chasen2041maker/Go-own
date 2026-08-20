---
title: '原创投资内容社区与治理系统'
type: 'feature'
created: '2026-08-19'
status: 'in-progress'
review_loop_iteration: 0
baseline_commit: '45d8a53500d049eb14d3f39cf4fc4c923c2ef96b'
context:
  - '{project-root}/README.md'
  - '{project-root}/projects/README.md'
  - '{project-root}/docs/plans/2026-08-19-investment-community-design.md'
---

<frozen-after-approval reason="human-owned intent — do not modify unless human renegotiates">

## Intent

**Problem:** 当前仓库只有入门级 Go 项目，缺少一套能训练真实后端分层、MySQL、鉴权、事务、治理、通知和工程化的完整个人作品，也缺少可供用户亲手重写的独立学习空间。

**Approach:** 新增原创的投资内容社区与治理系统：完整参考实现和可运行 starter 只共享 OpenAPI/验收契约，按八阶段提供中文教学材料；业务闭环覆盖圈子、股票标签帖子、评论回复、通知、举报、管理员隐藏/恢复和操作审计。

## Boundaries & Constraints

**Always:** 全部代码、接口、表结构、状态机和测试重新设计；只使用虚构 Seed；使用 Go 1.26、`net/http`、MySQL、SQL Migration、JWT、密码哈希、结构化日志、Docker、OpenAPI 和 CI；reference/starter 不互相导入；关键注释解释设计原因与不变量；默认 `go test ./...` 不依赖 Docker并保持通过；真实 MySQL 集成测试覆盖约束、事务和并发。

**Ask First:** 接入真实行情或公司数据；增加 Vue/Flutter；加入 Redis、WebSocket、Push、消息队列、Kubernetes、微服务；修改或删除前三个既有项目；改变根 Module 路径。

**Never:** 复制或改写 Gulele/公司源码、目录、命名、契约、表结构、数据、密钥和品牌资产；记录密码或 Token；信任请求体中的用户身份；用 SQLite 代替 MySQL 集成验证；提交默认失败的 starter；为未来需求预建通用框架。

## I/O & Edge-Case Matrix

| 场景 | 输入/状态 | 预期行为 | 错误处理 |
| --- | --- | --- | --- |
| 加入并发帖 | 已登录用户加入公开圈子后提交带 1～5 个有效股票标签的帖子 | 成员关系唯一，帖子与标签同事务保存 | 非成员 `403`；无效标签 `422`；同幂等键异请求 `409` |
| 回复通知 | 成员回复同一帖子中的可见顶级评论 | 回复与通知同事务；回复自己不通知 | 已隐藏/删除或跨帖父评论 `409/422` |
| 举报治理 | 用户举报可见帖子/评论；管理员处理 pending 举报 | 忽略或隐藏目标，同时更新举报并写审计 | 非管理员 `403`；重复/并发处理 `409` |
| 内容恢复 | 管理员恢复被治理隐藏的内容 | 内容重新可见并写审计 | 未隐藏目标 `409` |
| 数据隔离 | 用户查询通知或删除内容 | 只能访问本人通知、本人内容 | 越权统一 `403` 或资源隐藏式 `404` |

</frozen-after-approval>

## Code Map

- `projects/04-investment-community/contracts/` -- OpenAPI 与 reference/starter 共用的业务验收场景。
- `projects/04-investment-community/reference/` -- 完整原创参考实现、迁移、Seed 与分层测试。
- `projects/04-investment-community/starter/` -- 可运行的健康检查骨架和用户亲手重写空间，不导入参考实现。
- `projects/04-investment-community/docs/` -- 架构、数据、治理、原创声明和八阶段教学文档。
- `projects/04-investment-community/acceptance/` -- 通过 build tag 启用的纯 HTTP 黑盒验收。
- `README.md`、`projects/README.md`、`docs/README.md`、`.repo-wiki/wiki-plan.toml` -- 新高级项目入口与文档治理。

## Tasks & Acceptance

**Execution:**

- [ ] 建立双轨骨架、配置、日志、统一错误、健康/就绪、迁移运行器和 MySQL Compose。
- [ ] 以 TDD 实现注册、登录、JWT、`user/admin` RBAC 和当前用户。
- [ ] 以 TDD 实现静态证券目录、公开圈子、成员关系、帖子与稳定分页。
- [ ] 以 TDD 实现评论、一级回复、软删除、通知和全部已读。
- [ ] 以 TDD 实现举报、治理状态机、隐藏/恢复、审计和并发冲突。
- [ ] 完成 OpenAPI、Seed、Swagger、黑盒演示、CI、集成测试和安全/生命周期收口。
- [ ] 完成原创说明、架构/数据/治理文档及八阶段中文学习路线。
- [ ] 更新仓库入口和 Repo Wiki，保留既有项目行为与文档历史。

**Acceptance Criteria:**

- Given 新环境，when 按 README 启动 Compose 和 Seed，then 可从 Swagger 跑通“注册→入圈→发帖→回复通知→举报→管理员隐藏→审计→恢复”。
- Given reference 或 starter，when 执行默认仓库命令，then 全部包可编译且测试不依赖外部数据库。
- Given MySQL 测试环境，when 执行 integration/acceptance 测试，then 唯一键、外键、事务回滚、行锁并发和权限矩阵均被验证。
- Given 用户从 starter 开始，when 按阶段文档学习，then 每阶段都有先写测试、预期失败、最小实现、验证命令、变式题和理解问题。
- Given 仓库变更，when 审查源码和 Git 历史，then 不包含公司实现、真实数据、密钥、品牌资产或与本项目无关的修改。

## Spec Change Log

## Design Notes

保持根 `go.mod`，使 `go test ./...` 继续覆盖全部项目。reference 与 starter 分别使用自己的 `internal/`，Go 编译器阻止二者互相导入；它们只通过非代码契约对齐。HTTP、日志、配置和服务生命周期使用标准库，只为 MySQL、JWT 和密码哈希增加必要依赖。

## Verification

**Commands:**

- `go test ./... -count=1` -- 默认测试全部通过且不需要 Docker。
- `go vet ./projects/04-investment-community/...` -- 新项目无静态检查报告。
- `go build ./projects/04-investment-community/reference/cmd/... ./projects/04-investment-community/starter/cmd/...` -- 全部入口可构建。
- `go test -tags=integration ./projects/04-investment-community/reference/... -count=1` -- 真实 MySQL 仓储、事务和并发测试通过。
- `go test -tags=acceptance ./projects/04-investment-community/acceptance -count=1` -- 完整 HTTP 业务闭环通过。
- `python C:/Users/15234/.codex/skills/maintain-repo-wiki/scripts/repo_wiki.py check --root .` -- 文档结构与路由检查通过。

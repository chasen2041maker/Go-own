# Repo Wiki 文档目录

> 本页由 `maintain-repo-wiki generate` 确定性生成。请修改机器事实源或 Wiki 配置后重新生成，不要直接维护本页。

事实源：`.repo-wiki/wiki-plan.toml`。

该目录按权威角色展示已登记页面。普通服务或模块 `README.md` 由 inventory 自动归类为 reference，按任务通过 context 路由加载。

## 统一入口

| 页面 | 目标 | 标签 |
| --- | --- | --- |
| [项目文档入口](../README.md) | 集中导航学习计划和项目设计文档 | `docs`, `learning` |

## 当前权威页面

| 页面 | 目标 | 标签 |
| --- | --- | --- |
| [原创投资内容社区设计](../plans/2026-08-19-investment-community-design.md) | 冻结高级个人项目的原创边界、架构、数据和治理不变量 | `architecture`, `go`, `mysql`, `governance` |
| [原创投资内容社区开发规格](../plans/spec-investment-community.md) | 约束 reference/starter 双轨、业务闭环和验收门槛 | `spec`, `go`, `learning` |
| [投资内容社区架构](../../projects/04-investment-community/docs/architecture.md) | 说明原创分层、事务、身份与并发边界 | `architecture`, `backend`, `mysql` |
| [投资内容社区验收场景](../../projects/04-investment-community/contracts/acceptance-scenarios.md) | 以 Given/When/Then 固定 21 个 API 的业务行为 | `contract`, `acceptance`, `api` |
| [投资内容社区开发总账](../../projects/04-investment-community/DEVELOPMENT_LEDGER.md) | 保存跨对话可恢复的提交、验证、未完成块和下一步 | `ledger`, `handoff`, `progress` |
| [投资内容社区数据模型](../../projects/04-investment-community/docs/data-model.md) | 解释表、约束、索引、Migration 和并发数据不变量 | `mysql`, `schema`, `data` |
| [投资内容社区治理设计](../../projects/04-investment-community/docs/governance.md) | 固定举报、隐藏、恢复、通知、审计和锁顺序 | `governance`, `transactions`, `audit` |
| [投资内容社区原创边界](../../projects/04-investment-community/docs/originality.md) | 证明实现、数据和教学材料不复制公司内部资产 | `originality`, `scope` |

## 专题参考

| 页面 | 目标 | 标签 |
| --- | --- | --- |
| [原创投资内容社区实施计划](../plans/2026-08-19-investment-community-implementation.md) | 记录八个依赖开发块、TDD 顺序和复审门禁 | `implementation`, `tdd`, `projects` |
| [投资内容社区八阶段学习路线](../../projects/04-investment-community/docs/learning/README.md) | 指导学习者在 starter 中按 RED/GREEN 顺序独立重写 | `learning`, `tdd`, `starter` |
| [Practice 练习任务总览](../../practice/TASKS-OVERVIEW.md) | 导航语法练习任务与参考答案 | `practice`, `learning` |
| [投资内容社区分块执行分析](../plans/2026-08-19-investment-community-implementation.chunked-analysis.md) | 记录实施块依赖、复审策略和执行顺序 | `implementation`, `chunks` |
| [阶段 01：工程地基](../../projects/04-investment-community/docs/learning/stage-01.md) | 指导配置、HTTP 生命周期和 Migration 重写 | `learning`, `stage-01` |
| [阶段 02：认证](../../projects/04-investment-community/docs/learning/stage-02.md) | 指导密码、JWT、数据库身份和 RBAC 重写 | `learning`, `stage-02` |
| [阶段 03：证券与圈子](../../projects/04-investment-community/docs/learning/stage-03.md) | 指导虚构目录、游标和成员关系重写 | `learning`, `stage-03` |
| [阶段 04：帖子](../../projects/04-investment-community/docs/learning/stage-04.md) | 指导帖子、证券标签、幂等和乐观锁重写 | `learning`, `stage-04` |
| [阶段 05：评论与通知](../../projects/04-investment-community/docs/learning/stage-05.md) | 指导评论回复、通知和事务边界重写 | `learning`, `stage-05` |
| [阶段 06：举报](../../projects/04-investment-community/docs/learning/stage-06.md) | 指导举报查重、权限和 author_deleted 收口 | `learning`, `stage-06` |
| [阶段 07：治理与审计](../../projects/04-investment-community/docs/learning/stage-07.md) | 指导治理状态机、锁序、版本和审计重写 | `learning`, `stage-07` |
| [阶段 08：工程化交付](../../projects/04-investment-community/docs/learning/stage-08.md) | 指导 Docker、CI、OpenAPI、验收和展示收口 | `learning`, `stage-08` |
| [文档目录（自动生成）](document-catalog.md) | 确定性列出仓库文档登记及职责 | `generated`, `docs` |
| [上下文路由（自动生成）](context-routes.md) | 确定性列出任务上下文与文档门禁路由 | `generated`, `routing` |

## 显式历史页面

| 页面 | 目标 | 标签 |
| --- | --- | --- |
| [Go 实战项目集重构方案](../plans/spec-refactor-learning-structure.md) | 约束练习归档、多项目结构和最小 Task API 起点 | `architecture`, `refactor`, `projects` |
| [项目集审查反馈落实规格](../plans/spec-address-review-feedback.md) | 约束 Task API 网络边界、HTTP 契约测试和三个项目的验收口径 | `review`, `testing`, `projects` |

## 历史范围

以下 glob 原位保留，但默认不进入 AI 上下文：

- `docs/plans/spec-go-learning-foundation.md`
- `docs/plans/spec-refactor-learning-structure.md`
- `docs/plans/spec-address-review-feedback.md`

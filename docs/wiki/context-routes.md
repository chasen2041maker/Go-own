# AI 上下文与变更路由

> 本页由 `maintain-repo-wiki generate` 确定性生成。请修改机器事实源或 Wiki 配置后重新生成，不要直接维护本页。

事实源：`.repo-wiki/wiki-plan.toml`。

每条路由把查询词或代码路径映射到最小阅读集、建议更新页和高信号门禁。普通变化只报告影响；命中 gate 时才阻断。

## `project-structure`

- 查询词：`项目结构`, `目录`, `重构`, `project`
- 代码范围：`README.md`, `practice/**`, `projects/**`, `docs/**`
- 优先读取：[项目文档入口](../README.md)、`@nearest-readme`
- 建议更新：[项目文档入口](../README.md)、`@nearest-readme`
- 高信号范围：—
- 命中后至少更新其一：—
- 命中后全部更新：—

## `investment-community`

- 查询词：`投资社区`, `治理`, `举报`, `审核`, `通知`, `MySQL`, `starter`, `reference`
- 代码范围：`projects/04-investment-community/**`, `docs/plans/*investment-community*`
- 优先读取：[原创投资内容社区开发规格](../plans/spec-investment-community.md)、[原创投资内容社区设计](../plans/2026-08-19-investment-community-design.md)、[投资内容社区架构](../../projects/04-investment-community/docs/architecture.md)、[投资内容社区八阶段学习路线](../../projects/04-investment-community/docs/learning/README.md)
- 建议更新：[项目文档入口](../README.md)、[原创投资内容社区开发规格](../plans/spec-investment-community.md)、[投资内容社区架构](../../projects/04-investment-community/docs/architecture.md)、[投资内容社区八阶段学习路线](../../projects/04-investment-community/docs/learning/README.md)、`@nearest-readme`
- 高信号范围：`projects/04-investment-community/**`
- 命中后至少更新其一：—
- 命中后全部更新：—

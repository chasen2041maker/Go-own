# Chunked Execution Analysis

**Source Plan:** `docs/plans/2026-08-19-investment-community-implementation.md`
**Generated:** 2026-08-19T14:00:00+08:00
**Status:** APPROVED

## Parsed Chunks

| Chunk | 名称 | 技能 | 复杂度 | 依赖 | 主要文件所有权 | 复审 |
| --- | --- | --- | --- | --- | --- | --- |
| chunk-01 | 工程地基与双轨骨架 | general Go | COMPLEX | none | platform/httpapi foundation、starter、migrations、go.mod | 规格+质量 |
| chunk-02 | 用户、认证与请求身份 | general Go | COMPLEX | 01 | user/auth/JWT/password | 规格+质量+安全 |
| chunk-03 | 静态证券、圈子与成员 | general Go + DB | COMPLEX | 02 | security/circle/community/seed | 规格+质量 |
| chunk-04 | 帖子、标签与信息流 | general Go + DB | COMPLEX | 03 | post/cursor/posts HTTP/store | 规格+质量+DB |
| chunk-05 | 评论、回复与通知 | general Go + DB | COMPLEX | 04 | comment/notification/interaction | 规格+质量+事务 |
| chunk-06 | 举报受理 | general Go + DB | MODERATE | 05 | report files only | 规格+质量 |
| chunk-07 | 治理、恢复与审计 | general Go + DB | COMPLEX | 06 | governance/audit files only | 规格+质量+并发 |
| chunk-08 | 工程化与教学交付 | docs/contracts/CI | COMPLEX | 07 | acceptance、Docker、docs、repo portals | 规格+质量+文档 |

## Dependency Graph

```text
chunk-01 → chunk-02 → chunk-03 → chunk-04
                                  ↓
chunk-08 ← chunk-07 ← chunk-06 ← chunk-05

contracts/docs 可提前并行起草 ───────────────↗
```

## Execution Order

| Step | Chunk | Depends On | 可并行工作 |
| ---: | --- | --- | --- |
| 1 | chunk-01 | - | 契约、教学文档起草 |
| 2 | chunk-02 | 01 | 契约、教学文档起草 |
| 3 | chunk-03 | 02 | 文档数据模型校对 |
| 4 | chunk-04 | 03 | OpenAPI 内容接口校对 |
| 5 | chunk-05 | 04 | 通知/事务文档校对 |
| 6 | chunk-06 | 05 | 治理场景校对 |
| 7 | chunk-07 | 06 | 黑盒场景准备 |
| 8 | chunk-08 | 07 | 最终复审 |

## 统一执行规则

1. 每个业务行为先产生可编译的失败测试并运行确认 RED，再写最小实现确认 GREEN。
2. 每个 chunk 使用新的实现代理，禁止修改其他 chunk 独占文件；共享装配文件只由当前最靠后的 chunk 修改。
3. 实现代理完成后先做规格复审，再做代码质量/安全/数据库复审；重要问题修复后必须重新运行聚焦测试。
4. 每个 chunk 独立提交 `chunk(chunk-0N): ...`，更新 `HANDOFF.md`，并在绿色提交上记录阶段 Tag。
5. 默认 `go test ./...` 不依赖 Docker；integration/acceptance 必须显式启用并在交付阶段真实运行。

## Plan Validation

- [x] 所有 chunk 有明确文件域和 Given/When/Then 验收。
- [x] 依赖是单向链，无循环依赖。
- [x] reference/starter/contract 的共享边界明确。
- [x] 治理、教学、原创、数据库和工程化要求均有落点。
- [x] 用户已明确批准全量开发，因此分析状态为 `APPROVED`。

# 原创投资内容社区与治理系统实施计划

**目标：** 在现有 Go 学习项目集中交付一套完全原创、可运行、可测试、可供用户在 starter 中逐阶段重写的投资内容社区与治理后端。

**架构：** 保留根 `go.mod`。完整实现位于 `reference/`，学习骨架位于 `starter/`，二者只共享 OpenAPI 与黑盒验收场景，不共享业务代码。参考实现采用 `httpapi → usecase → domain ← store/mysql` 的原创分层模块化单体。

**技术栈：** Go 1.26、`net/http`、MySQL 8、`database/sql`、go-sql-driver/mysql、golang-jwt/jwt/v5、x/crypto/bcrypt、OpenAPI 3.1、Docker Compose、GitHub Actions。

---

## chunk-01：工程地基与双轨骨架

**文件：**

- 创建 `projects/04-investment-community/reference/cmd/api/main.go`
- 创建 `projects/04-investment-community/reference/internal/platform/{config,server}.go` 及测试
- 创建 `projects/04-investment-community/reference/internal/httpapi/{router,response,middleware}.go` 及测试
- 创建 `projects/04-investment-community/starter/cmd/api/main.go`、`starter/internal/health/handler.go` 及测试
- 创建 `projects/04-investment-community/reference/migrations/001_initial.{up,down}.sql` 与 migration runner
- 修改根 `go.mod` / `go.sum`

**TDD：** 先写 `/healthz`、`/readyz`、统一错误、Request ID 和非法配置失败测试并确认 RED，再写最小服务器、配置与数据库 readiness；迁移 SQL 先由结构验证测试约束。

**验收：**

- Given reference/starter 服务，when 请求 `/healthz`，then 返回 200、JSON 和 Request ID。
- Given reference 数据库不可用，when 请求 `/readyz`，then 返回 503；可用时返回 200。
- Given 缺少安全配置，when 启动 reference，then 明确失败且日志不泄露密钥。
- Given 默认仓库测试，when 未启动 Docker，then reference/starter 仍可编译且单元测试通过。

**依赖：** 无。

---

## chunk-02：用户、认证与请求身份

**文件：**

- 创建 `reference/internal/domain/user.go`
- 创建 `reference/internal/usecase/auth.go` 及测试
- 创建 `reference/internal/store/mysql/users.go` 及 integration 测试
- 创建 `reference/internal/platform/{password,jwt}.go` 及测试
- 创建 `reference/internal/httpapi/auth.go` 及测试

**TDD：** 依次固定注册、重复邮箱、密码哈希、统一登录失败、过期/错误 JWT、disabled 用户和 admin/user RBAC 的失败行为，再实现最小用例与 Handler。

**验收：**

- Given 合法资料，when 注册并登录，then 密码只保存哈希且返回短期 Access JWT。
- Given 不存在邮箱或错误密码，when 登录，then 均返回同一 401 错误，不泄露账号状态。
- Given disabled 用户或过期 Token，when 访问受保护接口，then 请求被拒绝。
- Given普通用户，when 访问 admin 路由，then 返回 403；身份只能来自 JWT 与当前数据库用户。

**依赖：** chunk-01。

---

## chunk-03：静态证券、圈子与成员关系

**文件：**

- 创建 `reference/internal/domain/{security,circle}.go`
- 创建 `reference/internal/usecase/community.go` 及测试
- 创建 `reference/internal/store/mysql/{securities,circles}.go` 及 integration 测试
- 创建 `reference/internal/httpapi/{securities,circles}.go` 及测试
- 创建 `reference/cmd/seed/main.go`

**TDD：** 先写证券查询、公开圈子列表、`{"joined":true|false}` 成员状态的行为测试；用唯一键证明重复加入/退出安全重试，用对象身份测试证明不能替其他用户操作。

**验收：**

- Given 虚构 Seed，when 搜索证券代码/名称，then 只返回启用目录且不包含行情价格。
- Given 已登录用户，when 两次加入同一圈子，then 只有一条成员关系且结果可重试。
- Given 用户退出圈子，when 再尝试发内容，then 失去创作权限但历史内容仍保留。

**依赖：** chunk-02。

---

## chunk-04：帖子、证券标签与稳定信息流

**文件：**

- 创建 `reference/internal/domain/post.go`
- 创建 `reference/internal/usecase/posts.go` 及测试
- 创建 `reference/internal/store/mysql/posts.go` 及 integration 测试
- 创建 `reference/internal/httpapi/posts.go` 及测试
- 创建 `reference/internal/platform/cursor.go` 及测试

**TDD：** 先固定非成员发帖、1～5 个有效证券标签、Idempotency-Key 重放/冲突、游标分页、作者更新乐观锁和软删除权限；确认失败后写最小实现。

**验收：**

- Given 圈子成员，when 使用有效证券标签发帖，then 帖子和标签在同一事务保存。
- Given 同用户同幂等键，when 请求相同，then 返回原资源；请求不同返回 409。
- Given 两个并发版本更新，when version 相同，then 只有一个成功，另一个返回 409。
- Given 多页信息流，when 使用游标翻页，then 稳定排序且无重复/遗漏。

**依赖：** chunk-03。

---

## chunk-05：评论、一级回复与站内通知

**文件：**

- 创建 `reference/internal/domain/{comment,notification}.go`
- 创建 `reference/internal/usecase/interactions.go` 及测试
- 创建 `reference/internal/store/mysql/{comments,notifications}.go` 及 integration 测试
- 创建 `reference/internal/httpapi/{comments,notifications}.go` 及测试

**TDD：** 先覆盖顶级评论、一级回复、跨帖父评论、隐藏/删除资源、作者删除、顶级评论通知帖子作者、回复通知父评论作者、自操作不通知和事务回滚，再实现。

**验收：**

- Given 可见帖子/顶级评论，when 其他成员评论或回复，then `comment/reply` 通知与评论同事务提交。
- Given 通知写入失败，when 创建回复，then 回复也不落库。
- Given 父评论是回复或属于其他帖子，when 再回复，then 返回 422。
- Given 用户读取/标记通知，when 通知不属于本人，then 不可枚举或修改。

**依赖：** chunk-04。

---

## chunk-06：举报受理与管理员待办

**文件：**

- 创建 `reference/internal/domain/report.go`
- 创建 `reference/internal/usecase/reports.go` 及测试
- 创建 `reference/internal/store/mysql/reports.go` 及 integration 测试
- 创建 `reference/internal/httpapi/reports.go` 及测试

**TDD：** 先固定帖子/评论二选一目标、原因白名单、禁止自举报、重复举报先查重并返回原资源、非管理员不可查看待办以及待办稳定分页；再回补作者删除帖子/评论时同事务把相关 pending 举报收口为 `author_deleted`。

**验收：**

- Given 可见的他人内容，when 合法举报，then 创建一条 pending 举报。
- Given 同一用户重复举报同一目标，when 重试，then 返回原举报而不新增。
- Given 无效、隐藏、删除或自己的目标，when 举报，then 返回稳定业务错误。
- Given 非管理员，when 查看举报待办，then 返回 403。
- Given 作者删除带 pending 举报的内容，when 删除提交，then 内容软删除与全部 `author_deleted` 收口同时成功或同时回滚。

**依赖：** chunk-05。

---

## chunk-07：治理状态机、恢复与审计

**文件：**

- 创建 `reference/internal/domain/audit.go`
- 创建 `reference/internal/usecase/governance.go` 及测试
- 创建 `reference/internal/store/mysql/governance.go` 及 integration/concurrency 测试
- 创建 `reference/internal/httpapi/governance.go` 及测试

**TDD：** 先写 dismiss/hide、关闭同目标 pending 举报、隐藏作者通知、审计只追加和两个管理员并发决策只有一个成功的测试；恢复必须携带 `expected_moderation_version`，覆盖直接重试和 hidden→visible→hidden 后旧请求返回 ABA 冲突，再实现统一锁序事务。

**验收：**

- Given pending 举报，when 管理员 hide，then 举报、内容、同目标待办、通知和审计在一个事务内一致更新。
- Given 两个管理员并发处理，when 决策冲突，then 只有一个提交，另一个返回 409。
- Given hidden 内容，when 管理员用匹配治理版本恢复，then 可见性和治理版本原子更新并只写一条恢复通知/审计；旧版本绝不覆盖后续隐藏。
- Given审计查询，when 非管理员访问或尝试修改，then 被拒绝；正常记录只追加。

**依赖：** chunk-06。

---

## chunk-08：交付工程化与教学闭环

**文件：**

- 完成 `projects/04-investment-community/contracts/*`、`acceptance/api_test.go`
- 完成 `projects/04-investment-community/{README.md,.env.example,compose.yaml,Dockerfile}`
- 完成 `projects/04-investment-community/docs/**` 与 `scripts/demo.ps1`
- 创建 `.github/workflows/investment-community.yml`
- 修改 `README.md`、`projects/README.md`、`docs/README.md`、`.repo-wiki/wiki-plan.toml`、`.gitignore`

**TDD：** 先用黑盒验收测试表达完整业务闭环，再补齐组合装配、迁移/Seed、Docker、Swagger、演示脚本、日志与优雅关闭，使验收变绿。

**验收：**

- Given 全新 Docker 环境，when 一键启动并 Seed，then API、MySQL、迁移和 Swagger 可用。
- Given acceptance 测试，when 执行完整业务场景，then 注册到治理恢复与审计全部通过。
- Given starter，when 用户按八阶段文档学习，then 每阶段包含 RED、GREEN 边界、命令、变式题和理解题且不导入 reference。
- Given 仓库，when 执行格式、测试、项目 vet/build、Repo Wiki 和敏感信息检查，then 新项目全部通过且既有项目行为未变。

**依赖：** chunk-07；契约和教学文档可与早期代码并行准备，最终以实现校验。

---

## 主要风险

- 根模块新增依赖影响全仓：仅固定必要依赖版本并运行全仓测试。
- MySQL 与默认单测环境不同：默认测试使用 fake port，真实语义放入显式 integration tag。
- reference/starter 漂移：只共享 OpenAPI 与黑盒验收，不共享实现；文档明确每阶段目标。
- 治理并发不可靠：使用 `SELECT ... FOR UPDATE`、唯一约束、乐观锁和真实 MySQL 并发测试。
- 项目被误认为公司代码：原创声明、虚构 Seed、独立命名和 Git diff 审查作为交付门禁。

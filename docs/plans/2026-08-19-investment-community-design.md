# 投资内容社区与治理系统设计

## 1. 项目定位

本项目是一个原创的个人 Go 后端实战项目。它只参考成熟内容社区通常具备的功能范围，不复制任何公司内部项目的源码、目录、接口、表结构、命名、数据和品牌资产。

系统围绕一条完整业务链展开：用户注册登录后浏览并加入公开圈子，发布带静态证券标签的观点，参与评论和一级回复；评论与回复会生成站内通知；用户可以举报违规内容，管理员通过治理工作台忽略举报、隐藏或恢复内容，并在同一事务产生作者通知与不可缺失的操作审计。

第一版不接入真实行情、交易、直播、支付、AI、Push、WebSocket、Redis、Kubernetes 或历史数据迁移。股票目录只保存虚构的代码、名称和交易所，用于训练多对多关系和内容筛选，不提供投资建议。

## 2. 双轨学习交付

项目采用同一仓库中的双轨结构：

- `reference/` 是完整、经过测试的原创参考实现。
- `starter/` 只提供可运行健康检查和学习入口，不导入参考实现，也不包含故意失败的默认测试。
- `contracts/` 保存双方共同遵守的 OpenAPI 和黑盒验收场景，不共享业务代码。
- `docs/learning/` 按八个阶段说明前置知识、调用链、先写的失败测试、最小实现边界、变式练习和面试追问。

中文代码注释解释分层原因、权限来源、事务边界、状态不变量、唯一索引和并发行为，不逐行翻译普通 Go 语法。学习者在 `starter/` 中亲手实现，遇到困难时再对照 `reference/`。

## 3. 原创架构

参考实现使用一个 Go 模块中的模块化单体，HTTP 使用 Go 标准库 `net/http` 和增强后的 `ServeMux` 路由能力。代码按职责分成：

```text
cmd/api|migrate|seed
        ↓
internal/httpapi        协议解析、身份提取、DTO、状态码
        ↓
internal/usecase        用例编排、权限、事务、幂等
        ↓
internal/domain         实体、状态和业务错误
        ↑
internal/store/mysql    MySQL 仓储与事务实现
internal/platform       配置、日志、JWT、数据库、迁移
```

依赖只向内指向领域和用例。Handler 不写 SQL，仓储不返回 HTTP 状态码，领域对象不依赖数据库或网络。只引入三个必要依赖：MySQL Driver、JWT 实现和密码哈希库；路由、配置、日志、JSON 和服务器生命周期优先使用标准库。

## 4. 数据模型与查询边界

业务表为：`users`、`circles`、`circle_memberships`、`securities`、`posts`、`post_securities`、`comments`、`reports`、`notifications`、`admin_audit_logs`；工程表为 `schema_migrations`。

关键约束：

- 用户邮箱经“去首尾空白、全小写、严格单地址解析”后唯一；角色固定为 `user/admin`。对外公开用户摘要只包含 ID 和显示名，邮箱只在认证响应与 `/me` 返回给本人。
- 圈子成员以 `(circle_id, user_id)` 为主键，重复加入天然幂等。
- 全部业务 ID 使用 MySQL 有符号自增 `BIGINT`，Go/OpenAPI 使用最小值为 1 的 `int64`，从存储层杜绝超过 `math.MaxInt64` 的不可表示 ID。
- 每篇帖子最多关联五只启用的证券；关联表以 `(post_id, security_id)` 唯一。
- 评论最多一层回复，父评论必须属于同一帖子且本身不能是回复。
- 删除使用 `deleted_at`，治理使用独立 `visibility=visible/hidden`，避免把用户删除和管理员治理混为一类。
- 举报通过互斥的 `post_id/comment_id` 外键指向目标，避免无约束的多态 ID。
- 帖子、评论和通知列表必须分页并使用稳定的 `(created_at,id)` 顺序；对应查询建立最小复合索引。

数据库迁移是顺序、可重复执行的 SQL 文件。真实 MySQL 集成测试验证外键、唯一键、事务、行锁和并发，不使用 SQLite 冒充 MySQL 语义。

## 5. 核心接口与权限

接口分为七组：认证与当前用户、静态证券目录、圈子与成员、帖子、评论与回复、举报治理、站内通知与管理员审计。V1 固定为 21 个接口，以 `contracts/openapi.yaml` 为唯一接口事实源。

全局角色只区分普通用户和管理员；对象权限另外检查是否为圈子成员、内容作者、通知所有者。客户端提交的 `user_id` 永远不能决定数据归属，真实用户只能来自已验证 JWT。

治理状态机：

```text
pending → ignored
pending → resolved(hidden)
hidden content → visible(restored by admin)
```

管理员处理举报时先锁定目标内容，再按 ID 锁定同目标 pending 举报；隐藏时在同一事务更新可见性和治理版本、收口全部待办、写作者通知与审计。两个管理员并发时只有一次状态变化；相同决策可都返回既有 200，不同决策返回 409。顶级评论、回复、隐藏和恢复产生的通知都与对应事实同事务提交。

## 6. 错误、安全与运行

统一错误信封包含稳定错误码、用户可读消息和 Request ID。至少区分非法请求、未认证、无权限、不存在、冲突、业务校验失败和内部错误。JSON 请求拒绝未知字段、尾随数据、超大 Body 和不支持的媒体类型。

密码使用专用哈希算法；JWT 只接受配置的签名算法、发行者和受众，并校验过期时间。日志不记录密码、Token、Authorization Header 或完整内容正文。服务器配置读写超时、连接池和优雅关闭；`/healthz` 只证明进程存活，`/readyz` 还检查数据库。

## 7. 测试与作品交付

开发严格采用 RED → GREEN → REFACTOR。默认测试不依赖 Docker；MySQL 集成测试和 HTTP 黑盒验收通过 build tag 显式启用。测试层级包括领域规则、用例、Handler、MySQL 仓储、治理并发和完整业务场景。

八个阶段依次为：工程地基、认证、股票目录与圈子、帖子、评论与通知、举报受理、治理与审计、工程化收口。每阶段完成时 reference 和 starter 都可编译，默认仓库测试保持绿色。

最终交付还包括 Docker Compose、Seed、Swagger UI、自动演示脚本、CI、架构图、数据模型、治理说明、原创声明和八份学习文档。V2 才考虑 Refresh Session、圈主/入圈审批、点赞收藏、限流指标和薄 Vue 客户端。

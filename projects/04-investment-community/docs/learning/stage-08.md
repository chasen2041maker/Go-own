# 阶段 08：契约、集成验收与工程化收口

## 阶段目标

把前七阶段组合为可复现作品：新环境可启动，OpenAPI 可操作，真实 MySQL 证明约束与并发，黑盒测试跑通完整业务链，CI 默认快速且稳定。

## 1. 前置知识

- Go build tag、测试缓存、`go vet` 与可重复构建；
- Docker Compose、健康检查、依赖就绪与数据卷；
- OpenAPI 请求/响应、错误和安全方案；
- HTTP 黑盒测试与进程外边界；
- CI 分层：默认单元测试和显式 MySQL 集成任务；
- 连接池、服务器超时、优雅关闭和日志脱敏；
- Seed 幂等及演示数据安全。

## 2. 业务故事

评审者在干净环境按项目 README 启动 Compose、执行迁移和 Seed，从 Swagger 完成“注册→登录→GET /me→入圈→发帖→评论/回复→查看通知→举报→管理员隐藏→查看审计→恢复”。CI 同时证明默认测试不依赖 Docker、真实 MySQL 行为正确、外部契约没有漂移。

项目 README 与 Compose 是本阶段要生成并验证的交付物；前面阶段提到它们只是前向引用。在文件尚未存在、命令尚未实跑前，不能声称环境已经可复现。

## 3. 调用链

运行时：

```text
Compose 启动 MySQL并等待健康
  → migrate 按版本应用 SQL
  → seed 幂等写入虚构数据
  → api 通过 readyz 后接流量
  → Swagger UI 使用 contracts/openapi.yaml
```

黑盒验收：

```text
acceptance 测试启动/连接 API
  → 只按 OpenAPI 发 HTTP 请求
  → 保存动态 ID 与 Token（不写日志）
  → 跑完整普通用户和管理员场景
  → 校验状态、权限、隐藏/恢复与审计
```

CI 先跑格式、默认测试、vet 和 build；MySQL 服务就绪后再跑 `integration`，最后跑 `acceptance`。

## 4. 数据变化

本阶段不新增业务概念，只收口全部迁移和虚构 Seed：

- 迁移从 `schema_migrations` 顺序建立 `users`、`circles`、`circle_memberships`、`securities`、`posts`、`post_securities`、`comments`、`reports`、`notifications`、`admin_audit_logs`；
- Seed 只写虚构证券、圈子、演示账户和少量内容，重复运行不重复、不覆盖用户数据；
- 测试数据库每次获得可预测基线并在用例间隔离；
- 连接串和演示凭据只来自环境或明确的本地占位配置，不提交生产秘密。
- 全部业务主键、外键、OpenAPI ID 和黑盒测试动态 ID 都使用正数 int64；契约与数据库不再存在 UUID/BIGINT 双轨。

## 5. 先写的失败测试及为何失败

1. `TestOpenAPIContainsEveryRegisteredOperation`：路由与契约操作集合不一致即失败。最初失败，因为文档和路由由不同阶段累积；
2. `TestStrictJSONContractMatrix`：未知字段、尾随 JSON、错误媒体类型和超大 Body 得到稳定错误。最初失败，因为早期 Handler 可能各自解析；
3. `TestSecurityHeadersAndSensitiveLogRedaction`：密码、Token、Authorization 和完整正文不出现在日志。最初失败，因为尚无统一捕获测试；
4. `TestGracefulShutdownStopsNewRequestsAndBoundsExistingWork`：关闭后拒绝新流量且不无限等待。最初失败，因为只测过启动；
5. 集成测试矩阵：唯一键、外键、CHECK、业务 DML 事务回滚、稳定游标、DDL 部分失败后幂等重跑和治理行锁。最初失败，因为单元 fake 无法证明数据库事实；
6. 黑盒 `TestCommunityGovernanceJourney`：完整链路从空环境执行。最初失败，因为接口之间的 ID、权限或状态语义可能漂移；
7. `TestStarterDoesNotImportReference`：扫描 Go import/构建边界，发现跨轨依赖即失败。最初失败，因为复制实现是最容易出现的捷径；
8. CI 冷启动测试：不使用本机缓存和手工数据。最初失败，因为脚本可能隐含开发机状态。

## 6. GREEN 最大边界

只补齐 OpenAPI、Swagger UI、Compose、迁移/Seed、自动演示、CI、集成/验收测试、安全输入、日志脱敏、连接池和生命周期缺口。修复契约漂移时以已批准行为为准，不借收口阶段扩展产品。

不要增加 Refresh Session、圈主/审批、点赞收藏、限流指标、Vue/Flutter、真实行情、Redis、WebSocket、Push、消息队列、Kubernetes 或微服务。发现这些愿望时记入 V2 候选，不预建抽象。

## 7. 验证命令

在仓库根目录执行完整门禁。先用 `-list '^Test'` 确认 integration 与 acceptance 均至少发现一个测试；Go 的零匹配退出 0 不能算通过。

```powershell
go test ./... -count=1
go vet ./projects/04-investment-community/...
go build ./projects/04-investment-community/reference/cmd/... ./projects/04-investment-community/starter/cmd/...
$integration = go test -tags=integration ./projects/04-investment-community/reference/... -list '^Test'
if (-not ($integration | Select-String '^Test')) { throw '没有发现 integration 测试' }
go test -tags=integration ./projects/04-investment-community/reference/... -count=1
$acceptance = go test -tags=acceptance ./projects/04-investment-community/acceptance -list '^Test'
if (-not ($acceptance | Select-String '^Test')) { throw '没有发现 acceptance 测试' }
go test -tags=acceptance ./projects/04-investment-community/acceptance -count=1
$wikiScript = Join-Path $env:USERPROFILE '.codex/skills/maintain-repo-wiki/scripts/repo_wiki.py'
python $wikiScript check --root .
```

另外执行一次干净 Compose 启动、迁移、Seed、Swagger 手工闭环和优雅关闭。只有真实输出全部成功，才能声明完成。

## 8. 变式练习

- 暂时删除一个 OpenAPI 响应字段，确认契约测试能定位漂移；
- 连续运行迁移和 Seed 两次，比较第二次数据库变化；
- 关闭 MySQL，验证 healthz 仍存活、readyz 变未就绪且日志无连接串；
- 在验收旅程中交换两个普通用户 Token，确认作者、通知和治理权限不会串号；
- 给治理并发测试增加重复运行，消除依赖 `Sleep` 的偶发成功；
- 从空 Docker volume 开始复现一次，列出所有隐含本机依赖并删除。

## 9. 理解 / 面试问题

1. 为什么默认测试与 integration/acceptance 要分开？
2. OpenAPI 如何成为契约事实源而不是装饰文档？
3. 黑盒验收为什么不能导入 `reference/internal`？
4. SQLite 为什么不能替代 MySQL 行锁和约束测试？
5. Seed 幂等与迁移幂等有什么不同？
6. CI 为什么要验证冷启动而不仅是开发机测试？
7. healthz、readyz 和 Compose 健康检查分别服务谁？
8. 哪些日志字段有助排障但不会泄露敏感信息？
9. 如何证明 `starter` 是独立学习空间？

## 10. 中文注释落点

值得注释：build tag 为什么隔离外部依赖；黑盒验收为何只走 HTTP；Seed 为什么只能 upsert 固定虚构主数据而不能覆盖用户内容；关闭和连接池顺序；安全日志为何只保留摘要性上下文。

不值得注释：CI 每条命令的直译、Docker YAML 字段说明、Swagger 页面如何点击或测试清理的普通循环。

## 完成定义

默认、vet、build、integration 和 acceptance 全部有新鲜成功证据；干净环境能复现完整闭环；OpenAPI 与路由一致；日志无秘密；两条学习轨道不共享业务代码；仓库只含原创实现和虚构数据。

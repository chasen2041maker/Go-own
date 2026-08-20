# 阶段 01：工程地基

## 阶段目标

建立一个可启动、可诊断、可安全停止的 Go HTTP 服务，并准备可重复的 MySQL 迁移机制。完成后业务仍然为空，但后续每个阶段都有统一的配置、日志、错误和测试入口。

## 1. 前置知识

- Go module、包可见性、接口和显式依赖注入；
- `net/http`、`ServeMux`、`httptest` 与 `context.Context`；
- 环境变量、结构化日志和进程退出码；
- OS Signal、超时 Context 与优雅关闭；
- MySQL DDL 隐式提交、幂等 DDL，以及“迁移版本只能前进”的原因；
- 测试金字塔和 RED → GREEN → REFACTOR。

如果还不能独立写一个 `httptest` 请求，先用最小 Handler 练习状态码和 JSON，再进入本阶段。

## 2. 业务故事

运维人员需要区分“进程还活着”和“服务已经能接业务请求”：数据库短暂不可用时 `/healthz` 仍表示进程存活，`/readyz` 则必须报告未就绪。开发者需要在不同环境得到相同的配置校验、错误信封和 Request ID；部署前还要能按顺序、且只执行一次数据库迁移。

## 3. 调用链

存活检查：

```text
GET /healthz
  → Request ID 中间件
  → Health Handler（不访问数据库）
  → 200 + 简短状态
```

就绪检查：

```text
GET /readyz
  → Request ID 中间件
  → Readiness Handler
  → 带短超时的数据库 Ping 接口
  → 成功 200；超时或失败 503 + 统一错误信封
```

迁移入口：

```text
cmd/migrate
  → 读取并校验配置
  → 打开 MySQL 连接
  → 获取迁移互斥边界
  → 比较 schema_migrations 与顺序 SQL 文件
  → 逐条执行幂等 DDL
  → 全部成功后登记版本
```

`cmd/api` 只负责装配依赖和服务生命周期；Handler 不读取环境变量，迁移器也不导入 HTTP 包。

## 4. 数据变化

参考实现的首份 Migration 会一次性建立十张业务表，并使用
`schema_migrations(version, name, checksum, applied_at)` 记录版本；这样后续阶段可以专注业务代码。
本阶段只要求理解和重写 Migration runner、命名锁、checksum 与结构标记，不要求实现任何业务仓储。
一旦 001 被登记就禁止修改；学习过程中若主动改变结构，必须新增 002、003 等顺序 Migration，
不能重写旧文件来“修历史”。starter 也遵守同一规则。

- `version` 唯一，保证一个迁移只登记一次；
- `name` 保存迁移的描述性文件名；
- `applied_at` 便于诊断，不参与迁移排序；
- 重跑迁移应跳过已登记版本；
- MySQL DDL 会隐式提交，中途失败可能留下已经创建的表，但版本不得登记；修复后依靠 `CREATE TABLE IF NOT EXISTS` 安全重跑。

## 5. 先写的失败测试及为何失败

按顺序一次只写一个：

1. `TestHealthzDoesNotCallReadinessDependency`：注入一个调用即失败的 readiness fake，期望 `/healthz` 仍返回 200。最初失败，因为路由或 Handler 尚不存在；
2. `TestReadyzReturnsServiceUnavailableWithRequestID`：让 Ping fake 返回错误，期望 503、稳定错误码和 Request ID。最初失败，因为尚无就绪依赖与统一错误映射；
3. `TestJSONErrorEnvelopeIsStable`：构造领域外的测试错误，期望信封只暴露安全消息。最初失败，因为各 Handler 还会自行写响应；
4. `TestConfigRejectsMissingJWTSecret`：缺少必要配置时返回明确错误。最初失败，因为配置可能仍有不安全默认值；
5. `TestMigrationRunnerSkipsAppliedVersion`：用迁移存储 fake 验证重复版本不再执行。最初失败，因为迁移顺序和登记协议尚未定义；
6. `TestMigrationFilesRejectDuplicateVersion`：两个文件使用相同版本时必须在执行 SQL 前失败。最初失败，因为文件发现逻辑尚未校验版本唯一；
7. 集成测试 `TestFailedDDLDoesNotRecordMigrationVersion`：让中间语句失败，期望版本不登记，同时允许此前 DDL 已因隐式提交而存在。最初失败，因为运行器错误地把 DDL 当成可整体回滚；
8. 集成测试 `TestMigrationRetryAfterPartialDDL`：修复失败点后重跑，幂等 CREATE 收敛并最终登记一次版本。最初失败，因为 SQL 还不能安全重入。

每个 RED 都要检查失败信息。若失败原因是端口被占用或 Docker 未启动，应先修测试环境，不能把它当作业务 RED。

## 6. GREEN 最大边界

只实现：配置加载与校验、JSON 日志、Request ID、恢复中间件、统一错误信封、`/healthz`、`/readyz`、HTTP 超时、优雅关闭、数据库连接装配和迁移运行器。可先连接专用本地 MySQL；可重复 Compose 在阶段 08 作为统一交付入口补齐。

本阶段不要实现用户、JWT、通用 CRUD 框架、Repository 基类、自动依赖注入容器或可插拔数据库。`/healthz` 不查询数据库；`/readyz` 不执行迁移。迁移失败应由独立入口清楚退出；不能宣称 MySQL DDL 文件具有整文件事务回滚能力。

## 7. 验证命令

下面的命令会实际执行 `-list` 并检查非空；找不到匹配测试时直接抛错，不会继续把 `-run` 的零匹配当成通过。

```powershell
$unitPattern = 'Test(Healthz|Readyz|JSONErrorEnvelope|Config|Migration)'
$unitList = go test ./projects/04-investment-community/starter/... -list $unitPattern
if ($LASTEXITCODE -ne 0) { exit $LASTEXITCODE }
if (-not ($unitList | Select-String '^Test')) { throw '阶段 01 没有匹配的单元测试' }
go test ./projects/04-investment-community/starter/... -run $unitPattern -count=1
go test ./projects/04-investment-community/starter/... -count=1
go test ./... -count=1
go vet ./projects/04-investment-community/starter/...
```

真实 MySQL 准备好后再运行迁移集成测试：

```powershell
$integrationPattern = 'Test(Migration|FailedDDL)'
$integrationList = go test -tags=integration ./projects/04-investment-community/starter/... -list $integrationPattern
if ($LASTEXITCODE -ne 0) { exit $LASTEXITCODE }
if (-not ($integrationList | Select-String '^Test')) { throw '阶段 01 没有匹配的集成测试' }
go test -tags=integration ./projects/04-investment-community/starter/... -run $integrationPattern -count=1
```

手工停止服务一次，确认新请求停止进入、在途请求有截止时间、进程最终退出。

## 8. 变式练习

- 给 readiness fake 增加“超过截止时间才返回”的情况，证明 Handler 不会无限等待；
- 写一个重复的 Request ID 请求，决定是接受合法上游 ID 还是总由服务生成，并固定测试；
- 添加一个没有必填配置的测试，确保错误指出变量名但不打印秘密值；
- 模拟第二个迁移失败，验证第一个已提交、第二个未登记；
- 比较“启动 API 时自动迁移”和“独立 migrate 命令”的失败半径，写下选择理由。

## 9. 理解 / 面试问题

1. 为什么 liveness 不应依赖数据库，而 readiness 应依赖？
2. HTTP 读写超时分别防御什么问题？
3. 为什么优雅关闭需要截止时间，不能无限等？
4. MySQL DDL 隐式提交后，运行器如何让中断重跑保持安全？
5. 为什么默认单元测试不能要求 Docker？
6. Request ID 应在哪一层生成和回传？
7. 配置错误为什么应在启动时失败，而不是第一次请求时失败？

## 10. 中文注释落点

值得注释：为什么 `/healthz` 故意不查库；为什么 readiness 使用独立短超时；为什么迁移只有全部语句成功才登记版本；为什么 DDL 依靠幂等重跑而不是虚构整文件回滚；为什么关闭顺序先停接流量再关数据库。

不值得注释：调用 `ListenAndServe`、读取一个环境变量、判断 `err` 或写 JSON 的逐行过程。

## 完成定义

默认测试不需要 MySQL并全部通过；服务能区分存活与就绪；错误信封总有 Request ID；失败迁移不登记版本且修复后可安全重跑；停止信号不会让新请求继续进入。

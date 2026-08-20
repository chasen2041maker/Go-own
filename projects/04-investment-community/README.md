# 原创投资内容社区与治理系统

这是一个用于学习 Go 后端工程的个人项目。系统围绕“加入圈子 → 发布带证券标签的观点 → 评论/回复 → 站内通知 → 举报 → 管理员治理与审计”设计，但所有代码、接口、表结构、状态机、测试和教学材料都从本项目的规格独立推导。

本项目只借鉴内容社区普遍具有的**功能类别**，不模仿或改写任何公司、客户或商业产品的源码、目录、命名、接口、数据库、数据、密钥和品牌资产。证券、账户、圈子与帖子示例必须完全虚构；项目不接真实行情、不提供交易能力，也不构成投资建议。详细边界见[原创性声明](docs/originality.md)。

> **当前状态**：项目正在按八个阶段开发。`starter` 已提供 `/healthz` 重写起点；`reference` 已完成并验证工程地基、真实 MySQL Migration、注册/登录/`/me`、虚构证券目录、公开圈子与加入/退出成员状态。`contracts/openapi.yaml` 描述的是最终 V1 的 21 个业务操作，尚未实现的帖子、互动与治理路由不能视为可用。Docker Compose、Swagger 和 HTTP acceptance 属于阶段 08；对应文件与测试实际交付前，本文不会把它们写成当前可用能力。

跨对话续接以 [开发总账](DEVELOPMENT_LEDGER.md) 为准；总账明确区分已提交完成、未提交资产和正在开发的半成品。

## 双轨结构：自己写，不照抄

```text
contracts/       reference 与 starter 共同遵守的外部契约
docs/            架构、数据、治理、原创边界与中文学习路线
reference/       独立开发的参考实现；用于核对约束，不是复制模板
starter/         留给学习者亲手实现的工作区，默认测试始终应为绿色
acceptance/      阶段 08 才交付的纯 HTTP 黑盒验收（当前尚不可用）
compose.yaml     阶段 08 才交付的本地编排文件（当前尚不可用）
```

`reference` 与 `starter` 不互相导入业务代码，只共享 [OpenAPI](contracts/openapi.yaml) 和[验收场景](contracts/acceptance-scenarios.md)。推荐这样使用：

1. 先读本阶段文档和外部契约，自己画出调用链；
2. 在 `starter` 中先写一个能因“能力尚缺”而失败的测试；
3. 只写让当前测试通过的最小实现，然后保持仓库默认测试绿色；
4. 卡住时先看 `reference` 的测试名称和分层职责，不同时展开全部源码；
5. 关掉参考文件，用自己的类型和命名重写；最后以 HTTP 契约证明行为一致。

如果 `starter` 必须导入 `reference/internal/...` 才能工作，说明学习隔离已经被破坏。

## 最终 V1 目标（不是当前完成清单）

完成八个阶段后，V1 将覆盖：注册、登录、可信 JWT 身份、静态虚构证券目录、公开圈子与成员关系、带 1～5 个证券标签的帖子、一级回复、站内通知、用户举报、管理员忽略/隐藏/恢复，以及不可缺失的操作审计。

核心工程约束包括：身份只来自已验证 JWT，角色重新从数据库读取；帖子与标签、回复与通知、治理与审计分别保持事务原子性；作者软删除与管理员隐藏使用两条独立状态轴；列表以 `(created_at, id)` 稳定分页；MySQL 唯一键、外键、CHECK、事务和行锁由真实 MySQL 集成测试验证。

## 八阶段学习顺序与教学注释

顺序不要倒置。后一阶段会复用前一阶段已验证的身份、错误、事务和可见性规则。源码注释解释“为什么必须这样做”和“不这样做会破坏什么”，不逐行翻译 Go 语法。

| 阶段 | 学习主题 | 本阶段重点 | 值得写的中文教学注释 |
| --- | --- | --- | --- |
| [01：工程地基](docs/learning/stage-01.md) | 配置、错误信封、健康/就绪、迁移、优雅关闭 | 建立可启动、可诊断、可停止的服务边界 | 为什么 health 不查库、readiness 使用短超时、迁移全部成功后才登记、关闭时先停流量 |
| [02：认证与可信身份](docs/learning/stage-02.md) | 密码哈希、JWT、当前用户、RBAC | 建立后续权限唯一可信的身份来源 | 为什么错误密码与未知用户同口径、固定 JWT 算法/issuer/audience、授权不信任 role claim |
| [03：证券目录与公开圈子](docs/learning/stage-03.md) | 只读目录、圈子、成员关系 | 用复合主键收敛重复与并发入圈 | 为什么用户 ID 只能来自认证上下文、重复键在入圈场景代表幂等成功、停用证券仍保留历史 |
| [04：帖子、标签与稳定分页](docs/learning/stage-04.md) | 成员权限、多对多事务、幂等键、游标 | 原子写帖子与标签，稳定翻页 | 为什么规范化后计算请求哈希、唯一键是并发最终裁判、ID 必须打破相同时间戳平局 |
| [05：评论、一级回复与通知](docs/learning/stage-05.md) | 自引用、跨表事务、通知所有权 | 回复与必要通知同事务提交 | 为什么只允许顶级评论作父级、路径帖子必须复核、通知 SQL 始终带当前 `user_id` |
| [06：举报受理](docs/learning/stage-06.md) | 互斥目标、可举报性、管理员队列 | 举报只进入待审队列，不自动隐藏内容 | 为什么目标字段必须恰好一个非空、事务内重新确认可举报性、RBAC 必须先于查询 |
| [07：治理与审计](docs/learning/stage-07.md) | 状态机、统一锁序、治理版本、恢复 | 每次真实治理变化恰有通知/审计且旧重试不覆盖新事实 | 为什么先锁目标、审计必须同事务、相同动作可安全重试、治理版本阻止 ABA |
| [08：工程化收口](docs/learning/stage-08.md) | OpenAPI、Seed、Compose、真实 MySQL、黑盒验收、CI | 从干净环境复现完整闭环 | 为什么 build tag 隔离外部依赖、验收只走 HTTP、Seed 只能写虚构数据、日志必须脱敏 |

每阶段固定使用 `基线绿色 → RED → GREEN → REFACTOR → 阶段测试 → 全仓测试 → 变式练习与理解题`。总说明见[八阶段学习路线](docs/learning/README.md)。

## 当前可用命令

以下命令都从仓库根目录运行。需要 Go 1.26 或更高版本。

### 默认验证：不需要 Docker 或 MySQL

```powershell
go test ./projects/04-investment-community/... -count=1
go test ./... -count=1
go vet ./projects/04-investment-community/...
go build ./projects/04-investment-community/reference/cmd/... ./projects/04-investment-community/starter/cmd/...
```

默认测试只能证明当前已有代码；它不会自动证明带 `integration` 或 `acceptance` build tag 的最终行为。

### 运行 starter 健康检查

`starter` 当前不读取 `.env.example`，并固定只监听本机 `127.0.0.1:8081`。

```powershell
go run ./projects/04-investment-community/starter/cmd/api
```

另开一个 PowerShell 窗口：

```powershell
Invoke-RestMethod http://127.0.0.1:8081/healthz
```

### 运行 reference 工程地基

Go 程序只读取进程环境变量，不会自动加载 [.env.example](.env.example)。把变量配置到当前 shell、IDE 或秘密管理系统；不要提交本地密码或密钥。

下面只展示安全的本地设置方式。DSN 占位符必须替换为你自己的本地 MySQL 账号；JWT 密钥在本机随机生成，不写入仓库。

```powershell
$env:DATABASE_DSN = '<your-local-mysql-dsn>'
$env:JWT_ISSUER = 'investment-community'
$env:JWT_AUDIENCE = 'investment-community-api'

$secretBytes = [byte[]]::new(32)
$generator = [System.Security.Cryptography.RandomNumberGenerator]::Create()
$generator.GetBytes($secretBytes)
$generator.Dispose()
$env:JWT_SECRET = [Convert]::ToBase64String($secretBytes)

go run ./projects/04-investment-community/reference/cmd/api
```

另开窗口检查：

```powershell
Invoke-RestMethod http://127.0.0.1:8084/healthz
Invoke-RestMethod http://127.0.0.1:8084/readyz
```

`/healthz` 不访问数据库；数据库不可达时 `/readyz` 应返回 `503`，这不是 liveness 失败。迁移命令需要 `DATABASE_DSN` 指向一个可连接的 MySQL 数据库：

```powershell
go run ./projects/04-investment-community/reference/cmd/migrate
```

迁移会改变目标数据库结构。运行前确认 DSN 没有指向生产库或不应修改的数据库。

虚构 Seed 只允许在明确确认的本地/开发数据库运行，并把用户、证券与圈子作为一个事务提交：

```powershell
$env:SEED_CONFIRM = 'fictional-development-data'
$env:SEED_ADMIN_PASSWORD = '<local-admin-password>'
$env:SEED_USER_PASSWORD = '<local-user-password>'
go run ./projects/04-investment-community/reference/cmd/seed
```

Seed 失败会整体回滚；不要把上述确认值配置到生产部署，也不要提交密码。

## 配置字段

运行时字段与 `reference/internal/platform.Config` 保持一致：

| 环境变量 | 必填 | 默认值 | 规则/用途 |
| --- | --- | --- | --- |
| `HTTP_ADDR` | 否 | `127.0.0.1:8084` | 必须是合法的 `host:port`；默认只监听本机 8084 |
| `DATABASE_DSN` | 是 | 无 | MySQL Driver DSN；错误与日志不得回显完整值 |
| `JWT_SECRET` | 是 | 无 | 至少 32 字节且不能带首尾空白；必须由安全随机源生成 |
| `JWT_ISSUER` | 是 | 无 | JWT 发行者，签发和验证必须一致 |
| `JWT_AUDIENCE` | 是 | 无 | JWT 受众，防止 Token 被其他服务误用 |
| `HTTP_READ_HEADER_TIMEOUT` | 否 | `5s` | 正数 Go duration |
| `HTTP_READ_TIMEOUT` | 否 | `10s` | 正数 Go duration |
| `HTTP_WRITE_TIMEOUT` | 否 | `15s` | 正数 Go duration |
| `HTTP_IDLE_TIMEOUT` | 否 | `60s` | 正数 Go duration |
| `HTTP_SHUTDOWN_TIMEOUT` | 否 | `10s` | 正数 Go duration |
| `READINESS_TIMEOUT` | 否 | `2s` | 数据库 readiness 的独立正数 Go duration |
| `SEED_CONFIRM` | Seed 时必填 | 无 | 必须精确为 `fictional-development-data`，防止误跑 |
| `SEED_ADMIN_PASSWORD` | Seed 时必填 | 无 | 本地虚构管理员密码，至少 12 字符且不超过 bcrypt 72 字节 |
| `SEED_USER_PASSWORD` | Seed 时必填 | 无 | 本地虚构学习用户密码，规则同上 |

## 真实 MySQL integration 环境

当前 migration 集成测试会删除并重建它所管理的表，因此必须使用**专用、可丢弃的测试 schema**，绝不能与开发、演示或生产数据库共用。

| 环境变量 | 默认 | 安全规则 |
| --- | --- | --- |
| `COMMUNITY_TEST_DSN` | 无 | 只允许指向专用测试 schema |
| `COMMUNITY_TEST_ALLOW_RESET` | 非 `1` | 只有你确认测试库可被重置后才显式设为 `1` |

```powershell
$env:COMMUNITY_TEST_DSN = '<dedicated-disposable-test-schema-dsn>'
$env:COMMUNITY_TEST_ALLOW_RESET = '1'

$integration = go test -tags=integration ./projects/04-investment-community/reference/... -list '^Test'
if ($LASTEXITCODE -ne 0) { exit $LASTEXITCODE }
if (-not ($integration | Select-String '^Test')) { throw '没有发现 integration 测试' }
go test -tags=integration ./projects/04-investment-community/reference/... -count=1 -v
```

检查输出没有 `SKIP`，才能把本次运行当作真实 MySQL 证据。当前命令只覆盖已经存在的 integration 测试；最终的约束、事务、游标和治理并发矩阵要等对应阶段实现并有测试后才算完成。

## 阶段 08 完成后才可用的最终命令

> **现在不可作为可运行指令或完成证据。** 只有 `compose.yaml`、`reference/cmd/seed` 和 `acceptance/` 实际存在，并且下面命令获得新鲜成功输出后，才能使用本节。

```powershell
docker compose -f ./projects/04-investment-community/compose.yaml up -d --build
go run ./projects/04-investment-community/reference/cmd/migrate
go run ./projects/04-investment-community/reference/cmd/seed

go test ./... -count=1
go vet ./projects/04-investment-community/...
go build ./projects/04-investment-community/reference/cmd/... ./projects/04-investment-community/starter/cmd/...

$integration = go test -tags=integration ./projects/04-investment-community/reference/... -list '^Test'
if (-not ($integration | Select-String '^Test')) { throw '没有发现 integration 测试' }
go test -tags=integration ./projects/04-investment-community/reference/... -count=1

$acceptance = go test -tags=acceptance ./projects/04-investment-community/acceptance -list '^Test'
if (-not ($acceptance | Select-String '^Test')) { throw '没有发现 acceptance 测试' }
go test -tags=acceptance ./projects/04-investment-community/acceptance -count=1
```

Go 在 `-run` 或 build tag 下没有匹配测试时也可能退出 0，所以 integration/acceptance 必须先 `-list` 并确认发现测试。最终还要从空 Compose volume 手工跑通“注册 → 加入/退出圈子 → 发帖 → 评论/回复通知 → 举报 → 隐藏通知与审计 → 恢复通知与审计”，不能用契约文件存在或零匹配测试代替真实验收。

## 安全约定

- [.env.example](.env.example) 只列变量和公开占位符；空的必填值是故意的，避免已知密钥成为默认配置。
- 当前程序不自动加载 `.env`。若你创建本地环境文件，先确保它被 Git 忽略；不要把密码、JWT、Token、完整 DSN 或管理员凭据提交到仓库。
- 默认只监听回环地址。除非明确配置了网络和访问控制，不要改为 `0.0.0.0`。
- 日志不得记录密码、密码哈希、JWT、Authorization Header、完整连接串或完整内容正文。
- Seed 和测试数据只能使用 `example.test` 等虚构资料；测试库重置开关默认保持关闭。

## 文档入口

- [OpenAPI V1 契约](contracts/openapi.yaml)
- [跨接口验收场景](contracts/acceptance-scenarios.md)
- [系统架构](docs/architecture.md)
- [数据模型](docs/data-model.md)
- [治理、权限与审计](docs/governance.md)
- [原创性与素材边界](docs/originality.md)
- [八阶段学习路线](docs/learning/README.md)

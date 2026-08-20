# 八阶段学习路线

## 1. 怎么使用这套材料

本项目有两条彼此隔离的轨道：

- `starter/` 是你的工作区。每一阶段都先在这里写一个能说明行为的失败测试，再补最小实现；
- `reference/` 是通过同一外部契约的原创参考实现。卡住时先比较职责和调用链，最后才看具体写法；
- `contracts/` 是双方共同遵守的 OpenAPI 与黑盒验收，不共享业务代码。

不要把 `reference` 文件复制进 `starter`。学习目标是能从约束推导实现，而不是记住函数名。阶段文档只规定行为边界；内部类型和文件名可以不同，只要依赖方向、数据不变量和验收结果一致。

全路线只使用一套业务 ID：MySQL 有符号 `BIGINT`、Go 正数 `int64`、OpenAPI `integer/int64`（最小 1）。不要新增 UUID、公私双 ID 或字符串 ID 映射；0、负数和 int64 溢出输入都应在协议边界返回 `400 invalid_request`。

## 2. 推荐学习顺序

| 阶段 | 主题 | 本阶段建立的能力 | 主要数据变化 |
| --- | --- | --- | --- |
| [01](stage-01.md) | 工程地基 | 配置、错误信封、健康检查、迁移、优雅关闭 | `schema_migrations` |
| [02](stage-02.md) | 认证与身份 | 密码哈希、JWT、当前用户、RBAC 起点 | `users` |
| [03](stage-03.md) | 证券目录与圈子 | 只读目录、成员关系、数据库幂等 | `securities`、`circles`、`circle_memberships` |
| [04](stage-04.md) | 帖子 | 成员权限、多对多、事务、幂等键、游标分页 | `posts`、`post_securities` |
| [05](stage-05.md) | 评论与通知 | 一级回复、软删除、跨聚合事务、数据归属 | `comments`、`notifications` |
| [06](stage-06.md) | 举报受理 | 互斥目标、可举报性、待处理队列 | `reports` |
| [07](stage-07.md) | 治理与审计 | 状态机、行锁、并发冲突、不可缺失审计 | `admin_audit_logs` 与治理状态 |
| [08](stage-08.md) | 工程化收口 | OpenAPI、Seed、真实 MySQL、黑盒验收、CI、安全生命周期 | 全模型综合验证 |

顺序不能随意倒置：后一个阶段会复用前一阶段已经测试过的身份、错误、事务和数据可见性。如果一开始就做治理，失败时无法判断问题来自 JWT、成员权限、内容状态还是行锁。

## 3. 每阶段固定节奏

1. **基线绿色**：先运行默认测试，确认失败不是上一阶段遗留；
2. **RED**：只写一个描述可观察行为的测试，运行并确认它因能力缺失而失败；
3. **GREEN**：写恰好能通过当前测试的实现，不顺手预建下一阶段；
4. **REFACTOR**：保持测试绿色，消除重复、收紧命名和依赖方向；
5. **横向验证**：运行本阶段包测试，再运行仓库默认测试；
6. **记录理解**：完成变式练习，并用自己的话回答文末问题。

RED 必须“因正确原因失败”。编译错误可以是创建新 API 的第一步，但应尽快变成断言失败；测试夹具路径错误、Docker 没启动或随机端口冲突都不是有价值的 RED。

## 4. 测试边界

- 默认 `go test ./... -count=1` 不需要 Docker；
- 领域规则用纯单元测试；
- 用例通过小型 fake 验证权限、事务分支和调用顺序；
- Handler 用 `httptest` 验证 HTTP 状态、Header、严格 JSON 和错误信封；
- MySQL 唯一键、外键、事务、行锁和查询计划放在 `integration` build tag；
- 完整业务闭环放在 `acceptance` build tag，只通过 HTTP 调用；
- `starter` 不提交故意失败的默认测试。某阶段的 RED 只存在于你的学习分支，完成 GREEN 后再提交。

测试优先断言稳定错误码和状态变化，不依赖整段中文错误文案、内部函数调用次数或未承诺的 JSON 字段顺序。

## 5. 通用验证命令

在仓库根目录运行：

```powershell
go test ./projects/04-investment-community/starter/... -count=1
go test ./... -count=1
go vet ./projects/04-investment-community/...
go build ./projects/04-investment-community/reference/cmd/... ./projects/04-investment-community/starter/cmd/...
```

上述命令都从仓库根目录运行。需要真实 MySQL 时，先按 [项目 README](../../README.md) 启动 Compose MySQL，再用下方脚本创建独立的 `investment_community_test`；API 演示库 `investment_community` 不能用于会重置表的 integration。若你正在重做阶段 08，应先让黑盒测试因缺少环境正确失败，再补 Compose，不能把参考实现已有的运行结果冒充自己的 RED/GREEN 证据。

```powershell
docker compose -f ./projects/04-investment-community/compose.yaml up -d mysql --wait
$env:COMMUNITY_TEST_DSN = ./projects/04-investment-community/scripts/create-integration-schema.ps1
$env:COMMUNITY_TEST_ALLOW_RESET = '1'
$integration = go test -tags=integration ./projects/04-investment-community/reference/... -list '^Test'
if (-not ($integration | Select-String '^Test')) { throw '没有发现 integration 测试' }
$integrationOutput = go test -p=1 -tags=integration ./projects/04-investment-community/reference/... -count=1 -v
$integrationOutput
if ($LASTEXITCODE -ne 0) { exit $LASTEXITCODE }
if ($integrationOutput | Select-String '^--- SKIP:') { throw 'integration 出现 SKIP' }

$env:COMMUNITY_ADMIN_PASSWORD = 'LocalAdminPass!2026'
docker compose -f ./projects/04-investment-community/compose.yaml up -d --build --wait
$acceptance = go test -tags=acceptance ./projects/04-investment-community/acceptance -list '^Test'
if (-not ($acceptance | Select-String '^Test')) { throw '没有发现 acceptance 测试' }
$env:COMMUNITY_ACCEPTANCE_BASE_URL = 'http://127.0.0.1:8084'
$env:COMMUNITY_ACCEPTANCE_ADMIN_PASSWORD = $env:COMMUNITY_ADMIN_PASSWORD
$acceptanceOutput = go test -tags=acceptance ./projects/04-investment-community/acceptance -count=1 -v
$acceptanceOutput
if ($LASTEXITCODE -ne 0) { exit $LASTEXITCODE }
if ($acceptanceOutput | Select-String '^--- SKIP:') { throw 'acceptance 出现 SKIP' }
```

阶段文档给出的 `-run` 名称是建议的学习测试名；如果你使用不同命名，请保持测试表达的行为一致。Go 在 `-run` 零匹配时也可能以 0 退出，因此每次必须先用相同包、相同 build tag 和正则执行 `-list`，确认至少出现一个以 `Test` 开头的目标；零匹配绝不能作为完成证据。

## 6. 中文教学注释怎么写

注释的读者是假设已经看懂 Go 语法、但还不理解业务取舍的后来维护者。优先解释：

- 为什么身份只能来自验证后的上下文；
- 为什么某些写入必须在同一事务；
- 为什么唯一键、稳定次排序键或行锁不可省略；
- 为什么作者删除和管理员隐藏要分开；
- 为什么相反状态迁移必须冲突，而相同治理重试应安全返回既有结果。

不要写“遍历切片”“判断错误是否为空”“返回 JSON”之类逐行翻译。一个好检查方法是：删除注释后，如果代码仍完整说明“做了什么”，但读者会不知道“为何必须这样做”，这条注释就有价值。

每阶段文档都列出“中文注释落点”。只在这些关键边界附近写短注释，不把教学文章塞进源码；长解释留在本文档。

## 7. 如何对照参考实现

卡住时按以下顺序使用 `reference`：

1. 先只读 OpenAPI 和本阶段文档，自己画调用链；
2. 再看参考实现的测试名称，确认遗漏了哪条边界；
3. 只看对应层的职责，不同时展开所有目录；
4. 关掉参考文件，用自己的命名在 `starter` 重写；
5. 最后跑黑盒契约，证明不同内部实现仍有相同行为。

如果你的实现需要从 `starter` 导入 `reference/internal/...` 才能通过，说明学习隔离已经破坏。

## 8. 总完成标准

- 八阶段默认测试全部绿色且不依赖外部服务；
- 真实 MySQL 集成测试验证约束、事务和并发，而不是用 SQLite 替代；
- 黑盒场景跑通“注册→入圈→发帖→回复通知→举报→隐藏→审计→恢复”；
- 日志、测试夹具和文档不含密码、Token、真实个人资料或品牌资产；
- 能不看源码解释身份来源、三类事务边界、稳定分页和治理并发；
- `starter` 与 `reference` 没有业务代码依赖。

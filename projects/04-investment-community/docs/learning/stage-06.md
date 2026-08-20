# 阶段 06：举报受理与待处理队列

## 阶段目标

允许用户举报当前可见的帖子或评论，并让管理员查看稳定分页的待处理队列。本阶段只建立“线索进入系统”的边界，不提前实现治理决定。

## 1. 前置知识

- 帖子、评论的 `visibility` 与 `deleted_at` 双轴状态；
- JWT 当前身份和 `user/admin` 全局角色；
- 可空外键、MySQL CHECK 约束与三值逻辑；
- 事务内重新校验，避免“先查后写”竞态；
- 管理员私有列表和 `(created_at, id)` 游标分页；
- 输入长度限制、纯文本保存与输出转义。

## 2. 业务故事

已登录用户看到疑似违规内容后提交原因。系统记录真实举报者，并确保目标是一个当前可见、未删除的帖子或评论。管理员能查看待处理举报；普通用户无法访问队列。提交举报只是创建待审事实，不自动隐藏内容。

## 3. 调用链

```text
POST /api/v1/reports
  → 从 JWT Context 取得 reporter
  → 严格解析 target_type、target_id、reason、details
  → 先按 reporter + 目标查重；已有记录直接返回 200
  → 新举报才选择帖子或评论仓储并锁定目标
  → 事务内确认目标可见、未删除且不是举报者自己的内容
  → 写 reports(status=pending)

GET /api/v1/admin/reports?status=pending&target_type=post&cursor=...
  → 认证 → admin RBAC
  → 按 status、created_at、id 稳定查询
  → 返回最少必要的目标摘要，不复制完整敏感正文
```

Handler 负责拒绝未知目标类型；用例负责可举报性；数据库负责目标互斥和引用完整性。

## 4. 数据变化

新增 `reports`：`reporter_id`、互斥的 `post_id/comment_id`、`reason_code`、`details`、`status`、处理字段和时间字段。

`reports.id`、`reporter_id`、`post_id`、`comment_id`、`handled_by` 及 API `target_id` 全部使用 BIGINT/Go int64。协议字段 `reason` 与持久化 `reason_code` 使用同一枚举：`spam/harassment/misleading/illegal/other`，不做有损翻译。

- CHECK 保证 `post_id` 与 `comment_id` 恰好一个非空；
- 两个目标列分别有真实外键；
- 新记录固定为 `pending`，客户端不能提交处理状态或管理员；
- `(reporter_id,post_id)` 与 `(reporter_id,comment_id)` 唯一；同一用户重复举报返回原 receipt 和 `200`；
- `(status, created_at, id)` 索引支持待处理队列；
- 举报不会修改目标 `visibility` 或 `deleted_at`。
- 从本阶段开始，作者删除帖子/评论时必须在同一事务把该目标全部 pending 举报收口为 `resolved/author_deleted`，`handled_by` 为空且不写管理员审计。

## 5. 先写的失败测试及为何失败

1. `TestCreateReportUsesAuthenticatedReporter`：伪造 `reporter_id` 不能改变归属。最初失败，因为 DTO 仍能决定用户；
2. `TestCreateReportRequiresValidTargetType`：缺少或未知 `target_type` 返回 `422 validation_failed`。最初失败，因为协议枚举没有收紧；
3. `TestCreateReportHidesUnavailableTarget`：不存在、deleted 或 hidden 目标统一返回 `404 not_found`。最初失败，因为只检查目标 ID；
4. `TestCreateReportAlwaysStartsPending`：请求额外提交 `status=resolved` 必须返回 `400 invalid_request`。最初失败，因为直接绑定数据库模型；
5. `TestCreateReportDoesNotHideContent`：写入后目标可见性不变。最初失败，因为把举报误当治理；
6. `TestListReportsRequiresAdminBeforeQuery`：普通用户 403 且仓储不被调用。最初失败，因为只检查登录；
7. `TestDuplicateReportReturnsExistingReceipt`：同一用户再次举报同一目标返回 `200` 和原举报 ID，不需要 Idempotency-Key。最初失败，因为唯一键仍被翻译成冲突；
8. `TestListReportsCursorHandlesEqualTimestamps`：同时间戳分页无重复遗漏。最初失败，因为只按时间排序；
9. 集成测试 `TestReportTargetCheckConstraint`：零目标和双目标直接 SQL 均失败。最初失败，因为迁移未建立 CHECK；
10. 集成测试 `TestReportCreationLosesRaceWithContentHide`：内容在提交前变 hidden 时不留下 pending 举报。最初失败，因为可举报性只在事务外预查。
11. `TestCreateReportRejectsOwnContent`：举报自己的帖子/评论返回 `422 self_report_forbidden`。最初失败，因为只校验目标存在；
12. `TestDuplicateReportChecksUniqueRelationBeforeVisibility`：目标后来隐藏后重试仍返回原举报 200。最初失败，因为先检查目标可见性；
13. `TestAuthorDeleteClosesPendingReportsInSameTransaction`：帖子/评论删除与 `author_deleted` 举报收口要么同时成功，要么同时回滚。最初失败，因为旧删除用例不知道 reports；

## 6. GREEN 最大边界

只实现创建帖子/评论举报、管理员读取举报队列，以及对既有帖子/评论删除用例补上 `author_deleted` 收口。`target_type` 统一使用 `post/comment`；reason 映射到持久化 `reason_code`，details 做长度和 Body 大小限制；新举报固定 `pending`。举报创建不要求 Idempotency-Key，业务唯一键使重复请求安全返回既有结果。

不要实现自动关键词审核、举报计分、重复举报合并、封禁用户、管理员决策、隐藏、恢复、审计或通知管理员。也不要建立无外键的通用 `target_type + target_id` 表；第一版两个明确目标更安全。

## 7. 验证命令

下面的命令会实际执行 `-list` 并检查非空；找不到匹配测试时直接抛错，不会继续把 `-run` 的零匹配当成通过。

```powershell
$unitPattern = 'Test(CreateReport|ListReports)'
$unitList = go test ./projects/04-investment-community/starter/... -list $unitPattern
if ($LASTEXITCODE -ne 0) { exit $LASTEXITCODE }
if (-not ($unitList | Select-String '^Test')) { throw '阶段 06 没有匹配的单元测试' }
go test ./projects/04-investment-community/starter/... -run $unitPattern -count=1
go test ./projects/04-investment-community/starter/... -count=1
go test ./... -count=1
go vet ./projects/04-investment-community/starter/...
```

真实 MySQL 环境：

```powershell
$integrationPattern = 'Test(Report(Target|Creation|ForeignKey)|AuthorDeleteClosesPendingReports)'
$integrationList = go test -tags=integration ./projects/04-investment-community/starter/... -list $integrationPattern
if ($LASTEXITCODE -ne 0) { exit $LASTEXITCODE }
if (-not ($integrationList | Select-String '^Test')) { throw '阶段 06 没有匹配的集成测试' }
go test -tags=integration ./projects/04-investment-community/starter/... -run $integrationPattern -count=1
```

## 8. 变式练习

- 直接构造零目标和双目标 SQL，观察 CHECK 与应用校验各提供什么价值；
- 在目标校验与举报插入之间模拟隐藏，比较事务内锁定、条件写入两种方案；
- 给举报原因输入超长文本和 HTML，验证长度边界与输出转义责任；
- 用普通用户请求管理列表，确认在查询前失败；
- 同一用户连续举报同一内容，验证第二次返回 200、原举报 ID，数据库仍只有一行。

## 9. 理解 / 面试问题

1. 为什么举报不应自动隐藏内容？
2. 两个可空外键比通用多态 ID 安全在哪里？
3. 为什么应用校验后仍要 CHECK 和外键？
4. “目标存在”与“目标可举报”有什么区别？
5. 为什么举报者必须来自 JWT？
6. 管理员队列为何也需要稳定游标？
7. 如何避免目标刚被隐藏却仍写入 pending 举报的竞态？
8. 为什么重复举报可以依靠业务唯一键安全重试，而不要求 Idempotency-Key？

## 10. 中文注释落点

值得注释：为什么两个目标字段必须恰好一个非空；为什么事务内要重新确认可举报性；为什么创建举报不修改内容；为什么管理列表在查库前做 RBAC。

不值得注释：根据 target_type 进入分支、给 status 赋 `pending` 或扫描列表字段。

## 完成定义

举报者不可伪造，target_type 只有 post/comment，目标不可见统一 404，新记录总是 pending，重复举报返回既有 receipt，普通用户无法查看管理队列，真实 MySQL 约束能拒绝非法目标。

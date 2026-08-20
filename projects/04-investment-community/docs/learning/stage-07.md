# 阶段 07：治理状态机、行锁与审计

## 阶段目标

实现管理员 ignore 举报、隐藏内容、恢复内容和审计列表。通过两个管理员竞争同一举报的测试，掌握行锁、合法状态迁移、安全重试和“状态变化必须带审计”的原子性。

## 1. 前置知识

- OpenAPI 的 `pending → ignored/resolved` 与数据库 `pending → dismissed/resolved` 映射；
- 内容 `visibility=visible/hidden` 与独立 `deleted_at`；
- 内容 `moderation_version` 与 CAS/ABA 问题；
- MySQL 事务隔离、`SELECT ... FOR UPDATE` 或等价条件更新；
- 并发测试的两个独立连接与确定性同步；
- append-only 审计、Request ID 和敏感信息边界；
- `403`、`404`、`409`、`422` 的业务差异。

## 2. 业务故事

管理员查看 pending 举报后选择 ignore 或 hide。两个管理员同时提交不同决策时恰好一个改变状态，另一个得到 `409 report_already_decided`；相同决策重试则返回当前结果且不重复审计。恢复也按既有治理状态支持安全重试。

## 3. 调用链

```text
POST /api/v1/admin/reports/{reportId}/decision  (ignore | hide)
  → admin RBAC
  → BEGIN
  → 读取 report 快照定位目标
  → 先锁帖子或评论，再按 report.id 锁同目标 pending 举报
  → ignore：API report=ignored；数据库 status=dismissed/resolution_action=dismiss
     hide：content.visibility=hidden、moderation_version+1，同目标 pending 全部 resolved
  → hide 时写 content_hidden 通知；再写 admin_audit_logs
  → COMMIT

POST /api/v1/admin/content/{targetType}/{targetId}/restore
  → admin RBAC → 读取 expected_moderation_version → 锁定内容
  → 确认 target_type、未删除、hidden 且治理版本匹配
  → visibility=visible + moderation_version+1 + content_restored 通知与审计
```

`GET /api/v1/admin/audit-logs?target_type=post&cursor=...` 只读并使用 `(created_at, id)` 稳定分页；响应统一使用 `target_type/target_id`。普通内容查询始终过滤 hidden。

## 4. 数据变化

- ignore 时数据库 `reports.status=dismissed`、`resolution_action=dismiss`，API 投影为 `status=ignored/decision=ignore`；hide 时为 `resolved/hide`；两者同时写处理人、时间和说明；
- 被 hide 的 `posts/comments.visibility` 从 `visible` 变 `hidden`，并递增 `moderation_version`；
- restore 只在期望治理版本匹配时把未删除内容从 `hidden` 变 `visible`，再递增版本，不改历史举报；
- 新增 `admin_audit_logs`，记录管理员、动作、目标、相关举报、前后状态、Request ID 和时间；
- 审计只追加，没有更新/删除 API。
- `content_hidden/content_restored` 通知的 actor 为空，收件人是内容作者；它们与治理状态、审计同事务，安全重试不能重复通知。

`reportId`、`targetId`、`target_id`、`admin_id` 以及审计表所有 `*_id` 均为 BIGINT/Go int64。restore 路径中 ID 合法但 `targetType` 不是 `post/comment` 时固定返回 `400 invalid_request`。

## 5. 先写的失败测试及为何失败

1. `TestDecisionRejectsNonAdminBeforeTransaction`：普通用户 403 且事务未开启。最初失败，因为角色检查位置不明确；
2. `TestIgnoreChangesOnlyReportAndWritesAudit`：API 返回 ignored，数据库写 dismissed，内容不变，API 只看到一条 `report_ignored` 审计。最初失败，因为协议映射与决定分支混在一起；
3. `TestHideChangesContentReportAndAuditAtomically`：任一写入失败全部回滚。最初失败，因为三个 Repository 各自提交；
4. `TestDecisionRetryDistinguishesSameAndDifferentDecision`：相同既有决策返回 200 当前结果且不重复审计；不同决策返回 `409 report_already_decided`。最初失败，因为更新没有区分安全重试与冲突；
5. `TestRestoreUsesModerationVersionToRejectABA`：直接重试可返回 200 既有结果；恢复后再次隐藏会递增版本，旧重试必须返回 `409 moderation_version_conflict`。最初失败，因为只看 hidden/visible 会误覆盖新治理；
6. `TestRestoreWritesAuditWithoutReopeningReport`：恢复新增审计但不改原举报。最初失败，因为把恢复误做成“撤销处理”；
7. `TestAuditUsesAuthenticatedAdminAndServerState`：请求伪造管理员/前后状态无效。最初失败，因为直接绑定审计模型；
8. `TestRestoreRejectsInvalidTargetType`：restore 路径使用未知 targetType 时返回 `400 invalid_request`，不访问仓储。最初失败，因为路径枚举未统一校验；
9. 集成测试 `TestConcurrentDecisionHasOneStateChange`：两个连接提交不同决策，恰好一个成功、一个 `409 report_already_decided`、仅一条处理审计。最初失败，因为无行锁或条件更新；
10. 集成测试 `TestAuditInsertFailureRollsBackGovernance`：制造审计插入失败，内容和举报均保持原状。最初失败，因为审计在事务外；
11. `TestGovernanceNotificationSharesTransactionAndRetryBoundary`：隐藏/恢复通知与状态、审计同事务，重试不重复通知。最初失败，因为把通知当成事务后附加动作。

## 6. GREEN 最大边界

只实现 `ignore`、`hide`、`restore` 和审计列表；只治理帖子/评论。治理决策不要求 Idempotency-Key；恢复必须携带 `expected_moderation_version`。只有“当前 visible 且版本恰为 expected+1”可识别为直接重试并返回既有 200；更高版本一律冲突。

不要实现封号、自动审核、批量处理、申诉、审计编辑、内容改写、管理员层级或最终一致性消息。安全重试只复用已经存在的结果，不生成新的状态变化或审计。

## 7. 验证命令

下面的命令会实际执行 `-list` 并检查非空；找不到匹配测试时直接抛错，不会继续把 `-run` 的零匹配当成通过。

```powershell
$unitPattern = 'Test(Decision|Ignore|Hide|Restore|Audit)'
$unitList = go test ./projects/04-investment-community/starter/... -list $unitPattern
if ($LASTEXITCODE -ne 0) { exit $LASTEXITCODE }
if (-not ($unitList | Select-String '^Test')) { throw '阶段 07 没有匹配的单元测试' }
go test ./projects/04-investment-community/starter/... -run $unitPattern -count=1
go test ./projects/04-investment-community/starter/... -count=1
go test ./... -count=1
go vet ./projects/04-investment-community/starter/...
```

真实 MySQL 并发是本阶段完成门槛：

```powershell
$integrationPattern = 'Test(ConcurrentDecision|AuditInsertFailure|GovernanceRetry|GovernanceNotification)'
$integrationList = go test -tags=integration ./projects/04-investment-community/starter/... -list $integrationPattern
if ($LASTEXITCODE -ne 0) { exit $LASTEXITCODE }
if (-not ($integrationList | Select-String '^Test')) { throw '阶段 07 没有匹配的集成测试' }
go test -tags=integration ./projects/04-investment-community/starter/... -run $integrationPattern -count=1
```

## 8. 变式练习

- 用通道/屏障让两个事务在锁前就绪，避免靠 `Sleep` 猜并发时序；
- 故意把审计写到事务外，运行失败注入测试并描述破坏的事实；
- 重放相同 ignore/hide，再提交相反决策，验证 200 安全重试与 409 冲突的分界；
- 模拟 hidden→visible→hidden 后重放第一次 restore，证明治理版本阻止 ABA 覆盖；
- 删除一个已隐藏内容，再尝试恢复，解释 `deleted_at` 的优先级；
- 比较行锁与 `UPDATE ... WHERE status='pending'` 条件更新的可读性和错误判定。

## 9. 理解 / 面试问题

1. 为什么所有治理路径都必须先锁目标，再按 ID 锁举报？
2. 两个管理员并发处理时，怎样证明恰好只有一次状态变化和一条审计？
3. 审计为什么必须与状态更新同事务？
4. 恢复内容为什么不能把举报改回 pending？
5. 管理员为什么不能恢复作者已删除内容？
6. 为什么仅看当前 visibility 不能安全识别恢复重试？
7. 行锁范围过大有什么影响？
8. 审计为何不保存完整正文和 Token？

## 10. 中文注释落点

值得注释：统一锁顺序保护的并发不变量；审计为何在同一事务；治理版本如何阻止 ABA；作者删除为何返回 content_not_restorable。

不值得注释：`BEGIN/COMMIT` 字面含义、每次状态赋值或角色字符串比较。

## 完成定义

只有管理员能治理；每次真实状态变化恰有一条审计；相同治理重试与紧邻恢复重试不重复审计；旧恢复版本绝不覆盖后续隐藏；不同决策返回 409；失败完整回滚。

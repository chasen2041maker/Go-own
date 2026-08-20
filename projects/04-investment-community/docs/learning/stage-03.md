# 阶段 03：证券目录与公开圈子

## 阶段目标

实现静态证券目录、公开圈子列表和幂等入圈，为“只有成员可以创作”建立可靠对象权限。重点是把目录只读、身份归属和数据库唯一约束组合起来。

## 1. 前置知识

- 阶段 02 的认证中间件和当前用户 Context；
- MySQL 主键、外键、复合主键与重复键；
- 只读查询 DTO 与领域实体的区别；
- 用例层对象权限和 Repository 窄接口；
- 幂等操作与“重复请求返回同一最终状态”；
- Seed 可重复执行的基本策略。

先能解释“已登录”不等于“是某圈子成员”，再开始本阶段。

## 2. 业务故事

已登录用户浏览虚构证券目录和公开圈子，通过 `joined=true/false` 设置本人成员状态。网络重试或重复点击不会创建两条成员记录，也不会返回随机错误。之后的发帖与评论只检查当前成员关系，不相信客户端声称自己已经加入。

## 3. 调用链

证券目录：

```text
GET /api/v1/securities
  → 认证中间件
  → List Securities Use Case
  → 查询 status=active 的静态证券并按 (code,id) 稳定排序
  → 返回虚构代码、名称、交易所
```

圈子与入圈：

```text
GET /api/v1/circles
  → List Circles Use Case → 返回公开圈子

PUT /api/v1/circles/{circleId}/membership
  → 从 Context 取得当前 user ID
  → 严格解析 {"joined": true|false}
  → 事务内锁定 active 圈子行
  → joined=true：尝试插入；新建或重复键都返回 joined=true 和原 joined_at
  → joined=false：删除本人关系；存在或已不存在都返回 joined=false、joined_at=null
```

Handler 不接收能决定归属的 `user_id`。Repository 只报告“已存在”等数据事实，由用例决定其幂等语义。

## 4. 数据变化

新增三张表：

- `securities`：`market`、`code`、`name`、`status`、`created_at`，`(market, code)` 唯一；
- `circles`：唯一 `slug` 和 `name`、描述、`active/archived` 状态和时间字段；
- `circle_memberships`：`circle_id`、`user_id`、`joined_at`，复合主键 `(circle_id, user_id)`。

`securities.id`、`circles.id`、`circle_id` 和 `user_id` 均为 BIGINT/Go int64；API 的 `security_id`、`circleId` 也使用 int64，不接受 UUID。

所有证券和圈子数据均为虚构 Seed。`market` 使用 VARCHAR 保存原创市场标识；`status=inactive` 的证券不再提供给新内容使用，但不删除记录。第一版仅列出 active 公开圈子，不预建申请状态、圈主或成员角色。

## 5. 先写的失败测试及为何失败

1. `TestListSecuritiesReturnsOnlyActiveInStableOrder`：混入 inactive 证券和相同代码场景，期望只返回 active 记录并按 `(code,id)` 排序。最初失败，因为查询过滤与次排序键尚未定义；
2. `TestListCirclesDoesNotExposeInternalFields`：响应只含契约允许字段。最初失败，因为直接序列化数据库模型；
3. `TestSetMembershipUsesAuthenticatedUser`：请求携带伪造 `user_id` 会因未知字段被拒绝，仓储只能收到 Context 用户。最初失败，因为 DTO 与身份边界尚未收紧；
4. `TestJoinMissingCircleReturnsNotFoundWithoutInsert`：圈子不存在时不能调用成员写入。最初失败，因为用例只有盲目插入；
5. `TestJoinCircleTreatsExistingMembershipAsSuccess`：仓储返回已存在事实时，HTTP 仍返回 200 和原成员结果。最初失败，因为重复键被错误当成失败；
6. `TestJoinRequiresAuthentication`：无 Token 时在调用用例前返回 401。最初失败，因为路由未挂认证中间件；
7. `TestLeaveCircleIsIdempotentAndClearsJoinedAt`：首次与重复退出都返回 `joined=false, joined_at=null`，历史内容不删除。最初失败，因为只有插入路径；
8. 集成测试 `TestConcurrentJoinCreatesOneMembership`：两个连接同时入圈，最终仅一行且两次请求具有幂等成功语义。最初失败，因为 fake 无法模拟唯一键竞争；
9. 集成测试 `TestMembershipForeignKeysRejectUnknownIDs`：无效用户或圈子被外键拒绝。最初失败，因为迁移尚未建立约束。
10. 集成测试 `TestConcurrentMembershipJoinAndLeaveNeverMisreportCircleMissing`：加入与退出并发时都按自身操作返回合法状态，不能把成员行被删误报为圈子不存在。最初失败，因为 INSERT 与读取 joined_at 分属两个自动提交语句；

## 6. GREEN 最大边界

只实现启用证券列表、公开圈子列表和当前用户加入/退出圈子。加入由复合主键收敛，退出由目标用户条件删除收敛；Seed 使用虚构数据并可重复执行。

不要实现真实行情、证券搜索服务、自动同步、私密圈子、入圈审批、圈主、成员列表、邀请或圈子 CRUD。也不要为了下一阶段把“成员检查”做成能表达任意 ACL 的权限框架；一个清晰的 `IsMember`/查询接口足够。

## 7. 验证命令

下面的命令会实际执行 `-list` 并检查非空；找不到匹配测试时直接抛错，不会继续把 `-run` 的零匹配当成通过。

```powershell
$unitPattern = 'Test(ListSecurities|ListCircles|SetMembership|JoinCircle|LeaveCircle)'
$unitList = go test ./projects/04-investment-community/starter/... -list $unitPattern
if ($LASTEXITCODE -ne 0) { exit $LASTEXITCODE }
if (-not ($unitList | Select-String '^Test')) { throw '阶段 03 没有匹配的单元测试' }
go test ./projects/04-investment-community/starter/... -run $unitPattern -count=1
go test ./projects/04-investment-community/starter/... -count=1
go test ./... -count=1
go vet ./projects/04-investment-community/starter/...
```

真实 MySQL 环境：

```powershell
$integrationPattern = 'Test(ConcurrentJoin|ConcurrentMembership|MembershipForeignKeys|Securities)'
$integrationList = go test -tags=integration ./projects/04-investment-community/starter/... -list $integrationPattern
if ($LASTEXITCODE -ne 0) { exit $LASTEXITCODE }
if (-not ($integrationList | Select-String '^Test')) { throw '阶段 03 没有匹配的集成测试' }
go test -tags=integration ./projects/04-investment-community/starter/... -run $integrationPattern -count=1
```

手工分别重复提交 `joined=true` 与 `joined=false`，确认加入保留原 `joined_at`，退出始终返回 null，数据库至多一条关系。

## 8. 变式练习

- 把两个证券设置成相同 `code`、不同 `market`，验证唯一边界和展示排序；
- 把一个证券设为 `inactive`，确认目录不再返回但记录没有被删除；
- 让圈子读取成功、成员插入返回重复键，验证用例不把所有数据库错误都吞成成功；
- 并发发起十次入圈，统计最终行数并解释为什么应用预检查不能保证唯一；
- 退出后尝试创建新帖子，确认失去创作权限；再验证退出前历史帖子仍然存在。

## 9. 理解 / 面试问题

1. 为什么 `circle_memberships` 适合复合主键？
2. 幂等成功与“忽略所有插入错误”有什么区别？
3. 应用先查成员不存在再插入，为什么仍可能重复？
4. 为什么证券变为 inactive 不应删除历史关联？
5. 为什么请求体中的 user ID 不能决定入圈人？
6. Seed 怎样做到重复执行而不覆盖用户数据？
7. 公开圈子为什么暂时不需要 `is_public` 或审批状态？

## 10. 中文注释落点

值得注释：为什么重复键在“加入”用例中代表幂等成功；为什么加入/退出需在事务内锁同一圈子行；为什么只有已验证身份能决定 `user_id`；为什么停用证券保留历史记录；为什么复合主键同时是数据约束和并发收敛点。

不值得注释：遍历结果、扫描字段、调用 `INSERT` 或把 DTO 写成 JSON。

## 完成定义

目录只返回启用虚构证券，圈子列表稳定，入圈只属于当前用户，重复与并发入圈都收敛为一条 `circle_memberships`，无效外键由真实 MySQL 拒绝。

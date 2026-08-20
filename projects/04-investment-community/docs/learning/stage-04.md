# 阶段 04：帖子、标签与稳定分页

## 阶段目标

让圈子成员通过统一 `/posts` 资源发布带 1～5 个 active 证券标签的帖子，支持列表、详情、作者更新和软删除。通过这一阶段掌握多对多事务、创建幂等与游标分页。

## 1. 前置知识

- 阶段 03 的圈子成员关系和证券启用状态；
- MySQL 事务、唯一键、外键与回滚；
- 多对多关联表和批量写入；
- HTTP `Idempotency-Key` 请求头；
- 内容摘要/指纹和规范化输入；
- 基于 `(created_at, id)` 的游标分页；
- 软删除与管理员隐藏的区别。

若还会把“分页第 2 页”等同于固定 OFFSET，先推演并发插入后 OFFSET 如何造成重复。

## 2. 业务故事

用户加入公开圈子后，在 `POST /api/v1/posts` 的请求体提交 `circle_id`、标题、正文和一到五个虚构证券标签。移动网络重试相同创建请求时只创建一次；若误用同一个幂等键提交不同内容，返回 `409 idempotency_conflict`。作者还能更新可编辑帖子或软删除自己的帖子。

## 3. 调用链

创建帖子：

```text
POST /api/v1/posts + Idempotency-Key
  → JWT 当前用户
  → 严格解析 circle_id、title、body 与 security_ids
  → 校验圈子存在、当前用户是成员
  → 校验 1～5 个不重复的 active 证券
  → 规范化业务输入并计算 request_hash
  → 事务：检查/创建 posts，写入 post_securities
  → 同键同哈希回放原 201；同键异哈希返回 409 idempotency_conflict
```

列表和删除：

```text
GET /api/v1/posts?circle_id={circleId}&cursor=...
  → 解码不透明游标
  → 过滤 visibility=visible 且 deleted_at IS NULL
  → 按 created_at DESC, id DESC 取 limit+1
  → 返回 items 与 next_cursor

PATCH /api/v1/posts/{postId}
  → 请求携带客户端刚读取的 version
  → 校验作者、仍为圈子成员、visibility=visible 且 deleted_at 为空
  → 条件更新 WHERE id=? AND version=?，同事务替换证券并令 version+1

DELETE /api/v1/posts/{postId}
  → 加载帖子 → 校验当前用户是作者 → 只写 deleted_at
```

## 4. 数据变化

新增 `posts` 与 `post_securities`：

- `posts.circle_id`、`author_id` 使用外键；
- `title` 最长 120 字符，`body` 有明确长度上限；
- `visibility` 初始为 `visible`，`moderation_version` 与正文 `version` 分别初始为 1，`deleted_at` 初始为空；
- `(author_id, idempotency_key)` 唯一；
- `request_hash` 用于区分同键相同/不同创建请求；
- `post_securities(post_id, security_id)` 使用复合主键；
- 圈子帖子列表建立匹配可见性过滤与 `(created_at, id)` 排序的复合索引。

`posts.id`、`circle_id`、`author_id`、`post_id` 和 `security_id` 都是 BIGINT/Go int64；API 请求数组使用 `security_ids`，列表筛选使用单数 `security_id`。`request_hash` 应包含方法、路径、`circle_id`、规范化标题/正文和排序后的 `security_ids`；不能包含时间、Request ID 或 JSON 字段原始顺序。

## 5. 先写的失败测试及为何失败

1. `TestCreatePostRejectsNonMemberBeforeWriting`：非成员得到 403，帖子和标签仓储均未调用。最初失败，因为只检查登录；
2. `TestCreatePostRequiresOneToFiveDistinctActiveSecurities`：覆盖 0、6、重复、未知和 inactive 证券。最初失败，因为关联输入尚未校验；
3. `TestCreatePostRollsBackWhenTagInsertFails`：标签写入失败后不能留下帖子。最初失败，因为两次写入没有共享事务；
4. `TestCreatePostReplaysSameIdempotentRequest`：相同键和相同规范化载荷返回同一帖子。最初失败，因为每次都创建新行；
5. `TestCreatePostRejectsSameKeyWithDifferentHash`：相同键不同正文或标签得到 `409 idempotency_conflict`。最初失败，因为仅靠唯一键无法解释请求是否相同；
6. `TestListPostsUsesCreatedAtAndIDCursor`：创建相同时间戳的多条记录，连续翻页无重复、无遗漏。最初失败，因为只按时间或 OFFSET 分页；
7. `TestGetPostHidesDeletedOrGovernanceHiddenContent`：普通读取 deleted/hidden 均返回 `404 not_found`。最初失败，因为查询只按 ID；
8. `TestUpdatePostRequiresAuthorMembershipAndVisibleState`：非作者 `403 forbidden`，作者已离圈返回 `403 membership_required`，hidden 返回 `409 content_not_editable`；成功时证券替换与正文更新同事务。最初失败，因为缺少状态和事务边界；
9. `TestDeletePostRequiresAuthor`：其他成员返回 `403 forbidden`；作者成功为 204；重复删除返回 `404 not_found`。最初失败，因为仓储盲目更新；
10. `TestUpdatePostRejectsStaleVersion`：两个客户端使用同一旧 version 更新，只有一个成功，另一个返回 `409 version_conflict`。最初失败，因为普通 UPDATE 会丢失先提交内容；
11. 集成测试 `TestPostAndSecuritiesRollbackTogether`、`TestConcurrentIdempotencyCreatesOnePost` 与 `TestConcurrentVersionUpdate`。最初失败，因为 fake 不能证明事务、唯一键和条件更新竞争。

## 6. GREEN 最大边界

只实现 `/posts` 的创建/列表、详情、作者更新和软删除；支持 1～5 个 active 证券标签。`Idempotency-Key` 只要求在 createPost 使用；PATCH 和 DELETE 依靠资源状态与 version，不进入创建幂等域。

不要实现图片、附件、点赞、收藏、转发、热度排序、全文搜索、真实行情或物理删除。幂等键只覆盖创建帖子，不建设跨所有接口的通用幂等平台。游标只承载当前查询所需的排序值，签名/编码策略保持可验证但不过度抽象。

## 7. 验证命令

下面的命令会实际执行 `-list` 并检查非空；找不到匹配测试时直接抛错，不会继续把 `-run` 的零匹配当成通过。

```powershell
$unitPattern = 'Test(CreatePost|ListPosts|GetPost|UpdatePost|DeletePost)'
$unitList = go test ./projects/04-investment-community/starter/... -list $unitPattern
if ($LASTEXITCODE -ne 0) { exit $LASTEXITCODE }
if (-not ($unitList | Select-String '^Test')) { throw '阶段 04 没有匹配的单元测试' }
go test ./projects/04-investment-community/starter/... -run $unitPattern -count=1
go test ./projects/04-investment-community/starter/... -count=1
go test ./... -count=1
go vet ./projects/04-investment-community/starter/...
```

真实 MySQL 环境：

```powershell
$integrationPattern = 'Test(PostAndSecurities|ConcurrentIdempotency|ConcurrentVersion|PostCursor)'
$integrationList = go test -tags=integration ./projects/04-investment-community/starter/... -list $integrationPattern
if ($LASTEXITCODE -ne 0) { exit $LASTEXITCODE }
if (-not ($integrationList | Select-String '^Test')) { throw '阶段 04 没有匹配的集成测试' }
go test -tags=integration ./projects/04-investment-community/starter/... -run $integrationPattern -count=1
```

再用同一个幂等键连续发送：完全相同请求两次、只改变标签顺序一次、改变正文一次。前两种重放原结果，改变正文返回 `409 idempotency_conflict`；结果必须能由 request_hash 规范化规则解释。

## 8. 变式练习

- 让五个标签输入包含重复项，分别设计“去重后合法”与“重复即校验失败”的政策并比较可理解性；
- 在第一页读取后插入更新帖子，再读第二页，证明游标不会像 OFFSET 一样漂移；
- 让标签写入第 3 条时失败，验证整笔事务回滚；
- 让两个用户使用相同幂等键，说明为什么按作者作用域不会冲突；
- 设计按证券过滤的帖子列表，推导 `post_securities` 反向索引而暂不实现新 API。

## 9. 理解 / 面试问题

1. 为什么帖子与标签必须同事务写入？
2. 幂等键为什么还需要 request_hash？
3. request_hash 为什么要基于规范化载荷？
4. `(created_at, id)` 比只按时间稳定在哪里？
5. 为什么游标应是不透明值？
6. 非成员检查应发生在查询证券之前还是之后，取舍是什么？
7. `visibility` 与 `deleted_at` 为什么必须独立，怎样阻止错误恢复？
8. 应用检查标签数量后，数据库约束还能保护什么？

## 10. 中文注释落点

值得注释：为什么 request_hash 对 `security_ids` 排序后计算；为什么 `(author_id, idempotency_key)` 是并发最终裁判；为什么帖子和标签共用事务；为什么分页必须用 int64 ID 打破同时间戳平局；为什么作者删除不能改 `visibility`。

不值得注释：计算切片长度、开启事务、循环插入标签或 Base64 编解码的语法。

## 完成定义

非成员无法创建或更新；有效帖子始终有 1～5 个不重复、active 的证券关联；创建幂等重试不重复且异载荷返回 `409 idempotency_conflict`；更新原子替换证券；分页稳定；删除与治理隐藏状态清楚。

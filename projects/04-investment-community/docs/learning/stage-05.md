# 阶段 05：评论、一级回复与站内通知

## 阶段目标

实现顶级评论、一级回复、作者软删除、通知列表与全部已读。通过“回复与通知同事务”理解跨表原子性，并用所有者条件保护私有数据。

## 1. 前置知识

- 阶段 04 的成员权限、内容可见性和稳定游标；
- 自引用外键与树结构的基本概念；
- 事务内读取、校验和写入；
- 私有资源的对象权限与资源存在性隐藏；
- 幂等更新和条件更新；
- UTC 时间、可空 `read_at` 与稳定分页。

本项目只做一级回复。如果你准备递归加载任意深度评论树，说明已经超出本阶段边界。

## 2. 业务故事

圈子成员在可见帖子下发表评论，也可以回复一条可见顶级评论。顶级评论会通知帖子作者，回复会通知父评论作者；操作自己内容时不制造无意义通知。每个用户只能读取和标记自己的通知。作者可以软删除自己的评论，但管理员隐藏仍是独立治理状态。

## 3. 调用链

评论或回复：

```text
POST /api/v1/posts/{postId}/comments + Idempotency-Key
  → 当前用户与严格 DTO（body、可选 parent_comment_id）
  → 校验帖子可见、未删除且用户是圈子成员
  → 若有 parent_comment_id：映射为 parent_id，加载父评论并校验同帖、顶级、可见、未删除
  → 计算 request_hash
  → 事务：幂等检查/写 comments；顶级评论通知帖子作者，回复通知父评论作者
  → 同键同哈希回放原 201；同键异哈希返回 409 idempotency_conflict
```

通知：

```text
GET /api/v1/notifications?cursor=...
  → 查询条件强制 user_id = 当前用户
  → created_at DESC, id DESC 稳定分页

PUT /api/v1/notifications/read
  → 条件更新 user_id = 当前用户 AND read_at IS NULL
  → 返回 200、read_count 和同一 read_at
```

删除评论先校验作者，再设置 `deleted_at`；不会删除回复或通知历史。

## 4. 数据变化

新增 `comments`：

- `post_id`、`author_id` 为外键；
- `parent_id` 可空并自引用 `comments.id`；
- `parent_id=NULL` 为顶级评论，否则只能指向同帖、可见、未删除且自身无父级的评论；
- `visibility` 只允许 `visible/hidden`；作者删除只写 `deleted_at`，不改 visibility；
- `idempotency_key` 与 `request_hash` 只服务 createComment；
- 列表按 `(created_at, id)` 稳定排序。

`comments.id`、`post_id`、`author_id`、`parent_id` 和通知中的各 `*_id` 都是 BIGINT/Go int64。外部创建字段叫 `parent_comment_id`，进入持久化后对应 `comments.parent_id`。

新增 `notifications`：

- `user_id` 是通知所有者，`actor_user_id` 是触发者；
- `type` 使用 `comment/reply/content_hidden/content_restored` 持久化枚举；
- `post_id`、`comment_id` 保存相关内容；
- `read_at` 为空表示未读；
- 顶级评论使用 `type=comment`，回复使用 `type=reply`；actor 是评论者；对自己的帖子评论或回复自己的评论不插入通知。

普通外键无法保证“父评论属于同帖且父评论不是回复”，该规则由用例事务内校验。

## 5. 先写的失败测试及为何失败

1. `TestCreateTopLevelCommentRequiresMembershipAndVisiblePost`：非成员返回 `403 membership_required`；deleted/hidden 帖子返回 `404 not_found`。最初失败，因为只按帖子 ID 插入；
2. `TestReplyRejectsParentFromAnotherPost`：路径帖子与父评论 `post_id` 不同得到 422。最初失败，因为只验证父 ID 存在；
3. `TestReplyRejectsReplyAsParent`：父评论已有 `parent_id` 时拒绝，限制深度为一层。最初失败，因为自引用模型默认允许无限嵌套；
4. `TestReplyRejectsDeletedOrHiddenParent`：不可见父评论返回 `409 content_not_editable`。最初失败，因为查询未带状态条件；
5. `TestReplyAndNotificationCommitTogether`：通知写入失败时回复也不存在。最初失败，因为先提交回复再另写通知；
6. `TestTopLevelCommentNotifiesPostAuthor`：他人评论帖子产生 `comment` 通知，评论自己帖子不通知。最初失败，因为只处理回复通知；
7. `TestCreateCommentIdempotencyDoesNotDuplicateNotification`：同键同哈希回放原 201 和评论 ID，同键异哈希返回 `409 idempotency_conflict`。最初失败，因为 comments 尚无幂等边界；
8. `TestReplyToSelfCreatesNoNotification`：回复自己只写评论。最初失败，因为所有回复都创建通知；
9. `TestListNotificationsUsesAuthenticatedUser`：伪造查询参数不能读取他人通知。最初失败，因为仓储按客户端 ID 查询；
10. `TestReadAllOnlyTouchesCurrentUser`：两个用户混合数据时只更新本人未读项，并返回准确 read_count。最初失败，因为 UPDATE 缺少 user_id 条件；
11. `TestCommentCursorHandlesEqualTimestamps`：相同时间戳分页无重复遗漏。最初失败，因为排序不完整；
12. 集成测试 `TestReplyNotificationRollback`、`TestConcurrentCommentIdempotency` 和 `TestNotificationOwnershipUpdate`。最初失败，因为 fake 不能证明真实事务、唯一键和条件更新。

## 6. GREEN 最大边界

只实现顶级评论、一级回复、评论稳定列表、作者软删除、评论/回复通知、通知列表和全部已读。`Idempotency-Key` 只要求 createComment；读取、删除和 `PUT /notifications/read` 不进入创建幂等域。通知是站内拉取模型，不主动推送。

不要实现任意深度树、评论编辑、@提及、点赞、未读计数缓存、WebSocket、Push、邮件、消息队列或通知模板系统。回复与通知直接在单个 MySQL 事务完成，不为第一版引入异步最终一致性。

删除评论不级联物理删除回复；普通列表根据契约隐藏已删除正文或整项，但必须保持引用和审计可能性。

## 7. 验证命令

下面的命令会实际执行 `-list` 并检查非空；找不到匹配测试时直接抛错，不会继续把 `-run` 的零匹配当成通过。

```powershell
$unitPattern = 'Test(CreateTopLevelComment|Reply|ListNotifications|ReadAll|CommentCursor|DeleteComment)'
$unitList = go test ./projects/04-investment-community/starter/... -list $unitPattern
if ($LASTEXITCODE -ne 0) { exit $LASTEXITCODE }
if (-not ($unitList | Select-String '^Test')) { throw '阶段 05 没有匹配的单元测试' }
go test ./projects/04-investment-community/starter/... -run $unitPattern -count=1
go test ./projects/04-investment-community/starter/... -count=1
go test ./... -count=1
go vet ./projects/04-investment-community/starter/...
```

真实 MySQL 环境：

```powershell
$integrationPattern = 'Test(ReplyNotification|ConcurrentCommentIdempotency|NotificationOwnership|CommentForeignKey)'
$integrationList = go test -tags=integration ./projects/04-investment-community/starter/... -list $integrationPattern
if ($LASTEXITCODE -ne 0) { exit $LASTEXITCODE }
if (-not ($integrationList | Select-String '^Test')) { throw '阶段 05 没有匹配的集成测试' }
go test -tags=integration ./projects/04-investment-community/starter/... -run $integrationPattern -count=1
```

用两个普通用户手工验证：A 发顶级评论、B 回复后 A 收到通知；A 回复自己时通知数量不变；B 无法读取 A 的通知。

## 8. 变式练习

- 让通知插入返回错误，分别观察有事务与无事务时用户会看到什么；
- 构造跨帖父评论、回复的回复、已隐藏父评论三组表驱动测试；
- 同时执行两次“全部已读”，解释第二次为何可以是更新 0 行的幂等成功；
- 设计删除顶级评论后的回复展示政策，并区分“保留结构”与“泄露正文”；
- 给相同回复模拟并发重放，验证 comments 的 `(author_id,idempotency_key)` 如何让通知副作用最多一次。

## 9. 理解 / 面试问题

1. 外键为什么不能保证只有一级回复？
2. 为什么父评论必须与路径中的帖子再次比对？
3. 回复和通知为什么要同事务，而普通日志不需要？
4. 回复自己为何不创建通知？
5. 私有通知查询为什么每条 SQL 都要带 user_id 条件？
6. 全部已读如何设计成安全幂等操作？
7. 删除父评论时为什么不应级联物理删除回复？
8. 如果未来通知改异步，需要补偿哪些一致性问题？

## 10. 中文注释落点

值得注释：为什么只允许顶级评论作为父级；为什么路径 post ID 必须和父记录一致；为什么 createComment 幂等记录、回复和通知共用事务；为什么自回复跳过通知；为什么通知 SQL 仍强制当前 user_id。

不值得注释：判断 `parent_id` 是否为空、追加切片、设置 `read_at` 或循环扫描行。

## 完成定义

评论仅由成员在可见帖子下创建，回复最多一级且不能跨帖；回复与必要通知原子提交；通知只对所有者可见；作者删除和管理员隐藏保持独立。

# V1 验收场景与手写学习顺序

本文与 `openapi.yaml` 共同约束 reference 和 starter。OpenAPI 固定请求、响应和状态码；
本文固定跨接口行为、权限、事务与并发结果。实现可以不同，但不得改变这些可观察结果。

> 原创边界：所有邮箱、显示名、圈子和证券均为虚构测试数据。本项目不使用任何公司系统的
> 源码、接口、表结构、命名、数据或品牌资产。

## 1. 固定词汇和验收约定

- 统一领域词汇为 `securities`、`post_securities`、`reports`。前者是虚构证券目录，
  中间表只表达帖子和证券的多对多关系，后者是用户举报；不得再引入含义相同的
  第二套命名。
- 基础地址是 `/api/v1`。除注册和登录外，每个请求都带
  `Authorization: Bearer <JWT>`。
- 用户身份只来自已验证 JWT 的 `sub`。请求体里出现 `user_id`、`author_id`、
  `reporter_id`、`recipient_id` 或 `admin_id` 都是未知字段，必须拒绝。
- 所有业务 ID 都是 JSON integer / OpenAPI `int64`，最小值为 1；JWT `sub` 是同一正整数的
  十进制字符串。0、负数、小数、UUID 字符串和溢出值都不是合法业务 ID。
- 他人资料只允许 `{id,display_name}`；帖子、评论、通知、举报和审计响应不得泄露他人的
  `email`、`role` 或 `status`。只有认证响应和 `GET /me` 返回当前用户私有资料。
- 所有错误体都严格是 `{ "error": { "code", "message", "request_id", "details" } }`；
  `X-Request-ID` 与体内 `request_id` 相同。测试只按稳定的 `code` 作程序判断。
- 所有列表用不透明游标。第一页不传 `cursor`，下一页只回传 `page.next_cursor`；
  客户端不解析游标。`page.has_more=false` 时 `next_cursor` 必须为 `null`。
- 标注 `[HTTP]` 的场景由纯 HTTP 黑盒测试覆盖；`[MySQL]` 场景使用真实 MySQL；
  `[Source]` 场景通过源码或配置审查覆盖。默认 `go test ./...` 不依赖 Docker。

### 测试夹具

每次测试使用随机后缀创建 `alice_<suffix>@example.test` 与 `bob_<suffix>@example.test`，
显示名分别为 `Alice <suffix>` 与 `Bob <suffix>`。测试环境另外提供两个管理员邮箱，
凭据由测试环境注入且不提交仓库。Seed 至少包含：

- 三个启用的虚构证券，代码按 `AURR`、`NOVA`、`TIDE` 升序；
- 两个公开圈子 `long-horizon` 与 `risk-lab`；
- 一个禁用证券，仅用于验证帖子标签校验，不出现在证券列表。

测试把运行时 ID 记作 `aliceID`、`bobID`、`circleA`、`postA`、`commentA`、`reportA`
等别名，不依赖固定 ID。时间只断言合法 UTC RFC 3339 和先后关系，不断言机器时钟的
精确值。

## 2. V1 恰好 21 个操作

| # | operationId | 方法与路径 | 主要权限 | 幂等方式 |
| ---: | --- | --- | --- | --- |
| 1 | `registerUser` | `POST /auth/register` | 公开 | 邮箱唯一约束 |
| 2 | `loginUser` | `POST /auth/login` | 公开 | 无状态验证 |
| 3 | `getCurrentUser` | `GET /me` | JWT 对应当前 DB 用户 | 只读 |
| 4 | `listSecurities` | `GET /securities` | JWT 用户 | 只读 |
| 5 | `listCircles` | `GET /circles` | JWT 用户 | 只读 |
| 6 | `setCircleMembership` | `PUT /circles/{circleId}/membership` | JWT 用户 | PUT 天然幂等 |
| 7 | `listPosts` | `GET /posts` | JWT 用户 | 只读 |
| 8 | `createPost` | `POST /posts` | 圈子成员 | `Idempotency-Key` |
| 9 | `getPost` | `GET /posts/{postId}` | JWT 用户、目标可见 | 只读 |
| 10 | `updatePost` | `PATCH /posts/{postId}` | 作者且仍是成员 | 状态校验 |
| 11 | `deletePost` | `DELETE /posts/{postId}` | 作者 | 软删除 |
| 12 | `listComments` | `GET /posts/{postId}/comments` | JWT 用户、帖子可见 | 只读 |
| 13 | `createComment` | `POST /posts/{postId}/comments` | 圈子成员 | `Idempotency-Key` |
| 14 | `deleteComment` | `DELETE /comments/{commentId}` | 作者 | 软删除 |
| 15 | `createReport` | `POST /reports` | JWT 用户、目标可见 | 唯一关系安全重试 |
| 16 | `listNotifications` | `GET /notifications` | 仅当前接收者 | 只读 |
| 17 | `markAllNotificationsRead` | `PUT /notifications/read` | 仅当前接收者 | PUT 天然幂等 |
| 18 | `listAdminReports` | `GET /admin/reports` | admin | 只读 |
| 19 | `decideAdminReport` | `POST /admin/reports/{reportId}/decision` | admin | 状态机安全重试 + 行锁 |
| 20 | `restoreAdminContent` | `POST /admin/content/{targetType}/{targetId}/restore` | admin | 状态机安全重试 |
| 21 | `listAdminAuditLogs` | `GET /admin/audit-logs` | admin | 只读 |

### 每个操作允许的稳定错误码

下表是验收白名单；未列出的业务/协议错误码不得从该操作返回。任意操作发生未预期服务端
故障时仍统一返回 `internal_error`。特别地，登录凭据失败只能是 `invalid_credentials`，
受保护接口的 JWT 失败只能是 `unauthenticated`。

| operationId | 允许的非 5xx 错误码 |
| --- | --- |
| `registerUser` | `invalid_request`, `invalid_json`, `email_taken`, `validation_failed`, `payload_too_large`, `unsupported_media_type` |
| `loginUser` | `invalid_request`, `invalid_json`, `invalid_credentials`, `validation_failed`, `payload_too_large`, `unsupported_media_type` |
| `getCurrentUser` | `unauthenticated` |
| `listSecurities` | `invalid_request`, `invalid_cursor`, `unauthenticated` |
| `listCircles` | `invalid_request`, `invalid_cursor`, `unauthenticated` |
| `setCircleMembership` | `invalid_request`, `invalid_json`, `unauthenticated`, `not_found`, `payload_too_large`, `unsupported_media_type` |
| `listPosts` | `invalid_request`, `invalid_cursor`, `unauthenticated` |
| `createPost` | `invalid_request`, `invalid_json`, `unauthenticated`, `membership_required`, `idempotency_conflict`, `validation_failed`, `payload_too_large`, `unsupported_media_type` |
| `getPost` | `invalid_request`, `unauthenticated`, `not_found` |
| `updatePost` | `invalid_request`, `invalid_json`, `unauthenticated`, `forbidden`, `membership_required`, `not_found`, `content_not_editable`, `version_conflict`, `validation_failed`, `payload_too_large`, `unsupported_media_type` |
| `deletePost` | `invalid_request`, `unauthenticated`, `forbidden`, `not_found`, `content_not_editable` |
| `listComments` | `invalid_request`, `invalid_cursor`, `unauthenticated`, `not_found` |
| `createComment` | `invalid_request`, `invalid_json`, `unauthenticated`, `membership_required`, `not_found`, `idempotency_conflict`, `content_not_editable`, `parent_comment_invalid`, `validation_failed`, `payload_too_large`, `unsupported_media_type` |
| `deleteComment` | `invalid_request`, `unauthenticated`, `forbidden`, `not_found`, `content_not_editable` |
| `createReport` | `invalid_request`, `invalid_json`, `unauthenticated`, `not_found`, `self_report_forbidden`, `validation_failed`, `payload_too_large`, `unsupported_media_type` |
| `listNotifications` | `invalid_request`, `invalid_cursor`, `unauthenticated` |
| `markAllNotificationsRead` | `unauthenticated` |
| `listAdminReports` | `invalid_request`, `invalid_cursor`, `unauthenticated`, `forbidden` |
| `decideAdminReport` | `invalid_request`, `invalid_json`, `unauthenticated`, `forbidden`, `not_found`, `report_already_decided`, `validation_failed`, `payload_too_large`, `unsupported_media_type` |
| `restoreAdminContent` | `invalid_request`, `invalid_json`, `unauthenticated`, `forbidden`, `not_found`, `content_not_restorable`, `moderation_version_conflict`, `validation_failed`, `payload_too_large`, `unsupported_media_type` |
| `listAdminAuditLogs` | `invalid_request`, `invalid_cursor`, `unauthenticated`, `forbidden` |

任何额外 V1 业务操作都属于范围变更；`healthz`、`readyz` 和 Swagger 静态资源是运行端点，
不计入这 21 个业务操作。

## 3. 建议手写顺序

每阶段都遵循同一个节奏：先把本阶段场景写成失败测试，再写最小实现使其通过，最后运行
前面所有阶段的测试。注释解释权限来源、事务边界和状态不变量，不逐行翻译 Go 语法。

1. **错误与认证地基**：A01～A06、E01～E03。先学严格 JSON、密码哈希、JWT 和统一错误。
2. **证券与圈子**：C01～C04。先实现只读仓储和游标，再实现唯一成员关系。
3. **帖子闭环**：P01～P07。重点是作者来源、成员权限、`post_securities` 事务和软删除。
4. **评论与通知**：M01～M07。先限制一级回复，再保证回复和通知原子提交。
5. **用户举报**：R01～R04。分清“用户删除”和“管理员隐藏”两套状态。
6. **通知读取**：N01～N04。用 JWT 所有权约束查询、批量已读和四类通知形状。
7. **治理与审计**：G01～G09。最后处理行锁、并发决策、恢复和不可缺失的审计。
8. **工程化收口**：E04～E08。补真实 MySQL、故障回滚、安全配置和全量回归。

## 4. 场景：认证与统一边界

### A01 `[HTTP]` 注册成功且角色固定

- **Given** 一个从未使用过的合法邮箱、显示名和不少于 12 个字符的密码
- **When** 调用 `POST /auth/register`
- **Then** 返回 `201`、`{id,email,display_name,role=user,status=active}` 和 Bearer JWT
- **And** JWT 的 `sub` 等于响应用户 ID，Token 不携带授权角色，`expires_in` 为正数
- **And** 响应与日志都不包含密码或密码哈希

### A02 `[HTTP][MySQL]` 邮箱唯一且失败不泄露密码信息

- **Given** A01 的邮箱已经存在
- **When** 用大小写等价的同一邮箱再次注册
- **Then** 返回 `409 email_taken`，数据库仍只有一个该邮箱的用户
- **And** 邮箱规范化严格依次使用 `strings.TrimSpace`、`strings.ToLower`、
  `mail.ParseAddress`，并要求 Name 为空且 Address 与规范化字符串完全相等
- **When** 提交显示名邮箱、尖括号邮箱、地址列表或其他非单一裸地址
- **Then** 返回 `422 validation_failed`，不创建用户
- **When** 用少于 12 个字符的密码注册
- **Then** 返回 `422 validation_failed`，`details` 指向 `password`，不创建用户
- **When** 用超过 bcrypt 72 字节上限的密码注册
- **Then** 同样返回 `422 validation_failed`，不截断密码

### A03 `[HTTP]` 登录成功与凭据失败口径一致

- **Given** A01 已创建用户
- **When** 使用正确凭据调用 `POST /auth/login`
- **Then** 返回 `200` 和可用于受保护操作的 JWT
- **When** 使用错误密码或不存在的邮箱登录
- **Then** 两者均返回 `401 invalid_credentials` 和相同的公开消息，不暴露邮箱是否存在

### A04 `[HTTP]` JWT 校验完整

- **Given** 缺失、过期、错误签名、错误算法、错误 issuer 或错误 audience 的令牌
- **When** 调用任意受保护操作
- **Then** 返回 `401 unauthenticated`，业务 Handler 不执行，数据库不发生变化

### A05 `[HTTP]` 当前用户和角色来自数据库

- **Given** 有效 JWT 的 `sub` 指向 Alice，token 中携带伪造或过期的 role claim
- **When** 调用 `GET /me` 或任一管理员操作
- **Then** `/me` 返回数据库中的 `{id,email,display_name,role,status}`，授权也使用数据库值
- **And** 数据库用户不存在或已停用时返回 `401 unauthenticated`
- **When** 查询帖子、评论、通知、管理员举报列表或审计日志中的其他用户资料
- **Then** 作者、actor、reporter、decided_by 和 admin 都只含 `{id,display_name}`，
  不含 `email`、`role` 或 `status`

### A06 `[HTTP]` 带 JSON body 的写请求不能伪造身份

- **Given** Alice 的有效 JWT
- **When** 在任一带 JSON body 的写请求中额外提交 Bob 的 `author_id`、`user_id` 或 `reporter_id`
- **Then** 因未知字段返回 `400 invalid_request`
- **And** 不创建任何归属于 Bob 的记录

## 5. 场景：证券、圈子与游标

### C01 `[HTTP]` 证券目录稳定分页

- **Given** 三个启用证券与一个禁用证券夹具
- **When** 调用 `GET /securities?limit=2`
- **Then** 返回两个启用证券，按 `(code,id)` 升序，`has_more=true` 且 `next_cursor` 非空
- **When** 使用该游标请求下一页
- **Then** 返回剩余启用证券，无重复、无禁用证券，末页 `has_more=false` 且 `next_cursor=null`
- **And** 任意列表响应都满足 `has_more=true` 当且仅当 `next_cursor` 是非空字符串

### C02 `[HTTP]` 游标绑定筛选条件

- **Given** 从 `GET /securities?q=NO&limit=1` 得到游标
- **When** 篡改游标，或把它用于不同的 `q`/`exchange`/`limit` 组合
- **Then** 返回 `400 invalid_cursor`，而不是静默重置到第一页

### C03 `[HTTP][MySQL]` 加入公开圈子天然幂等

- **Given** Alice 尚未加入 `circleA`
- **When** 两次以 `{"joined":true}` 调用 `PUT /circles/{circleA}/membership`
- **Then** 两次都返回 `200`、`joined=true`，`joined_at` 完全相同
- **And** `GET /circles` 中该圈子的 `is_member=true`
- **And** 数据库只有一条 `(circle_id,user_id)` 成员关系
- **When** 再两次以 `{"joined":false}` 调用同一路径
- **Then** 两次都返回 `200`、`joined=false, joined_at=null`，成员关系已删除且不会影响 Alice 的历史内容

### C04 `[HTTP]` 业务 ID 边界与不存在的圈子

- **Given** 0、负数、小数、UUID 字符串或超过 int64 的圈子 ID
- **When** 尝试加入
- **Then** 返回 `400 invalid_request`
- **Given** 一个合法正 int64 但不存在的圈子 ID
- **When** 尝试加入
- **Then** 返回 `404 not_found`，不创建孤立成员关系

## 6. 场景：帖子与 `post_securities`

### P01 `[HTTP]` 非成员不能发帖

- **Given** Alice 未加入 `circleA`
- **When** 使用合法正文和证券 ID 调用 `POST /posts`
- **Then** 返回 `403 membership_required`，帖子与关联行均未创建

### P02 `[HTTP][MySQL]` 帖子和证券关联原子创建

- **Given** Alice 已加入 `circleA`，并选择 1～5 个不重复的启用证券
- **When** 带唯一 `Idempotency-Key` 调用 `POST /posts`
- **Then** 返回 `201`，作者 ID 必须是 Alice 的 JWT `sub`
- **And** 响应证券集合与请求 ID 集合相同
- **And** `version=1`、`moderation_version=1`，两者都是正 int64
- **And** 帖子与全部 `post_securities` 行同事务可见

### P03 `[HTTP][MySQL]` 标签校验失败没有半成品

- **Given** 空集合、六个证券、重复证券、未知证券或禁用证券中的任一种输入
- **When** 创建或替换帖子证券
- **Then** 返回 `422 validation_failed`
- **And** 新建时没有帖子和关联行；更新时保留原帖子和原关联集合

### P04 `[HTTP][MySQL]` 发帖幂等重放和冲突

- **Given** 一个合法发帖请求与 key `post-key-1`
- **When** 以相同用户、方法、路径和正文重放该请求（包括并发重放）
- **Then** 每次都返回第一次的 `201` 和同一个帖子 ID，只产生一篇帖子及一组关联
- **When** Alice 在 `createPost` 操作内用 `post-key-1` 发送不同正文
- **Then** 返回 `409 idempotency_conflict`，不产生新业务副作用
- **And** Bob 可独立使用同名 key，`createComment` 也可独立使用同名 key；幂等域不跨用户或操作

### P05 `[HTTP]` 帖子列表、筛选和详情只暴露可见内容

- **Given** 多个圈子、多个证券标签和多个创建时间相同的帖子
- **When** 按 `circle_id` 或 `security_id` 分页查询
- **Then** 只返回匹配项，按 `(created_at,id)` 倒序且跨页不重复
- **And** `GET /posts/{postA}` 返回相同的完整帖子
- **And** 已软删除或已治理隐藏的帖子在列表中不存在，普通用户详情返回 `404 not_found`

### P06 `[HTTP]` 修改权限与状态边界

- **Given** `postA` 由 Alice 创建且可见，当前 `version=v`
- **When** Alice 携带 `version=v` 修改标题、正文或证券集合
- **Then** 返回 `200`，未提交的字段保持不变，`version=v+1`，`updated_at` 不早于原值
- **When** 再用旧 `version=v` 提交修改
- **Then** 返回 `409 version_conflict`，帖子和证券关联都不变化
- **When** Bob 或管理员通过作者接口修改 `postA`
- **Then** 返回 `403 forbidden`；管理员只能走治理接口
- **When** Alice 修改治理隐藏中的 `postA`
- **Then** 返回 `409 content_not_editable`

### P07 `[HTTP][MySQL]` 作者软删除并收口举报

- **Given** Alice 的可见 `postA`，且不同举报者对它有多条 pending 举报
- **When** Bob 删除它
- **Then** 返回 `403 forbidden`
- **When** Alice 删除它
- **Then** 返回 `204` 且无响应体，之后详情为 `404 not_found`
- **And** 同一事务把该帖子全部 pending 举报更新为 `status=resolved`、
  `decided_action=author_deleted`、`decided_by=null`，`decided_at` 为本次系统时间
- **And** 此作者路径不新增管理员审计日志；实现可记录带 Request ID 的作者删除运行日志
- **When** Alice 再次删除同一资源
- **Then** 返回资源隐藏式 `404 not_found`

## 7. 场景：评论、一级回复与通知事务

### M01 `[HTTP]` 成员创建顶级评论

- **Given** Bob 已加入 `postA` 所在圈子且帖子可见
- **When** 省略 `parent_comment_id` 创建评论
- **Then** 返回 `201`，`parent_comment_id=null`，作者来自 Bob 的 JWT，且保留合法 `updated_at`
- **And** `moderation_version=1` 且为正 int64
- **And** 若帖子作者不是 Bob，则为帖子作者创建一条 `type=comment` 通知

### M02 `[HTTP][MySQL]` 回复他人与通知原子提交

- **Given** Bob 的可见顶级评论 `commentA`，Alice 也是圈子成员
- **When** Alice 以 `commentA` 为父评论进行回复
- **Then** 返回 `201`，回复和一条发给 Bob 的 `type=reply` 通知同事务提交
- **And** 通知 actor 是 Alice，并引用正确的帖子和新回复

### M03 `[HTTP][MySQL]` 回复自己不通知

- **Given** Alice 的可见顶级评论
- **When** Alice 回复自己的评论
- **Then** 回复创建成功，但 Alice 的通知数量不增加

### M04 `[HTTP]` 只允许一级回复且父评论必须同帖

- **Given** 另一个帖子的顶级评论，或已经是回复的评论
- **When** 把它作为 `parent_comment_id` 提交到当前帖子
- **Then** 返回 `422 parent_comment_invalid`
- **Given** 父评论属于当前帖子但已删除或隐藏
- **When** 尝试回复
- **Then** 返回 `409 content_not_editable`

### M05 `[HTTP][MySQL]` 评论幂等不重复通知

- **Given** 一个回复请求和固定 `Idempotency-Key`
- **When** 相同请求重放
- **Then** 返回第一次的 `201` 与同一评论 ID，且最多产生一条通知
- **And** 指纹由 `createComment + 当前用户 ID + 含实际 postId 的路径 + 规范正文` 组成；
  规范正文固定为无空白且字段顺序固定的 `body,parent_comment_id` JSON，再做 SHA-256
- **When** 同一用户用同一个 key 搭配不同正文或不同 `postId`
- **Then** 返回 `409 idempotency_conflict`
- **And** 另一用户可独立使用同名 key，因为用户 ID 是指纹和唯一域的一部分

### M06 `[HTTP]` 评论分页保留线程顺序

- **Given** 创建时间相同的多条顶级评论和回复
- **When** 分页调用 `GET /posts/{postA}/comments`
- **Then** 按 `(created_at,id)` 升序，无重复、无遗漏
- **And** 回复的 `parent_comment_id` 指向同帖可见顶级评论，但父评论不保证与回复位于同一页
- **And** 客户端不得把“当前页找不到父评论”解释成孤立回复

### M07 `[HTTP][MySQL]` 评论删除权限、父级可见性与举报收口

- **Given** Bob 的顶级评论带有 Alice 的回复，且该顶级评论有多条 pending 举报
- **When** Alice 尝试删除 Bob 的顶级评论
- **Then** 返回 `403 forbidden`
- **When** Bob 删除自己的顶级评论
- **Then** 返回 `204`；该评论及其回复均不再出现在普通评论列表
- **And** 记录仍是软删除，不把管理员隐藏状态伪装成用户删除
- **And** 同一事务把该评论全部 pending 举报更新为 `status=resolved`、
  `decided_action=author_deleted`、`decided_by=null`，`decided_at` 为本次系统时间
- **And** 不新增管理员审计日志；实现可另记作者删除运行日志

## 8. 场景：举报

### R01 `[HTTP][MySQL]` 创建待处理举报

- **Given** Bob 能看到 Alice 的帖子或评论
- **When** Bob 调用 `POST /reports`
- **Then** 返回 `201`、`status=pending` 和正确目标
- **And** reporter 只能是 Bob 的 JWT `sub`
- **And** `reason` 只能是 `spam`、`harassment`、`misleading`、`illegal`、`other`
- **When** 提交列表外 reason
- **Then** 返回 `422 validation_failed`，不创建举报
- **When** Alice 举报自己的帖子或评论
- **Then** 返回 `422 self_report_forbidden`，不创建举报

### R02 `[HTTP][MySQL]` 唯一关系让重复举报安全返回

- **Given** R01 已成功
- **And** 该目标随后被管理员隐藏
- **When** Bob 再次举报相同 `target_type + target_id`
- **Then** 返回 `200` 和原举报资源，不创建第二行，也不改变原状态
- **And** Alice 可以独立举报同一目标，因为唯一关系包含 reporter_id
- **And** 服务端顺序必须是“认证与字段校验 → 按唯一关系查重 → 新举报才检查可见性”

### R03 `[HTTP]` 没有既有举报时不可见目标不泄露

- **Given** 当前举报者从未举报过，且目标不存在、已由作者删除或已被治理隐藏
- **When** 普通用户举报该目标
- **Then** 统一返回 `404 not_found`

### R04 `[MySQL]` 举报目标类型互斥

- **Given** 任一举报记录
- **When** 检查真实 MySQL 约束
- **Then** 它恰好指向一篇帖子或一条评论，不能同时为空或同时存在

## 9. 场景：通知所有权与全部已读

### N01 `[HTTP]` 通知只能由接收者读取

- **Given** M02 为 Bob 创建了一条未读通知
- **When** Bob 调用 `GET /notifications?unread_only=true`
- **Then** 能看到该通知，按 `(created_at,id)` 倒序分页
- **When** Alice 调用同一接口
- **Then** Alice 的结果中没有 Bob 的通知；接口不接受 `user_id` 查询参数

### N02 `[HTTP][MySQL]` 全部已读只影响当前用户

- **Given** Bob 有两条未读通知，Alice 也有一条未读通知
- **When** Bob 调用 `PUT /notifications/read`
- **Then** 返回 `read_count=2`，Bob 的两条通知使用同一个 `read_at`
- **And** Alice 的通知仍未读

### N03 `[HTTP]` 全部已读天然幂等

- **Given** N02 已完成
- **When** Bob 再次调用 `PUT /notifications/read`
- **Then** 返回 `200` 和 `read_count=0`，不需要幂等键

### N04 `[HTTP]` 四类通知的形状与隐私

- **Given** 已发生他人评论、他人回复、管理员隐藏和管理员恢复
- **When** 对应接收者查询通知
- **Then** 四种 `type` 恰好为 `comment`、`reply`、`content_hidden`、`content_restored`
- **And** comment/reply 的 `actor` 是 `{id,display_name}` 且 `comment_id` 非空
- **And** content_hidden/content_restored 的 `actor=null`，不泄露管理员身份
- **And** 帖子治理通知 `comment_id=null`；评论治理通知保留对应评论 ID

## 10. 场景：管理员治理、恢复与审计

### G01 `[HTTP]` 管理员边界

- **Given** 普通用户 JWT
- **When** 调用三个 `/admin/...` 资源中的任一操作
- **Then** 返回 `403 forbidden`
- **Given** 无效 JWT
- **Then** 在角色判断前返回 `401 unauthenticated`

### G02 `[HTTP]` 举报工作队列

- **Given** 多条不同状态与目标类型的举报
- **When** 管理员按 `status`、`target_type` 和游标查询
- **Then** 只返回匹配项，按 `(created_at,id)` 倒序
- **And** 目标快照足以做治理判断，但普通内容接口仍不能借此看到隐藏内容
- **And** 每个 target 快照包含当前正 int64 `moderation_version`
- **And** `AdminReport` 状态组合只能是：pending 全空决策；ignored/ignore 带管理员；
  resolved/hide 带管理员；或 resolved/author_deleted 且 `decided_by=null`、`decided_at` 非空

### G03 `[HTTP][MySQL]` 忽略举报必须写审计

- **Given** pending 举报且管理员提交 `decision=author_deleted`
- **When** 调用举报决策接口
- **Then** 返回 `400 invalid_request`，举报、目标和审计均不变化
- **And** `author_deleted` 只能由作者 DELETE 帖子或评论的系统事务写入
- **Given** pending 举报 `reportA`
- **When** 管理员以 `decision=ignore` 处理
- **Then** 返回 `200`、`status=ignored`，目标仍 visible
- **And** 同事务新增且仅新增一条 `report_ignored` 审计日志

### G04 `[HTTP][MySQL]` 隐藏目标必须原子完成

- **Given** 不同举报者对同一可见目标有多条 pending 举报，目标当前 `moderation_version=m`
- **When** 管理员以 `decision=hide` 处理
- **Then** 返回 `200`，目标变为 hidden
- **And** 同目标全部 pending 举报在同一事务中变为 `status=resolved`、
  `decided_action=hide`，共享本次 `decided_by` 与 `decided_at`
- **And** 目标变为 hidden、`moderation_version=m+1`，AdminReport target 返回该新值
- **And** 普通列表不再返回目标、普通详情为 `404 not_found`
- **And** 同事务新增且仅新增一条 `content_hidden` 审计日志
- **And** 同事务为目标作者创建一条 `type=content_hidden`、`actor=null` 的通知

### G05 `[HTTP][MySQL]` 两名管理员并发决策只有一个赢家

- **Given** 同一 pending 举报与两名管理员提交不同决策
- **When** 两个请求并发到达
- **Then** 恰好一个返回 `200`，另一个返回 `409 report_already_decided`
- **And** 举报只记录赢家、内容状态与赢家一致、审计日志只有一条

### G06 `[HTTP][MySQL]` 相同治理决策安全重试

- **Given** G03 或 G04 已完成
- **When** 完全相同请求重放
- **Then** 返回 `200` 和当前 AdminReport，而不是 `report_already_decided`
- **And** 不新增第二条审计日志

### G07 `[HTTP][MySQL]` 恢复只针对治理隐藏

- **Given** G04 隐藏的内容且作者没有删除它，当前 `moderation_version=m`
- **When** 管理员以严格 JSON `{"expected_moderation_version":m}` 调用恢复接口
- **Then** 返回 `200`、`visibility=visible`、`moderation_version=m+1`，普通查询重新可见
- **And** 同事务新增且仅新增一条 `content_restored` 审计日志
- **And** 同事务为目标作者创建一条 `type=content_restored`、`actor=null` 的通知

### G08 `[HTTP]` 恢复状态机安全重试

- **Given** G07 已完成
- **When** 用同一个 `expected_moderation_version=m` 直接重试
- **Then** 服务端以既有恢复记录确认 visible 且当前版本为 `m+1`，返回原 `200`，不重复审计
- **When** 内容在恢复后再次被隐藏，当前版本变为 `m+2`，再重试 expected=m
- **Then** 返回 `409 moderation_version_conflict`，不得把 ABA 状态误判为成功重试
- **When** 目标仍 hidden 但当前版本不等于 expected
- **Then** 返回 `409 moderation_version_conflict`
- **When** 恢复从未治理隐藏或作者已删除的内容
- **Then** 返回 `409 content_not_restorable`
- **When** 合法正 int64 目标 ID 从未存在
- **Then** 返回 `404 not_found`
- **When** `targetType` 不是 `post` 或 `comment`
- **Then** 固定返回 `400 invalid_request`，不执行目标查询

### G09 `[HTTP]` 审计只读、可筛选、稳定分页

- **Given** 忽略、隐藏和恢复都已发生
- **When** 管理员按 `action`、`target_type` 或 `admin_id` 查询审计日志
- **Then** 返回匹配项，按 `(created_at,id)` 倒序且跨页无重复
- **And** 每条日志包含管理员、动作、目标、可选举报 ID、说明和时间
- **And** V1 不存在修改或删除审计日志的接口

## 11. 场景：协议、安全与事务收口

### E01 `[HTTP]` 严格 JSON

- **Given** 任一带请求体的操作
- **When** Content-Type 不是 `application/json`
- **Then** 返回 `415 unsupported_media_type`
- **When** JSON 语法错误或包含尾随第二个 JSON 值
- **Then** 返回 `400 invalid_json`
- **When** JSON 包含未知字段
- **Then** 返回 `400 invalid_request`
- **When** 已知字段违反长度、枚举或数量规则
- **Then** 返回 `422 validation_failed` 和对应字段详情

### E02 `[HTTP]` 请求体上限与错误信封

- **Given** 原始字节恰好 1,048,576 bytes 或少于该值的合法 JSON 请求体
- **When** 提交到任一带正文的操作
- **Then** 不因大小本身返回 413
- **Given** 原始字节超过 1 MiB（1,048,576 bytes）的请求体
- **When** 提交到任一带正文的操作
- **Then** 返回 `413 payload_too_large`
- **And** 上述所有失败都使用统一错误信封，Request ID Header 与正文一致

### E03 `[HTTP]` 资源隐藏和权限错误不混用

- **Given** 用户能看到资源但不是作者
- **When** 调用作者专属修改或删除
- **Then** 返回 `403 forbidden`
- **Given** 资源对该用户不可见、已删除、已隐藏或不存在
- **When** 通过普通读取或举报入口访问
- **Then** 返回 `404 not_found`

### E04 `[MySQL]` 关键事务故障回滚

- **Given** 测试故障点分别位于帖子与关联之间、回复与通知之间、治理状态与审计之间
- **When** 后半段写入被强制失败
- **Then** 整个事务回滚，不留下帖子半成品、无通知回复或无审计治理状态

### E05 `[MySQL]` 并发唯一约束

- **Given** 同一用户和圈子的并发加入、并发重复举报，以及帖子/评论同 key 的并发创建
- **When** 请求同时提交
- **Then** 成员关系、业务实体、幂等记录和副作用都各自最多一份
- **And** 不把数据库 duplicate-key 原文泄露给客户端

### E06 `[Source]` 密码、JWT 与日志

- **Given** reference 的配置和日志代码
- **When** 审查密码存储、JWT 验证及结构化日志字段
- **Then** 密码使用专用哈希算法，JWT 固定算法/issuer/audience/过期校验
- **And** 密码、哈希、Token、Authorization Header 和完整正文不进入日志

### E07 `[HTTP]` 全量业务演示

- **Given** 全新数据库已迁移并加载虚构 Seed
- **When** 依次执行“注册 → 登录 → 浏览证券 → 浏览并加入圈子 → 发帖 → 评论 →
  回复并收到通知 → 举报 → 管理员隐藏 → 查询审计 → 管理员恢复”
- **Then** 21 个操作覆盖表中的每个 operationId 至少被一个场景调用
- **And** 全流程不需要真实市场、公司或个人数据

### E08 `[Source]` 双轨契约一致

- **Given** reference 与 starter
- **When** 分别构建并运行相同的契约测试
- **Then** 两者不互相导入业务代码，只共同读取本目录的 OpenAPI 与验收场景
- **And** 默认测试可在没有 Docker 和外部数据库时通过；MySQL 场景由显式标签启用

## 12. 完成判定

只有同时满足以下条件才可宣称 V1 契约实现完成：

1. OpenAPI 解析成功且恰好有 21 个唯一 `operationId`；
2. A、C、P、M、R、N、G、E 全部场景均有自动化检查或明确的 `[Source]` 审查证据；
3. reference 和 starter 的默认构建、测试均绿色，且 reference 的真实 MySQL 测试绿色；
4. 故障回滚、同 key 重放、双管理员并发和通知所有权均被测试，而不是只写在注释里；
5. 仓库中没有公司源码、真实数据、密钥、品牌资产或第二套冲突契约。

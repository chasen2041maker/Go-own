# 阶段 02：认证与可信身份

## 阶段目标

实现注册、登录和当前用户查询，建立后续所有对象权限的唯一身份来源。重点不是 JWT 语法，而是密码、Token 和请求数据之间的信任边界。

## 1. 前置知识

- 单向密码哈希、随机盐、成本参数与恒定语义的登录失败；
- JWT Header、Claims、签名、过期时间、发行者和受众；
- HTTP `Authorization: Bearer` 语义；
- `context.Context` 传递请求范围身份；
- 数据库唯一约束和并发注册；
- 用例接口、仓储 fake、可注入 Clock 和 Token 签发器。

不要自己设计加密算法，也不要把 Base64 编码误认为加密。

## 2. 业务故事

访客按 OpenAPI 提交 email、display_name 和密码完成注册并登录。服务只保存密码哈希。用户随后调用 `GET /api/v1/me`；服务验证 Token 后只取 `sub`，再加载当前数据库用户并只向本人返回 `{id,email,display_name,role,status}`。

## 3. 调用链

注册：

```text
POST /api/v1/auth/register
  → 严格解析 email、display_name 和密码
  → Register Use Case 规范化 email 并校验输入
  → 密码哈希器（事务外完成耗时计算）
  → User Repository 插入 user 角色
  → 唯一键冲突映射为 409
```

登录：

```text
POST /api/v1/auth/login
  → Login Use Case 按规范化 email 查用户
  → 密码哈希器验证
  → Token 签发器写入 subject / issuer / audience / exp
  → 返回访问 Token，不写入日志
```

当前用户：

```text
GET /api/v1/me
  → Bearer 中间件严格验证算法和 Claims
  → 只将已验证 subject 放入 Context
  → Me Use Case 按 subject 读取 active 用户和当前数据库角色
  → 返回仅本人可见的 PrivateCurrentUser 字段
```

## 4. 数据变化

新增 `users`：

- `id` 为 MySQL BIGINT 自增主键，在 Go/OpenAPI 中为正数 int64；
- `email` 保存规范化 email 并唯一；
- `display_name` 保存展示名，API 使用同名字段，最大 80 个字符；
- `password_hash` 只存专用哈希结果；
- 注册密码至少 12 个 Unicode 字符且 UTF-8 不超过 72 字节；登录请求允许 1～128 个字符，长度错误在执行一次 dummy bcrypt 后返回字段校验错误；
- `role` 只允许 `user` 或 `admin`，公开注册固定创建 `user`；
- `status` 只允许 `active` 或 `disabled`；
- `created_at`、`updated_at` 使用 UTC。

管理员由受控 Seed 或运维流程创建，注册请求不能提交 `role=admin`。`uq_users_email` 是并发注册的最终保护；应用层预检查不能代替它。JWT 中即使存在 role claim，也不参与授权；`/me` 和管理用例都以当前数据库用户的 role/status 为准，停用或不存在返回 `401 unauthenticated`。

## 5. 先写的失败测试及为何失败

1. `TestRegisterStoresHashAndNeverPlaintext`：注册后仓储收到的值不能等于原密码。最初失败，因为尚未注入密码哈希器；
2. `TestRegisterAlwaysCreatesUserRole`：请求带未知 `role` 字段返回 `400 invalid_request`，合法注册响应的 role 固定为 `user`。最初失败，因为 DTO 或严格解析不存在；
3. `TestRegisterDuplicateEmailReturnsConflict`：仓储返回 `uq_users_email` 冲突时得到稳定 `409 email_taken`。最初失败，因为数据库错误尚未翻译；
4. `TestLoginUnknownUserAndWrongPasswordSharePublicError`：两种情况都返回 `401 invalid_credentials`。最初失败，因为实现可能泄露账户是否存在；
5. `TestTokenRejectsUnexpectedAlgorithm`：伪造不同算法的 Token 必须返回 `401 unauthenticated`。最初失败，因为解析器可能接受 Token 自带算法；
6. `TestTokenValidatesIssuerAudienceAndExpiry`：分别改变三个 Claim 均返回 `401 unauthenticated`。最初失败，因为只验签、不验用途；
7. `TestMeUsesSubjectThenLoadsActiveDatabaseUser`：请求体或查询串伪造他人 ID 不改变结果；用户不存在或 disabled 均为 `401 unauthenticated`。最初失败，因为身份上下文与数据库状态尚未衔接；
8. `TestRoleClaimDoesNotGrantAdmin`：Token 自带 admin claim、数据库仍为 user 时必须按 user 授权。最初失败，因为中间件直接相信 role claim；
9. 集成测试 `TestConcurrentRegistrationCreatesOneUser`：并发同 email 注册恰好一个成功、一个 `409 email_taken`。最初失败，因为 fake 不能证明唯一键竞争。

确认错误来自缺少业务能力，而不是测试使用了真实时间导致偶发过期。Token 测试应注入固定 Clock。

## 6. GREEN 最大边界

只实现 email/display_name/密码注册、登录、短期访问 Token、认证中间件、数据库角色/状态加载和 `GET /api/v1/me`。给哈希器、Clock、Token 签发/验证与用户仓储建立窄接口。

不要加入 Refresh Token、Session 表、第三方登录、短信/邮件、找回密码、头像、用户资料编辑或通用权限 DSL。密码策略只实现规格明确的最小/最大长度与输入上限；不要自行构造复杂正则。

Token 只证明 `sub`；全局角色与 active 状态从数据库加载，对象归属仍在用例中查当前数据。客户端提交的作者、举报者或通知所有者 ID 永远不能覆盖 Context 身份。

## 7. 验证命令

下面的命令会实际执行 `-list` 并检查非空；找不到匹配测试时直接抛错，不会继续把 `-run` 的零匹配当成通过。

```powershell
$unitPattern = 'Test(Register|Login|Token|Me)'
$unitList = go test ./projects/04-investment-community/starter/... -list $unitPattern
if ($LASTEXITCODE -ne 0) { exit $LASTEXITCODE }
if (-not ($unitList | Select-String '^Test')) { throw '阶段 02 没有匹配的单元测试' }
go test ./projects/04-investment-community/starter/... -run $unitPattern -count=1
go test ./projects/04-investment-community/starter/... -count=1
go test ./... -count=1
go vet ./projects/04-investment-community/starter/...
```

真实 MySQL 环境补充：

```powershell
$integrationPattern = 'TestConcurrentRegistration|TestUser'
$integrationList = go test -tags=integration ./projects/04-investment-community/starter/... -list $integrationPattern
if ($LASTEXITCODE -ne 0) { exit $LASTEXITCODE }
if (-not ($integrationList | Select-String '^Test')) { throw '阶段 02 没有匹配的集成测试' }
go test -tags=integration ./projects/04-investment-community/starter/... -run $integrationPattern -count=1
```

手工请求后检查结构化日志：不得出现密码、哈希、完整 Token 或 Authorization Header。

## 8. 变式练习

- 把固定 Clock 前进到过期边界前后各一秒，写清边界是包含还是不包含；
- 让密码哈希器返回内部错误，验证 API 不泄露库名称或成本参数；
- 尝试 email 大小写和两端空白，先写出规范化政策再补测试；
- 构造合法签名但错误 audience 的 Token，观察“验签成功”为什么仍不能认证；
- 比较把完整用户对象放 Context 与只放不可伪造身份值的取舍。

## 9. 理解 / 面试问题

1. 密码为什么不能可逆加密后保存？
2. 用户不存在与密码错误为何应暴露相同语义？
3. 为什么必须限制 JWT 算法，而不能相信 Header？
4. issuer、audience 和 expiry 分别阻止什么误用？
5. 为什么客户端提交的 `user_id` 不能决定作者？
6. 应用层已查 email，数据库为什么仍要唯一索引？
7. 密码哈希为什么不应放在长事务中？
8. 为什么 JWT role claim 不能替代当前数据库角色与状态？

## 10. 中文注释落点

值得注释：为什么未知用户与错密码统一失败；为什么显式限定 JWT 算法/发行者/受众；为什么授权重新加载 active 用户而不相信 role claim；为什么身份从 Context 取而不从 DTO 取。

不值得注释：字符串裁剪、调用哈希函数、设置 Header 或读取 Claim 的逐行语法。

## 完成定义

注册从不保存明文密码，登录错误不枚举账户，JWT 用固定 Clock 可重复测试，`/me` 只信任已验证身份，并发注册由真实 MySQL 唯一约束收敛。

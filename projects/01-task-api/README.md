# Task API

当前主项目：使用 Go 标准库手写个人任务管理 HTTP API。

## 当前可运行内容

- `GET /health` 返回 `200` 和 JSON 健康状态。
- 其他方法访问 `/health` 返回 `405`，并设置 `Allow: GET`。
- 已定义 Task 数据模型，尚未实现任务 CRUD。

启动：

```powershell
go run ./projects/01-task-api
```

服务只监听 `127.0.0.1:8080`。

测试：

```powershell
go test ./projects/01-task-api
```

## 手写里程碑

每个里程碑都要同步编写并运行本阶段测试，不能先写完前三个里程碑再统一补测试。

## CRUD HTTP 契约

以下规则适用于以后实现的 `/tasks` CRUD，不改变当前 `/health` 的最小响应：

- POST 和 PATCH 必须提供可由 `mime.ParseMediaType` 解析且媒体类型为 `application/json` 的 `Content-Type`；可以带合法参数，例如 `application/json; charset=utf-8`。缺失、无法解析或其他媒体类型都返回 `415`。
- 请求 Body 必须是一个非 `null` 的 JSON 对象，并且对象结束后的空白后必须立刻到 EOF。空 Body、`null`、数组、语法错误、尾随非空白内容或连续多个 JSON 值都返回 `400`。
- CRUD 的 JSON 错误响应固定为 `{"error":{"code":"<稳定代码>","message":"<可读说明>"}}`，并设置 `Content-Type: application/json`。状态码与代码对应为：`400 invalid_request`、`404 task_not_found`、`405 method_not_allowed`、`415 unsupported_media_type`、`500 internal_error`；`405` 还必须设置正确的 `Allow`。
- `POST /tasks` 成功返回 `201`、`Location: /tasks/{id}` 和完整 Task JSON。
- `GET /tasks` 成功返回 `200` 和按 ID 升序的 JSON 数组；没有任务时必须是 `[]`，不能是 `null`。
- `GET /tasks/{id}` 成功返回 `200` 和完整 Task JSON。
- `PATCH /tasks/{id}` 成功返回 `200` 和修改后的完整 Task JSON。
- `DELETE /tasks/{id}` 成功返回 `204` 和空 Body，不再编码 JSON。

路径 ID 的唯一合法形式是单个 ASCII 十进制正整数段，例如 `/tasks/1`：只能包含 `0`～`9`，不能有正负号、前导零或额外路径段，并且必须能放入当前构建的 Go `int`。`0`、负数、`01`、非十进制字符和整数溢出都返回 `400 invalid_request`；`/tasks/1/extra` 等额外路径段返回 `404 task_not_found`。

### 1. 内存存储

实现任务的新增、按 ID 查询和列表查询，数据只保存在内存中。

完成条件：能用测试证明 ID 唯一；列表始终按 ID 从小到大返回；查询不存在的任务会返回可判断的错误；存储不依赖 HTTP。

### 2. 创建和查询 API

实现 `POST /tasks`、`GET /tasks` 和 `GET /tasks/{id}`。

完成条件：遵守上面的媒体类型、单对象 Body、响应和路径契约；POST 请求只允许出现 `title`、`description`、`status`，任何未知字段、服务器管理字段 `id`、`created_at`、`updated_at`，或任何显式 `null` 都返回 `400`；标题去掉首尾空白后不能为空，并存储去掉首尾空白后的值；创建时未提供状态则默认使用 `todo`，提供非法状态则返回 `400`；`id` 和时间由服务器生成。

### 3. 修改和删除 API

实现 `PATCH /tasks/{id}` 和 `DELETE /tasks/{id}`，状态只允许 `todo`、`doing`、`done`。

完成条件：遵守上面的媒体类型、单对象 Body、响应和路径契约；PATCH 只允许出现 `title`、`description`、`status`；未知字段或服务器管理字段 `id`、`created_at`、`updated_at` 返回 `400`；任何字段显式传 `null` 返回 `400`；空对象 `{}` 返回 `400`；字段缺席表示保持原值，空字符串 `description` 表示清空描述；明确提供的标题会先去掉首尾空白，结果为空或状态非法时返回 `400`。如果所有已提供字段经过规范化后都与原值相同，仍返回 `200` 和当前 Task，但不改变 `updated_at`；只有实际发生修改才更新 `updated_at`。

### 4. 补全表驱动测试和代码整理

补充前三个里程碑遗漏的边界场景，减少测试重复，并只在职责确实分开时拆文件。本阶段不是第一次写测试。

完成条件：每个已实现功能的正常与错误路径都有测试；`go test ./...` 和 `go vet ./...` 通过；没有为了“以后可能用到”而创建接口或分层。

### 5. 并发安全

用并发测试暴露共享 Map 的数据竞争，再使用合适的同步方式修复。

完成条件：能解释锁保护的数据和范围；满足下面的 Race Detector 环境前置条件后，`go test -race ./...` 不报告数据竞争。

### 6. JSON 文件持久化

按三个小任务依次完成：

持久化文件固定为一个 JSON 对象：`{"next_id":N,"tasks":[...]}`。顶层必须恰好包含 `next_id` 和 `tasks`，两者都不能缺失或为 `null`；`next_id` 是正整数，`tasks` 是非 `null` 数组并按 ID 升序写入，空集合写成 `[]`。每个任务对象必须恰好包含 `id`、`title`、`description`、`status`、`created_at`、`updated_at`，缺失、未知或 `null` 字段都使文件无效。读取时使用与 HTTP Body 相同的严格边界：只允许一个非 `null` 对象，拒绝未知字段、尾随内容和多个 JSON 值。

1. **6.1 启动加载：** 从 JSON 文件恢复任务；主文件和备份都不存在时视为空数据。验证 envelope、全部任务记录和 `next_id` 后才接受文件。
2. **6.2 可恢复写入：** 先写同目录临时文件，完成编码、同步和关闭后再替换目标文件。同目录临时文件可以降低部分写入风险，但不承诺 Windows 崩溃场景下的原子替换。
3. **6.3 串行状态提交：** 复用里程碑 5 保护任务 Map 和下一个 ID 的互斥锁，在同一次独占锁内严格执行“克隆当前状态 → 在克隆上应用变更 → 持久化克隆 → 成功后提交克隆”；不能先修改活动内存再尝试写盘。

加载文件必须拒绝以下数据并终止启动：重复或非正数 ID、空白标题、`todo`/`doing`/`done` 以外的状态、零值时间、`updated_at` 早于 `created_at`，以及不按 ID 升序排列的任务。空任务数组要求 `next_id == 1`；非空数组要求 `next_id == maxID + 1`。最大 ID 已等于 Go `int` 最大值时必须拒绝，不能让下一 ID 溢出。

备份路径固定为 `<数据文件>.bak`，绝不能覆盖已有备份。启动恢复顺序固定为：主文件缺失且备份存在时，先把备份恢复成主文件再严格加载；主文件与备份同时存在时拒绝启动，交给用户人工决定；主文件严格校验成功且没有备份时，可以清理同目录中属于该数据文件的陈旧临时文件，但绝不能把临时文件当作恢复来源。

写入生命周期固定为：创建同目录临时文件 → 编码升序 envelope → 同步并关闭 → 若有主文件则确认备份不存在并把主文件移到 `.bak` → 把临时文件移到主路径 → 校验新主文件 → 删除 `.bak` 后才报告持久化成功。任一步失败都不提交活动内存；替换新主文件失败时先把 `.bak` 恢复为主文件。新主文件建立后若校验或备份清理失败，则尝试删除新主文件并恢复 `.bak`；恢复也失败时保留 `.bak` 和仍存在的临时/主文件，在错误中报告所有路径，等待人工恢复。无论成功失败，都只能清理已经确认不再需要且不会破坏恢复能力的文件。

完成条件：各小任务都有独立测试；重启后数据仍存在；所有无效加载数据都被拒绝；写入失败不提交本次内存修改；替换失败后原文件已恢复，或存在错误中明确指出的可恢复备份。

### 7. 工程化

按三个小任务依次完成：

1. **7.1 配置与超时：** `PORT` 只配置端口，服务始终绑定 `127.0.0.1`；默认端口为 `8080`，只接受 `1`～`65535` 的十进制整数。`TASK_DATA_FILE` 默认 `tasks.json`，去掉首尾空白后不能为空。为 `http.Server` 设置 `ReadHeaderTimeout: 5s`、`ReadTimeout: 10s`、`WriteTimeout: 10s`、`IdleTimeout: 60s`，所有超时都必须为正数。
2. **7.2 日志：** 使用 `log/slog` 记录启动、关闭和关键错误，不记录任务正文等不必要数据。
3. **7.3 优雅退出与文档：** Windows 监听 `os.Interrupt`；Unix 监听 `os.Interrupt` 和 `syscall.SIGTERM`。收到信号后给 `Server.Shutdown` 最多 `10s` 等待已有请求完成；超时或 Shutdown 失败时记录错误并调用 `Server.Close` 强制关闭，最后补全运行与 API 调用示例。

完成条件：无配置时使用上述默认值；非法端口或空数据路径会明确失败；任何配置都不能把监听主机改出 `127.0.0.1`；正常启动后完成优雅退出时进程状态码为 `0`，配置、监听、服务或关闭错误以及被迫 `Server.Close` 时为 `1`；README 中的命令可以照着运行。

## Race Detector 环境

`go test -race` 需要启用 CGO。Windows/amd64 还需要兼容的 C 编译器和可用的 `mingw-w64` runtime；也可以在 WSL 中运行。开始并发安全里程碑前先执行 `go env CGO_ENABLED`：如果结果是 `0`，先配置上述环境，不要把 `-race requires cgo` 误认为业务代码错误。

一次只做一个里程碑。先提交自己的代码和运行结果，再进入下一项。

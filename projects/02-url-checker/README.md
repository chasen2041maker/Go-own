# 并发网址检测器

目标：编写命令行程序，检查多个网址的 HTTP 状态、请求耗时和错误，并用它真正理解 Go 并发。

## 命令行契约

调用格式固定为：

```text
url-checker [-mode serial|concurrent] [-workers N] [-timeout DURATION] URL...
```

- `-mode` 默认 `serial`，只允许 `serial` 或 `concurrent`。
- `-workers` 默认 `4`，单位是同时工作的 goroutine 数量，只允许 `1`～`256`；仅在 `concurrent` 模式使用。
- `-timeout` 默认 `10s`，必须是正的 Go duration，可使用 `ms`、`s`、`m` 等 `time.ParseDuration` 单位。它从批次开始计时，限制排队、校验和 HTTP 请求处理；deadline 到达后允许继续合成并输出每个输入对应的取消结果，因此进程结束时间可以略晚于该值。
- 至少提供一个 URL；没有 URL、未知 flag、非法 mode、非法 worker 数或非法 timeout 都向标准错误输出原因并以状态码 `2` 退出。
- URL 必须是带非空 host 的绝对 `http` 或 `https` URL；重复 URL 合法，并按各自在输入参数中的位置分别检查。

每个输入在标准输出占一行 JSON，固定字段如下：

```json
{"input_index":0,"url":"https://example.com","status_code":200,"duration_ms":12,"error_code":"","error":""}
```

`input_index` 从 `0` 开始，用于区分重复 URL；`status_code` 没有收到 HTTP 响应时为 `0`；`duration_ms` 使用 `time.Duration.Milliseconds()` 向下取整为非负整数毫秒：真正开始校验/请求的任务从开始处理计到 `http.Client.Do` 返回，仍在队列中就被取消的任务固定为 `0`，URL 校验立即失败也为 `0`；`error` 是面向人的错误文本。`error_code` 只能是空字符串、`invalid_url`、`request_failed`、`deadline_exceeded` 或 `canceled`。收到任何 HTTP 状态码都表示请求完成，因此 `error_code` 为空。

错误分类顺序必须固定：任务开始前批次 Context 已结束时，根据 `ctx.Err()` 记录 `deadline_exceeded` 或 `canceled`；否则先验证 URL，不合法记为 `invalid_url`；`http.Client.Do` 返回 `response != nil && err == nil` 时始终以该 HTTP 响应为成功结果，即使 Context 在同时变为 done；只有请求返回错误时，才依次判断批次 `ctx.Err()`、`errors.Is(err, context.DeadlineExceeded)`、`errors.Is(err, context.Canceled)`，分别记录 `deadline_exceeded` 或 `canceled`，都不匹配才记为 `request_failed`。并发输出顺序不保证，但每个 `input_index` 必须恰好出现一次。

程序还要监听 `os.Interrupt`。收到中断时取消批次 Context：尚未开始和正在执行的任务按上述规则合成 `canceled` 结果，完整输出每个 `input_index` 后以状态码 `1` 退出。

## 递进需求

1. 串行读取命令行中的网址并逐个请求。
2. 为每个结果输出网址、HTTP 状态、耗时或错误，单个失败不能让整个程序崩溃。
3. 使用 goroutine 和 channel 实现并发版本。
4. 改成固定数量的 Worker Pool，并允许用户设置 `1`～`256` 的 worker 数量；越界或无法解析的值应报错退出。
5. 使用 Context 为整批任务设置一个总超时和取消；每个输入网址最终都必须对应一条结果，尚未开始或尚未完成的检查按上面的稳定错误码记录。
6. 为结果收集、超时和错误场景编写测试，再用 `go test -race` 检查数据竞争。

## 完成条件

- 同一组网址可分别用串行和并发模式执行，结果条数与输入数一致；测试按 `input_index` 核对结果，不用 URL 作为唯一键。
- 自动化测试使用本地 `httptest.Server` 模拟成功、错误状态和延迟响应，不依赖公开网站。
- 无效网址、连接失败和超时都有明确结果，不会卡死。
- 实际并发数不会超过用户设置的 worker 数量。
- channel 的发送方、关闭方和退出条件清晰，不依赖 `time.Sleep` 等待完成。
- 参数非法按命令行契约返回 `2`；参数合法但任一结果的 `error_code` 非空时返回 `1`；全部 `error_code` 为空时返回 `0`。收到 HTTP 4xx 或 5xx 仍算完成检查，不导致非零退出。任何 JSON 编码或标准输出写入失败都向标准错误报告并返回 `1`；写失败后不承诺还能输出剩余结果。
- `go test ./...`、`go vet ./...` 和 `go test -race ./...` 通过。

`go test -race` 需要启用 CGO。Windows/amd64 还需要兼容的 C 编译器和可用的 `mingw-w64` runtime；也可以在 WSL 中运行。如果 `go env CGO_ENABLED` 输出 `0`，应先配置环境，不要把 `-race requires cgo` 当成项目代码失败。

Task API 的并发安全里程碑完成后再开始本项目；当前不要创建实现文件。

# 日志分析器

目标：从文本文件读取访问日志，解析为结构化记录并生成统计报告，练习文件处理、字符串解析、错误分类和性能分析。

## 命令行与输出契约

调用格式固定为：

```text
log-analyzer [-format text|json] FILE
```

`-format` 默认 `text`，只允许 `text` 或 `json`；`FILE` 必须恰好有一个，值为 `-` 时从标准输入读取。缺少/多余 FILE、未知 flag 或非法 format 向标准错误输出用法并返回 `2`。文件打开失败向标准错误输出原因、不输出报告并返回 `1`。

`text` 格式严格按下面的顺序逐行输出，以 `\n` 结尾：

```text
total_requests=2
status_counts=200:1,404:1
top_path=/tasks
average_duration_ms=12.5
invalid_lines=1
invalid_line=3:invalid_status
```

状态码按数值升序排列并用逗号连接；没有状态码时输出 `status_counts=`，没有热门路径时输出 `top_path=`。平均值先按下文规则舍入，再用 `strconv.FormatFloat(value, 'f', -1, 64)` 输出。`invalid_lines=N` 后按行号升序为每个坏行追加一条 `invalid_line=<line>:<reason>`；没有坏行时不追加 `invalid_line` 行。

`json` 格式输出一个对象并以 `\n` 结尾，字段固定为 `total_requests`、`status_counts`、`top_path`、`average_duration_ms` 和 `invalid_lines`。`invalid_lines` 每项为 `{"line":3,"reason":"invalid_status"}`。无论哪种格式，输出编码或写入失败都向标准错误报告并返回 `1`。

退出码固定为：没有坏行且无 I/O/输出错误时 `0`；存在任何坏行、读取错误、统计溢出或输出错误时 `1`，但能生成报告时仍先输出报告；命令行用法错误时 `2`。

## 输入约定

第一版使用 `strings.Fields` 按 Unicode 空白切分；每个逻辑行必须恰好包含五个字段：RFC3339 时间、HTTP 方法、路径、状态码和耗时毫秒数，例如 `2026-08-17T10:00:00Z GET /tasks 200 12`。

- 时间必须能由 `time.Parse(time.RFC3339, value)` 解析。
- 方法只允许大写的 `GET`、`HEAD`、`POST`、`PUT`、`PATCH`、`DELETE`、`OPTIONS`。
- 路径必须以 `/` 开头且不能为空。
- 状态码必须是 `100`～`599` 的十进制整数。
- 耗时必须是 `0`～`86400000` 的十进制整数，单位为毫秒。

任一规则不满足时整行无效，不把部分字段计入统计。先在测试中固定合法与非法样例，再实现解析器。

一行同时违反多项规则时只记录一个原因，并按 `field_count` → `invalid_timestamp` → `invalid_method` → `invalid_path` → `invalid_status` → `invalid_duration` 的顺序选择第一个失败项；`line_too_long` 在字段解析前判定。

## 递进需求

1. 逐行读取日志文件并解析记录。
2. 统计总请求数、各状态码数量、访问次数最多的路径和平均耗时。
3. 对坏行记录行号和原因，继续分析剩余内容。
4. 支持把报告输出为终端文本或 JSON；JSON 至少包含 `total_requests`、`status_counts`、`top_path`、`average_duration_ms` 和 `invalid_lines`。`invalid_lines` 是对象数组，每项固定包含从 1 开始的整数 `line` 和字符串 `reason`。
5. 使用基准测试测量大文件处理，再决定是否引入并发解析。
6. 如果并发确实更快，加入固定 Worker Pool，并保证最终统计没有数据竞争。

## 完成条件

- 空文件、合法文件和包含坏行的文件都有自动化测试。
- 统计结果可由小型手算样例验证，坏行不会污染正确记录。
- `invalid_lines` 必须按行号升序输出；没有坏行时是 `[]`，不能是 `null`。`reason` 使用稳定代码，只允许 `field_count`、`invalid_timestamp`、`invalid_method`、`invalid_path`、`invalid_status`、`invalid_duration`、`line_too_long`、`read_error`。
- 每个逻辑行最多 `1 MiB`（`1048576` 字节，不含行尾 `\n` 以及可选的 `\r`）。超长行记为 `line_too_long`，丢弃到该行结尾后继续；若丢弃超长行直到换行的过程中发生非 EOF 读取错误，则该行只记为 `read_error` 并立即停止。其他非 EOF 读取错误同样记为当前逻辑行的 `read_error`，输出此前的部分报告并停止读取。
- `average_duration_ms` 是 JSON 数字，表示所有合法记录耗时的算术平均值。用 `uint64` 保存请求数和精确总耗时，每次累加前检查 `math.MaxUint64-total < duration`，计数也必须检查溢出；溢出时报告致命统计错误并返回 `1`，不能回绕。全部读取完成后用商和余数计算 `total/count`，避免先把整个 `uint64 total` 转成浮点数，再按 `math.Round(value*100)/100` 保留最多两位小数；此结果与输入记录顺序无关。没有合法记录时固定为 `0`，`total_requests` 为 `0`、`status_counts` 为空对象、`top_path` 为 `""`，`invalid_lines` 仍按上述规则输出。
- 多个路径并列第一时使用 Go 字符串 `<` 比较，选择字节字典序最小的路径。
- 读取大文件时按行处理，不要求把整个文件一次载入内存；普通解析坏行不会中止分析。
- 在基准数据证明有收益前，不为了使用 goroutine 强行并发。
- `go test ./...`、`go vet ./...` 和需要并发时的 `go test -race ./...` 通过。

并发网址检测器完成后再开始本项目；当前不要创建实现文件。

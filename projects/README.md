# 实战项目路线

这里包含三个互相独立、难度递进的 Go 项目。它们共用根目录的 `go.mod`，但彼此不导入业务代码。

## 1. Task API（进行中）

入口：[01-task-api/README.md](01-task-api/README.md)

先用 Go 标准库完成一个任务管理 HTTP API，逐步练习内存数据、CRUD、测试、并发安全、JSON 文件持久化和服务工程化。

```powershell
go run ./projects/01-task-api
go test ./projects/01-task-api
```

## 2. 并发网址检测器（待开始）

入口：[02-url-checker/README.md](02-url-checker/README.md)

先写串行版本，再通过 goroutine、channel 和 Worker Pool 控制并发，最后加入超时、取消和数据竞争检查。

## 3. 日志分析器（待开始）

入口：[03-log-analyzer/README.md](03-log-analyzer/README.md)

读取并解析日志，完成统计报告；正确版本稳定后，再比较串行与并发处理的差异。

## 仓库级验证

在仓库根目录执行：

```powershell
go test ./...
go vet ./...
```

按各项目 README 声明的前置里程碑推进：Task API 完成并发安全里程碑后，可以进入并发网址检测器；其余 Task API 里程碑仍需继续完成。

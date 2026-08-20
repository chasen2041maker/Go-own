# 实战项目路线

这里包含四个互相独立、难度递进的 Go 项目。它们共用根目录的 `go.mod`，但彼此不导入业务代码。

## 完整推进顺序

请按下面的顺序学习：

1. 完成 Task API 里程碑 1～5，先建立 HTTP、测试和共享状态并发安全基础。
2. 完成并发网址检测器，集中练习 goroutine、channel、Worker Pool、Context 和超时。
3. 完成日志分析器，练习流式文件处理、坏数据和基准测试。
4. 回到 Task API 完成里程碑 6～7，把已经练过的文件处理和取消知识用于持久化及服务退出。
5. 完成前三个基础项目后，再进入投资内容社区，系统练习 MySQL、鉴权、事务、治理和工程化。

### 1. Task API（进行中）

入口：[01-task-api/README.md](01-task-api/README.md)

先用 Go 标准库完成一个任务管理 HTTP API，逐步练习内存数据、CRUD、测试、并发安全、JSON 文件持久化和服务工程化。

```powershell
go run ./projects/01-task-api
go test ./projects/01-task-api
```

### 2. 并发网址检测器（待开始）

入口：[02-url-checker/README.md](02-url-checker/README.md)

先写串行版本，再通过 goroutine、channel 和 Worker Pool 控制并发，最后加入超时、取消和数据竞争检查。

### 3. 日志分析器（待开始）

入口：[03-log-analyzer/README.md](03-log-analyzer/README.md)

读取并解析日志，完成统计报告；正确版本稳定后，再比较串行与并发处理的差异。

### 4. 原创投资内容社区与治理系统（开发中）

入口：[04-investment-community/README.md](04-investment-community/README.md)

这是高级个人作品：reference 提供原创完整实现，starter 留给学习者按八阶段重写。项目包含虚构证券标签、圈子、帖子、评论、通知、举报治理和审计；不接入真实行情、交易或公司数据。

```powershell
go test ./projects/04-investment-community/... -count=1
go vet ./projects/04-investment-community/...
```

## 仓库级验证

在仓库根目录执行：

```powershell
go test ./...
go vet ./...
```

每个里程碑都要同步编写并运行本阶段测试，不把测试集中拖到项目最后。

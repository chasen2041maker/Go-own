# Go 实战项目集

这个仓库用真实小项目学习 Go。`projects/` 是当前学习主线，`practice/` 保存以前手写的语法练习；不需要先看完所有知识点，再开始写项目。

## 项目路线

| 项目 | 状态 | 主要训练内容 |
| --- | --- | --- |
| [Task API](projects/01-task-api/README.md) | 进行中 | HTTP、JSON、错误处理、测试、并发安全、文件持久化 |
| [并发网址检测器](projects/02-url-checker/README.md) | 待开始 | goroutine、channel、Worker Pool、Context、超时 |
| [日志分析器](projects/03-log-analyzer/README.md) | 待开始 | 文件读取、解析、统计、并发处理、基准测试 |

详细顺序和统一命令见 [projects/README.md](projects/README.md)。

## 现在开始

确认已经安装 Go：

```powershell
go version
```

启动当前的 Task API：

```powershell
go run ./projects/01-task-api
```

另开一个 PowerShell 窗口检查服务：

```powershell
Invoke-RestMethod http://localhost:8080/health
```

验证整个仓库：

```powershell
go test ./...
go vet ./...
```

## 学习方式

每次只完成 README 中的一个里程碑：先手写，再运行，最后让 GPT 按验收条件审查。项目之间暂时不共享业务代码，也不使用框架、数据库或第三方依赖。

旧练习都保存在 [practice/README.md](practice/README.md)，它们是复习资料，不再决定主学习顺序。

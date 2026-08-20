# Go 实战项目集

这个仓库用真实小项目学习 Go。`projects/` 是当前学习主线，`practice/` 保存以前手写的语法练习；不需要先看完所有知识点，再开始写项目。

## 项目路线

| 项目 | 状态 | 主要训练内容 |
| --- | --- | --- |
| [Task API](projects/01-task-api/README.md) | 进行中 | HTTP、JSON、错误处理、测试、并发安全、文件持久化 |
| [并发网址检测器](projects/02-url-checker/README.md) | 待开始 | goroutine、channel、Worker Pool、Context、超时 |
| [日志分析器](projects/03-log-analyzer/README.md) | 待开始 | 文件读取、解析、统计、并发处理、基准测试 |
| [原创投资内容社区](projects/04-investment-community/README.md) | 实现完成；Docker 冷启动/CI 待验证 | MySQL、JWT、分层、事务、幂等、治理、审计、集成与黑盒验收 |

详细顺序和统一命令见 [projects/README.md](projects/README.md)。

## 现在开始

确认已经安装 Go：

```powershell
go version
```

本仓库最低需要 Go 1.26，命令输出应为 `go1.26` 或更高版本。旧版 Go 可能尝试自动下载所需工具链；如果下载失败，请先检查网络和 `GOTOOLCHAIN` 配置，不要把工具链错误误认为项目代码错误。

启动当前的 Task API：

```powershell
go run ./projects/01-task-api
```

另开一个 PowerShell 窗口检查服务：

```powershell
Invoke-RestMethod http://127.0.0.1:8080/health
```

当前学习服务只监听本机回环地址 `127.0.0.1:8080`，局域网中的其他设备无法直接访问。

验证整个仓库：

```powershell
go test ./...
go vet ./projects/04-investment-community/...
```

投资社区的本地 test/vet/build/integration/acceptance 已验证；Docker 冷启动与 CI 仍待具备 Docker Buildx 权限的环境验证。全仓 `go vet ./...` 另有既有练习基线失败：`practice/07-functions/answers/exercise.go:70` 不可达代码，因此高级项目使用上面的项目范围 vet 作为门禁。

## 学习方式

每次只完成 README 中的一个里程碑：先手写，再运行，最后让 GPT 按验收条件审查。项目之间不共享业务代码。前三个基础项目坚持标准库；第四个高级项目只引入 MySQL Driver、JWT 和 bcrypt 所需的最小安全依赖，不引入 Web 框架。

旧练习都保存在 [practice/README.md](practice/README.md)，它们是复习资料，不再决定主学习顺序。

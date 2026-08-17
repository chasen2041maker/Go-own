# 手写练习存档

这里完整保存重构前的 Go 练习代码。每个包含 `main()` 的程序都放在独立目录中，因此可以单独运行，也不会在仓库级测试时互相冲突。

| 目录 | 内容 |
| --- | --- |
| `01-hello` | 最小程序、函数和测试 |
| `02-welcome` | 控制台输出 |
| `03-variables` | 变量与常量声明 |
| `04-constants` | 常量组与基础计算 |
| `05-constant-functions` | 常量表达式、`len` 与 `unsafe.Sizeof` |
| `06-control-flow` | 条件、循环、`range` 与 `switch` |
| `07-functions` | 函数参数和返回值 |
| `08-struct-methods` | 结构体、方法与指针接收者 |
| `09-interfaces` | 接口与多种实现 |
| `10-worker-pool` | goroutine、channel 和 `WaitGroup` |
| `11-http-server` | `net/http` 与 JSON 响应 |

运行一个练习，例如：

```powershell
go run ./practice/06-control-flow
```

验证全部练习：

```powershell
go test ./practice/...
go vet ./practice/...
```

这些代码用于回顾，不再按章节继续扩建。新的学习内容放到 `projects/` 的真实需求中。

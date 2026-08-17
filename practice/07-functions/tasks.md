# 函数练习任务

完成以下 4 个任务，掌握函数的参数、返回值和 defer。

---

## 任务 1：多返回值 - 除法函数

**需求**：
创建一个函数 `divide(a, b float64) (float64, error)`：
- 如果 b 为 0，返回 `0` 和错误信息 "division by zero"
- 否则返回除法结果和 `nil`（表示没有错误）

在 main 中测试：
- `divide(10, 2)` → 应该返回 5.0, nil
- `divide(10, 0)` → 应该返回 0, error

**预期输出**：
```
10 / 2 = 5.00
10 / 0: division by zero
```

**提示**：
- Go 的错误处理模式：函数返回 `(结果, error)`
- 使用 `errors.New("错误信息")` 创建错误
- 需要导入 `errors` 包

---

## 任务 2：可变参数 - 求和函数

**需求**：
创建一个函数 `sum(numbers ...int) int`：
- 接受任意数量的整数参数
- 返回它们的总和
- 如果没有参数，返回 0

在 main 中测试：
- `sum()` → 0
- `sum(5)` → 5
- `sum(1, 2, 3, 4, 5)` → 15

**预期输出**：
```
sum() = 0
sum(5) = 5
sum(1,2,3,4,5) = 15
```

**提示**：
- `...int` 表示可变参数（variadic parameter）
- 在函数内部，`numbers` 是一个 `[]int` 切片
- 使用 for-range 遍历

---

## 任务 3：命名返回值 - 矩形面积和周长

**需求**：
创建一个函数 `rectangle(width, height float64) (area, perimeter float64)`：
- 使用**命名返回值**
- 计算面积：width × height
- 计算周长：2 × (width + height)
- 使用裸 return（直接写 `return`，不带变量名）

在 main 中测试：`rectangle(5, 3)`

**预期输出**：
```
矩形 (宽5, 高3): 面积=15.00, 周长=16.00
```

**提示**：
- 命名返回值在函数签名中声明：`func name() (result int)`
- 可以直接给命名返回值赋值
- 裸 return 会返回所有命名的返回值

---

## 任务 4：defer 延迟执行

**需求**：
创建一个函数 `processFile(filename string)`：
1. 打印 "打开文件: [filename]"
2. 使用 `defer` 打印 "关闭文件: [filename]"（会在函数结束时执行）
3. 打印 "处理文件: [filename]"
4. 模拟一个错误：打印 "发生错误！"
5. 提前返回

观察 defer 语句是否在 return 之前执行。

**预期输出**：
```
打开文件: data.txt
处理文件: data.txt
发生错误！
关闭文件: data.txt
```

**提示**：
- `defer` 语句会在函数返回前执行（无论正常返回还是因错误返回）
- 多个 defer 按照 LIFO（后进先出）顺序执行
- 常用于资源清理：关闭文件、释放锁、关闭连接等

---

## 如何完成

1. 在 `exercise.go` 文件中编写函数
2. 在 `main()` 中调用测试
3. 运行 `go run .`
4. 遇到困难查看 `answers/`

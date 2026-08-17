# 结构体和方法练习任务

完成以下 4 个任务，掌握结构体定义、方法接收者和嵌套结构体。

---

## 任务 1：定义和初始化结构体

**需求**：
定义一个 `Book` 结构体，包含以下字段：
- Title (string) - 书名
- Author (string) - 作者
- Pages (int) - 页数
- Price (float64) - 价格

使用三种方式创建并打印：
1. 字段名初始化（推荐）
2. 按顺序初始化
3. 零值初始化（var book Book）

**预期输出**：
```
书籍1: {Title:Go语言编程 Author:张三 Pages:300 Price:59.9}
书籍2: {Title:Effective Go Author:李四 Pages:250 Price:45}
书籍3 (零值): {Title: Author: Pages:0 Price:0}
```

**提示**：
- `type Book struct { ... }` 定义结构体
- `%+v` 格式化输出结构体，显示字段名
- 零值初始化：string="", int=0, float64=0.0

---

## 任务 2：值接收者方法

**需求**：
定义一个 `Point` 结构体，表示二维坐标点：
- X (int)
- Y (int)

为它添加一个**值接收者**方法 `Distance() float64`：
- 计算该点到原点 (0,0) 的距离
- 使用勾股定理：distance = √(x² + y²)

测试点 (3, 4)，距离应该是 5.0

**预期输出**：
```
点 (3, 4) 到原点的距离: 5.00
```

**提示**：
- 值接收者语法：`func (p Point) MethodName() { ... }`
- 使用 `math.Sqrt()` 和 `math.Pow()`
- 需要导入 `math` 包

---

## 任务 3：指针接收者方法

**需求**：
定义一个 `BankAccount` 结构体：
- AccountNumber (string) - 账号
- Balance (float64) - 余额

添加两个**指针接收者**方法：
1. `Deposit(amount float64)` - 存款
2. `Withdraw(amount float64) bool` - 取款，余额不足返回 false

测试场景：
- 初始余额 1000
- 存款 500 → 余额 1500
- 取款 300 → 成功，余额 1200
- 取款 2000 → 失败，余额不变

**预期输出**：
```
初始余额: 1000.00
存款 500 后: 1500.00
取款 300: 成功=true, 余额=1200.00
取款 2000: 成功=false, 余额=1200.00
```

**提示**：
- 指针接收者语法：`func (acc *BankAccount) MethodName() { ... }`
- 指针接收者可以修改原始结构体
- Go 自动解引用，不需要写 `(*acc).Balance`

---

## 任务 4：嵌套结构体

**需求**：
定义两个结构体：

`Address`：
- City (string)
- Street (string)

`Employee`：
- Name (string)
- Age (int)
- Address (Address) - 嵌套的地址信息

添加方法 `FullInfo() string`：
- 返回格式化的完整信息："姓名, 年龄岁, 住在城市 街道"

测试：王五, 30岁, 北京 中关村大街1号

**预期输出**：
```
王五, 30岁, 住在北京 中关村大街1号
```

**提示**：
- 嵌套结构体：一个结构体包含另一个结构体作为字段
- 访问嵌套字段：`emp.Address.City`
- 使用 `fmt.Sprintf()` 格式化字符串并返回

---

## 如何完成

1. 在 `exercise.go` 文件中定义结构体和方法
2. 在 `main()` 中测试
3. 运行 `go run .`
4. 遇到困难查看 `answers/`

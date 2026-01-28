# Uwasa Engine 噂

Uwasa 是一个为高性能、高动态性场景设计的规则引擎。它实现了专用的 DSL（领域特定语言），支持逻辑判断、数学运算

名称取自“噂/ウワサ”（取自《魔法纪录》）

## 特性

- **极致性能**:
    - **字节码虚拟机 (VM)**: 引入基于栈的虚拟机，通过指令融合 (Instruction Fusion) 和变量索引化极大降低运行开销。
    - **无装箱数值系统**: 使用非装箱 (Unboxed) 的 `Value` 类型存储数值，消除接口转换带来的堆分配。
    - **整数快速路径**: 内部数值自动区分 `int64` 与 `float64`，整数运算零转换开销。
    - **对象池化**: 全程复用 `Lexer`, `Parser`, `Context` 及 `Buffer` 对象，在高频场景下接近零分配。
    - **多级编译优化**: 提供常量折叠 (Constant Folding) 和 激进代数简化。
- **静态安全**: 独立再编译器 (Recompiler) 可在运行前检测除零、类型不匹配、不可达代码等错误。
- **基于 Go 1.25**: 采用最新语言特性，支持各种原生数值类型的无缝接入。
- **功能完备**:
    - 逻辑运算: `&&`, `||`, `==`, `!=`, `>`, `<`, `>=`, `<=`
    - 算术运算: `+`, `-`, `*`, `/`, `%`
    - 内置函数: 高性能 `concat` 等字符串处理函数。
    - 流程控制: `if...is...else` (多分支), `if...then` (前置动作)
    - 赋值系统: 支持表达式中直接修改上下文变量并返回。
    - 短路求值: 编译时与运行时双重短路逻辑。
- **Pratt 解析器**: 采用 Pratt 解析算法，能够优雅且高效地处理复杂的表达式优先级。
- **并发安全**: 编译后的引擎实例是只读且并发安全的。

## 安装

```bash
go get github.com/kamihama-railway/uwasa
```

## 快速开始

```go
package main

import (
	"fmt"
	"uwasa"
)

func main() {
	// 定义规则：支持逻辑或、括号优先级、赋值以及字符串操作
	input := `if (score >= 90 || attendance > 0.9) && status == "active" then bonus = 100`

	engine, _ := uwasa.NewEngine(input)

	// 准备上下文变量
	vars := map[string]any{
		"score":      95,
		"attendance": 0.8,
		"status":     "active",
	}

	// 执行
	result, err := engine.Execute(vars)
	if err != nil {
		panic(err)
	}

	fmt.Printf("Result: %v, New Bonus: %v\n", result, vars["bonus"])
}
```

## 语法概览

### 1. 条件判断
直接返回布尔值结果。
`if a == 0 || b >= 1`

### 2. 多层分支 (If-Is-Else)
顺序匹配条件并返回对应的结果。
`if a == 0 is "yes" else if a == 1 is "ok" else is "bad"`

### 3. 前置条件动作 (If-Then)
满足条件时执行后续计算或赋值。
`if user.is_vip then discount = 0.8`

### 4. 优先级说明
引擎严格遵循以下优先级（从高到低）：
1. 括号 `()`, 函数调用 `fn()`
2. 一元运算符 `-`
3. 算术乘法 `*`, `/`, `%`
4. 算术加法 `+`, `-`
5. 比较运算 `==`, `>`, `<`, `>=`, `<=`
6. 逻辑与 `&&`
7. 逻辑或 `||`
8. 赋值 `=`

## 技术细节

更多关于引擎架构、优化手段（整数快径、分层优化）以及详细的返回逻辑说明，请参阅文档：
- [设计文档](docs/design.md)
- [虚拟机设计与优化](docs/vm.md)
- [技术规格 (必读)](docs/technical_spec.md)
- [Recompiler 深度解析](docs/recompiler.md)
- [使用指南](docs/usage.md)
- [开发手册](docs/dev.md)

## 开源协议

GNU Affero General Public License, version 3.0 (AGPL-3.0)

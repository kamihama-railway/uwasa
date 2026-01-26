# Uwasa Engine 噂

Uwasa 是一个为高性能、高动态性场景设计的规则引擎。它实现了专用的 DSL（领域特定语言），支持复杂的逻辑判断、数学运算以及变量赋值。

名称取自“噂/ウワサ”（取自《魔法纪录》）。

## 特性

- **高性能**: 极致的内存优化，通过对象池化（`sync.Pool`）和零分配 Token 识别，将 GC 压力降至最低。
- **专为 Go 设计**: 原生支持 Go 1.25+，支持多种数值类型的自动转换（int, float32, float64 等）。
- **功能完备**:
    - 逻辑运算: `&&`, `||`, `==`, `>`, `<`, `>=`, `<=`, `true`, `false`
    - 算术运算: `+`, `-`, 字符串拼接
    - 流程控制: `if...is...else` (多分支), `if...then` (前置动作)
    - 副作用支持: 支持在规则中直接进行变量赋值。
    - 短路求值: `&&` 和 `||` 均支持短路逻辑。
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
1. 括号 `()`
2. 一元运算符 `-`
3. 算术运算 `+`, `-`
4. 比较运算 `==`, `>`, `<`, `>=`, `<=`
5. 逻辑与 `&&`
6. 逻辑或 `||`
7. 赋值 `=`

## 技术细节

更多关于引擎架构、优化手段（对象池、装箱优化等）以及开发指南，请参阅文档：
- [设计文档](docs/design.md)
- [开发手册](docs/dev.md)
- [使用指南](docs/usage.md)

## 开源协议

GNU Affero General Public License, version 3.0 (AGPL-3.0)

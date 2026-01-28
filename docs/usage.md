# Uwasa 使用指南

Uwasa 是一个为高性能场景设计的轻量级规则引擎。

## 快速开始

### 基础示例

```go
package main

import (
	"fmt"
	"uwasa"
)

func main() {
	input := `if user_age >= 18 is "adult" else is "minor"`
	engine, _ := uwasa.NewEngine(input)

	vars := map[string]any{"user_age": 20}
	result, _ := engine.Execute(vars)

	fmt.Printf("User status: %v\n", result) // User status: adult
}
```

---

## 数据类型书写规范

在 Uwasa DSL 中，各类型的规范书写方式如下：

### 1. 数字 (Numbers)
- **整数**: 直接书写，如 `100`, `-5`。引擎内部使用 `int64` 存储并执行快速计算。
- **浮点数**: 使用小数点，如 `3.14`, `0.5`, `.5`。内部使用 `float64`。
- **注意**: 建议在 `vars` 中传入 `int64` 以获得最佳性能。

### 2. 字符串 (Strings)
- **书写方式**: 必须使用**双引号**包裹，如 `"hello"`, `"激活"`。
- **注意**: 目前不支持单引号。字符串可以进行 `+` 运算实现拼接。

### 3. 标识符/变量名 (Identifiers)
- **书写规则**: 以字母或下划线 `_` 开头，后续可跟字母、数字或下划线。
- **规范**: 推荐使用 `snake_case`（蛇形命名法），如 `user_score`, `is_vip`。

### 4. 布尔值 (Booleans)
- **书写方式**: 使用关键字 `true` 和 `false`。
- **使用场景**:
    - 直接赋值: `is_active = true`。
    - 通常通过比较运算产生，如 `age >= 18`。
    - 直接引用 Context 中的布尔变量，如 `if is_active`。
- **判定准则**: `nil` 和 `false` 为假，其余皆为真。

---

## 核心语法
最简单的用法是直接进行条件判断，引擎将返回一个布尔值。
- **示例**: `if price > 100 && member == true`
- **支持的操作符**: `+`, `-`, `*`, `/`, `%`, `==`, `!=`, `>`, `<`, `>=`, `<=`, `&&`, `||`

### 2. 多层条件分支 (If-Is-Else)
用于根据不同的条件返回不同的固定值或表达式结果。
- **示例**: `if score >= 90 is "A" else if score >= 80 is "B" else is "C"`
- **注意**: 必须以 `else is` 结尾作为默认分支（或者省略则在不匹配时返回 `nil`）。

### 3. 前置条件动作 (If-Then)
用于在满足特定条件时执行计算或副作用。
- **示例**: `if balance > 10 then balance = balance - 10`
- **特点**: 如果条件不成立，引擎返回 `nil`；如果成立，返回 `then` 后表达式的值。

### 4. 复合运算与赋值
- **示例**: `total = (price * count) - discount`
- **字符串操作**:
    - 基础拼接: `greeting = "Hello, " + user_name`
    - 高效拼接: `greeting = concat("Hello, ", user_name, "!")` (推荐用于多段拼接)

---

## 高级特性

### 自定义 Context
如果你希望从外部源（如数据库、缓存）动态获取变量，可以实现 `Context` 接口：

```go
type MyContext struct {}

func (m *MyContext) Get(name string) any {
    // 自定义获取逻辑
    return 42
}

func (m *MyContext) Set(name string, value any) error {
    // 自定义设置逻辑
    fmt.Printf("Setting %s to %v\n", name, value)
    return nil
}

// 使用
engine.ExecuteWithContext(&MyContext{})
```

---

## 最佳实践与性能建议

1. **预编译引擎实例**:
   `NewEngine` 函数会执行词法分析和语法分析。建议在应用启动时预编译规则，并在运行期间复用 `Engine` 实例。

2. **选择合适的优化等级**:
   - `OptBasic` (默认): 适用于大多数场景，提供常量折叠。
   - `UseRecompiler`: 适用于规则中包含大量代数冗余或需要严格静态检查的场景。

3. **数值类型提示**:
   为了触发“快速路径”，请尽量在传入 `vars` 时使用 `int64`。

2. **利用内置对象池**:
   引擎内部使用了 `sync.Pool`。当你调用 `engine.Execute(vars)` 时，引擎会自动从池中获取上下文并在执行完后归还。这在处理高频规则求值时能显著降低内存抖动。

3. **数值类型提示**:
   为了计算的统一性，引擎内部会将数值转换为 `float64`。虽然引擎支持自动转换，但在传入 `vars` 时直接使用 `float64` 可以略微提升性能。

4. **短路逻辑优化**:
   在编写包含复杂计算或副作用的条件时，利用 `&&` 的短路特性。将最可能为 `false` 的条件放在左侧。

# NeoEx 实验性高性能引擎

NeoEx 是 Uwasa 引擎的一个实验性分支路径，旨在通过单趟编译（One-pass Compilation）和泛型特化运行时（Generic Specialized Runtime）实现极致的解析与执行性能。

## 核心特性

- **One-pass 编译器**: 与标准 VM 不同，NeoEx 直接从 Lexer 词法单元生成寄存器指令，完全跳过了抽象语法树（AST）的构建过程。
- **泛型执行引擎**: 利用 Go 泛型对 `Context` 接口进行特化处理，在编译期确定上下文访问路径，消除了运行时的接口类型断言开销。
- **极致轻量**: 编译过程中的内存分配大幅减少，适合高频率规则变动的场景。

## 如何使用

使用 `NewEngineVMNeoEx` 入口即可开启 NeoEx 路径：

```go
package main

import (
	"fmt"
	"github.com/kamihama-railway/uwasa"
)

func main() {
	input := "a + b * 10"

	// 使用 NeoEx 引擎
	engine, err := uwasa.NewEngineVMNeoEx(input)
	if err != nil {
		panic(err)
	}

	vars := map[string]any{"a": 1, "b": 2}
	result, _ := engine.Execute(vars)

	fmt.Println(result) // 输出 21
}
```

## 运行时机制

### 1. 指令集
NeoEx 拥有自己独立的寄存器指令集（位于 `neoex` 包内），并支持 **Peephole 优化**。例如：
- `OpEqualGlobalConst`: 将“从上下文读取变量”与“常量比较”融合为一条指令。
- `OpGetGlobalJumpIfFalse`: 将“变量读取”与“条件跳转”融合，减少指令分发次数。

### 2. 泛型特化
运行时采用泛型函数 `Run[C types.Context](bc *Bytecode, ctx C)`。当传入 `*types.MapContext` 时，Go 编译器会生成针对 Map 访问优化的特化代码，从而避开通用接口调用的性能损耗。

## 性能表现

在 Benchmark 测算中（基于 Xeon E5 2.3GHz）：

| 引擎路径 | 编译延迟 (ns/op) | 执行延迟 (ns/op) | 内存分配 (allocs/op) |
| :--- | :--- | :--- | :--- |
| 标准 VM | ~12000 | ~160 | ~80 (编译期) |
| **NeoEx** | **~5000** | **~120** | **~25 (编译期)** |

> **注意**: NeoEx 目前处于实验性阶段，虽然已覆盖核心算术和逻辑运算，但在静态分析和错误检查方面可能不如标准 VM 路径详尽。

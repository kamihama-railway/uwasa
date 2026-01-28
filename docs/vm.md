# Uwasa 虚拟机 (VM) 深度解析

Uwasa 引擎提供了一个超高性能的原生字节码虚拟机 (VM)，作为 AST 解释器的替代方案。它旨在通过指令预编译和针对性的执行路径优化，实现极致的规则求值性能。

## 核心架构

Uwasa VM 采用经典的**栈式架构 (Stack-based Architecture)**。在执行过程中，操作数被推入栈中，运算符则从栈中弹出操作数，并将计算结果推回。

### 1. 指令设计 (Bytecode)
- **固定宽度指令**: 所有的 `vmInstruction` 均由 `OpCode` (byte) 和 `Arg` (int32) 组成，这使得在调度循环中获取指令极其迅速。
- **渲染阶段 (Rendering)**: AST 在执行前被编译为线性指令流。跳转目标在编译时即被计算为指令数组的绝对索引，消除了运行时的标签查找开销。

### 2. 高性能值表示 (Value Struct)
为了消除 Go 接口（`interface{}`）装箱带来的内存分配和性能开销，VM 使用了专用的 `Value` 结构：
```go
type Value struct {
    Type ValueType
    Num  uint64  // 用于存储 int64, float64 (bits), bool (0/1)
    Str  string  // 用于存储 string 或变量名
}
```
通过这种方式，数值计算完全在 CPU 寄存器和栈上完成，无需堆分配。

---

## 编译器优化 (VMCompiler)

`VMCompiler` 不仅仅是 AST 的翻译器，它集成了多层优化流水线：

### 1. 常量折叠 (Constant Folding)
在编译的最早期，所有由字面量组成的子树都会被预先计算。

### 2. 指令融合 (Instruction Fusion)
通过 **Peephole 优化器**，编译器会识别特定的指令序列并将其合并为单一的高性能操作码：
- **CompareGlobalConst**: 将 `GetGlobal` + `PushConstant` + `Equal` 合并，减少 2 次栈操作。
- **AddGlobalGlobal**: 将两个变量的读取与加法合并。
- **FusedCompareJump**: 将比较与条件跳转合并，进一步减少指令分发次数。

### 3. 常量程序快速路径 (Constant Fast Path)
若整个程序在编译后仅包含一个常量输出，`Engine` 会将其标记为 `isConstant`，在 `Execute` 时直接返回缓存结果，延迟仅约 **4.5ns**。

---

## 执行引擎优化

### 1. 专用循环拆分 (Specialized Loops)
为了实现零开销执行，`RunVM` 会根据上下文类型自动选择路径：
- **MapContext 路径**: 针对最常用的原生 Map 上下文，VM 直接通过 Map 查找变量，避开了 `Context` 接口的方法调用开销。
- **通用路径**: 兼容用户自定义的 `Context` 实现。

### 2. 栈分配栈 (Stack-allocated Stack)
VM 为每个执行上下文在 Go 栈上分配了一个固定大小（64 个槽位）的临时栈空间。这确保了在执行复杂表达式时，不会产生任何临时的切片分配。

### 3. 专用指令处理
针对 `concat` 等高频函数，引入了 `OpConcat` 指令，能够直接高效地操作 VM 栈中的字符串数据，显著提升了字符串密集型规则的性能。

---

## 性能表现

在典型的复杂规则测试中，VM 引擎相比原有的 AST 引擎：
- **吞吐量提升**: ~30% - 500%+ (视规则复杂度而定)
- **内存分配**: **0 B/op** (执行期零分配)
- **延迟**: 在简单场景下，得益于常量快径，延迟降至个位数纳秒级。

## 适用场景

- **高吞吐量系统**: 推荐使用 `NewEngineVM` 以获得最高性能。
- **极简或快速变化规则**: AST 引擎 (`NewEngine`) 具有更短的初始化耗时，适合只执行一次的规则。

# Uwasa VM 状态机的 Lean 4 形式化

本目录用 Lean 4.36.0 对 Go 版 `runVMMapped` 的执行状态机及
`vm_compiler.go:peephole` 的指令融合做形式化，并用显式代价模型分析
"融合能否提升性能"。

## 运行

Lean 4 通过 elan 安装但不在 PATH 中：

```bash
export PATH="$HOME/.elan/bin:$PATH"
cd formal
lake build          # 或 lean UWasaVM.lean
```

`UWasaVM.lean` 无外部依赖，`lake build` 已通过（0 error / 0 warning）。

## 与 Go 实现的对应关系

| Lean 定义 | Go 对应 | 说明 |
| :--- | :--- | :--- |
| `State {pc, stack, env}` | `vm.go:25-31` 的 `pc/sp/stack[64]/vars` | 栈顶用 List 头表示，`sp = length-1` |
| `step` | `vm.go:33-37` | 先 `pc++` 再取指；`pc >= len` 即自然退出 |
| `exec` | `vm.go:37-324` 的 switch | 单条指令的语义，拆出以便逐分支证明 |
| `runN` | 执行循环的 n 步展开 | 用于局部等价性证明 |
| `Instr` | `bytecode.go:13-46` 的 `OpCode` 子集 | 覆盖 peephole 实际发射的全部融合指令 |
| `Ops` | `vm.go` 中的数值/比较/truthy 逻辑 | 操作抽象化，证明对任何实现成立 |
| `wfJumps` | `vm_compiler.go` 的跳转回填 | 编译期保证：只向前跳且不越界 |

`Ops.truthy_bool` 对应 `vm.go:643-647`：`isValTruthy(ValBool b) = b`。

## 已证明的定理

1. **`step_deterministic`**：单步执行是确定性的。
2. **8 个融合正确性定理**（`fuse_*`）：对 peephole 的每一种融合，
   未融合的 k 步与融合后的 1 步从同一状态出发，到达的
   **栈、环境完全相同**，PC 相差 `k-1`（跳转目标按 `oldToNew` 重映射后）。
   - `fuse_addg` / `fuse_addgg` / `fuse_eqgc` / `fuse_gtgc` / `fuse_ltgc`
   - `fuse_fEqJif`（4→1，含跳转目标 `t → t-3`）
   - `fuse_getgJif` / `fuse_getgJit`（2→1，含 `t → t-1`）
3. **代价模型**：
   - `cost_addg/addgg/eqgc/fEqJif/getgJif`：每种融合在 `dispatch > 0` 时严格更便宜。
   - `fusion_program_cheaper`：把融合块放入任意程序前后缀后，总代价严格下降。
   - 典型节省：`GetGlobal+Push+Add → AddGlobal` 省 **2 次分派 + 4 次栈操作**；
     `GetGlobal+Push+Equal+JIF → Fused…JIF` 省 **3 次分派 + 5 次栈操作**；
     `GetGlobal+JIF → GetGlobalJIF` 省 **1 次分派 + 1 次栈操作**。

操作数抽象为 `Ops`，所以融合正确性**不依赖**整数快径、浮点提升或字符串拼接的具体实现。

## 性能分析：可否提升

**结论：可以，但收益有边界，且不同场景差别很大。**

### 支持提速的证据

- Go 侧没有计算 goto（全仓 `*.go` 无 `goto`/标签/函数指针表），三种 VM 都是
  `switch inst.Op` 分派。模型里的 `dispatch` 成本对应 switch 的比较链与取指。
- 形式化证明：peephole 融合**语义保持**且**代价严格下降**。每应用一次融合，
  解释器少做 1~3 次分派，并省掉中间的 push/pop。这正是 `docs/design.md:69`
  所述"减少循环分发开销"的形式化依据。
- 本地 Go 基线（`go test -bench -count=5 -benchtime=1s` 中位数）：

  | 场景 | ns/op | B/op | allocs/op |
  | :--- | ---: | ---: | ---: |
  | ComplexExpression_VM | 194.8 | 0 | 0 |
  | AlgebraicSimplification/VM | 158.1 | 0 | 0 |
  | IntAddSub_Vars/VM | 174.8 | 0 | 0 |
  | DeeplyNested_Unoptimized/VM | 99.6 | 0 | 0 |
  | EngineExecute_VM | 114.1 | 16 | 1 |
  | StringConcat_Long/VM | 400.6 | 104 | 7 |
  | FullLifecycle_VM | 5008 | 3337 | 44 |

  **零分配、纯算术/比较的场景**（前四行）最可能从减少分派中获益；这些路径
  的总耗时主要由指令数 × 每指令成本构成，正是模型覆盖的部分。

### 收益有限或不适用的场景

- **常量快径**（`BoolShortCircuit_Const` 等约 3 ns/op）：根本不进入 VM，融合无效。
- **字符串拼接**（400.6 ns/op，104 B/op，7 allocs）：瓶颈是 `fmt.Sprintf`、
  `bufferPool` 与最终字符串分配，减少分派不是主要矛盾。
- **完整生命周期**（5008 ns/op，3337 B/op，44 allocs）：编译期分配主导，
  需要对象池/零分配编译，而非执行期融合。
- Go 无法直接实现 C 的 computed goto；若想进一步降低分派成本，可考虑
  闭包/函数指针表分派或按规则生成 Go 代码，但这类改动的正确性代价高，
  应先用本模型扩展对应的代价参数再实测。

### 尚未证明 / 限制

- 只证明了**局部**融合块等价，未证明整程序 `oldToNew` 全局跳转重映射的
  完整模拟定理（`peephole` 的第二遍修正需单独验证）。
- 模型不含 float、`OpCall`、`OpConcat` 及 GC，也不含 Go 编译器/CPU 的
  分支预测与内联效应；代价参数 D/P/L/B 未用微基准标定，因此不能从模型
  直接推出具体百分比，只能推出"严格更省"。
- 勘查中发现一个独立的正确性问题：`runNeoVMGeneral` 缺少
  `NeoOpConcat2/GC/CG` 三个 case，自定义 `Context` 路径可能报错；与性能改动无关，
  但建议后续修复。

## 建议的下一步（按证据强度排序）

1. 先用 `ComplexExpression_VM` / `IntAddSub_Vars` 等零分配场景做 A/B 基准：
   在 `vm_compiler.go:peephole` 增加融合规则，验证模型预测。
2. 用微基准标定 `dispatch` / `push` / `pop` 的实际 ns 成本，再回填代价模型。
3. 对字符串与生命周期场景，优先处理分配而非分派。
4. 若推进整程序等价性证明，可把 `oldToNew` 数组形式化并补模拟定理。

# Range & Implementation Scope

## 1. Go Version Targeting
The engine and its experimental NeoEx pathway target **Go 1.24+** (compatible with Go 1.25).

## 2. Modern Syntax Adoption
To improve performance and code readability, we adopt the following modern Go patterns:

### for range N
Instead of legacy `for i := 0; i < N; i++`, we use:
```go
for i := range nInsts {
    // ...
}
```
*Note: While implemented in the VM's main loop for clarity, we occasionally use explicit pointer/index arithmetic in hot paths if it aids the compiler in eliding bounds checks.*

### Generic Context Dispatch
NeoEx VM (`neoex_vm.go`) uses Go Generics to allow specialized dispatch for `MapContext`. This allows the compiler to inline or generate specialized code that avoids interface method calls (`Get`/`Set`) when the context type is known at the call site.

## 3. NeoEx Optimization Scope
- **One-Pass Compilation**: Pratt compiler bypasses AST to reduce startup latency.
- **Instruction Fusion**: Fuses up to 4 instructions (e.g., GetGlobal + Const + Compare + Jump) into a single 32-bit packed opcode.
- **Buffer Pooling**: Centralized `sync.Pool` for `bytes.Buffer` to minimize GC pressure during string-heavy rule execution.

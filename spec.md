# 定义
一个高动态性 高专用性的规则引擎 名称取自噂/ウワサ(取自魔法纪录)
# 实现定义
## 关键字
- if 条件式开头
- is 条件式结果赋值关键字
- else if 多层条件式下一条件关键字
- else 多层条件式unmatch时结果关键字
- then 若前置判断条件式为true则执行then后的计算
- = 非条件式中的结果赋值关键字(可用于修改变量列表中变量的值)
- + 非条件式中的加法计算关键字
- - 非条件式中的减法计算关键字
- * 非条件式中的乘法计算关键字
- / 非条件式中的除法计算关键字
- % 非条件式中的取模计算关键字
- > 比较计算关键字
- < 比较计算关键字
- >= 比较计算关键字
- <= 比较计算关键字
- == 相等计算关键字
- != 不等于计算关键字
- ! 逻辑非计算关键字
## 引擎运行相关
引擎实例创建前 传入一个 map[string]any 变量列表 实例返回值为any
### 变量操作
操作变量需要通过这样一个接口
type Context interface {
    Get(name string) any
    Set(name string, value any) error
}
## 例子
### 条件判断式
if a == 0 && b >=1
若条件成立则返回 true 不成立则返回false 忽略 is true
### 多层条件式
if a == 0 is "yes" else if a == 1 is "ok"  else is "bad"
多层条件式 顺序链条多可能结果
### 前置条件式
if a == 0 then b + 10
若前置条件判断式为true 则执行then后的计算式
### 计算赋值式
b = b + 10

### 内置函数式
res = concat("Hello ", name)

### 虚拟机执行 (高性能)
```go
// 默认使用栈式虚拟机
engine, _ := uwasa.NewEngine("if a > 10 then b = 1")
result, _ := engine.Execute(vars)

// 实验性：切换到寄存器式虚拟机 (RVM)
engine.UseRegisterVM()
result, _ = engine.Execute(vars)
```

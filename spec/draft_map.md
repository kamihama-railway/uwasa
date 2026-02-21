# Map实现草案
## 引入关键字
- map[]
- add
- del
- has
- get
- set
### 关键字详解
#### map[]
以下写法用于表述一个map
```uwasa
map[map变量名]
```
#### add
以下写法用于向map中添加元素
```uwasa
map[map变量名] = map[map变量名] add [key,value]
```
#### del
以下写法用于从map中删除元素
```uwasa
map[map变量名] = map[map变量名] del key
```
#### has
以下写法用于判断map中是否存在某个key
```uwasa
bool = map[map变量名] has key
```
#### get
以下写法用于获取map中的某个key的值
```uwasa
value = map[map变量名] get key
```
#### set
以下写法用于设置map中的某个key的值
```uwasa
map[map变量名] = map[map变量名] set [key,value]
```

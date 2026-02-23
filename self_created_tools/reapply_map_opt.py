import re

def add_mgetc_to_vm(filename, op_name, is_neo=False):
    with open(filename, "r") as f:
        content = f.read()

    if "OpMapGetConst" in content:
        return # Already added

    if is_neo:
        mgetc_code = r'''case NeoOpMapGetConst:
			obj := &stack[sp]
			if obj.Type != ValMap { return nil, fmt.Errorf("MGETC: not a map") }
			m := obj.Ptr.(map[string]any)
			val := m[(*Value)(unsafe.Add(unsafe.Pointer(pConsts), uintptr(inst.Arg)*valSize)).Str]
			switch v := val.(type) {
			case int64: *obj = Value{Type: ValInt, Num: uint64(v)}
			case int: *obj = Value{Type: ValInt, Num: uint64(int64(v))}
			case float64: *obj = Value{Type: ValFloat, Num: math.Float64bits(v)}
			case string: *obj = Value{Type: ValString, Str: v}
			case bool: *obj = Value{Type: ValBool, Num: boolToUint64(v)}
			case nil: *obj = Value{Type: ValNil}
			default: *obj = FromInterface(v)
			}'''
        op_prefix = "NeoOp"
    else:
        mgetc_code = r'''case OpMapGetConst:
			obj := &stack[sp]
			if obj.Type != ValMap { return nil, fmt.Errorf("MGETC: not a map") }
			m := obj.Ptr.(map[string]any)
			val := m[consts[inst.Arg].Str]
			switch v := val.(type) {
			case int64: *obj = Value{Type: ValInt, Num: uint64(v)}
			case int: *obj = Value{Type: ValInt, Num: uint64(int64(v))}
			case float64: *obj = Value{Type: ValFloat, Num: math.Float64bits(v)}
			case string: *obj = Value{Type: ValString, Str: v}
			case bool: *obj = Value{Type: ValBool, Num: boolToUint64(v)}
			case nil: *obj = Value{Type: ValNil}
			default: *obj = FromInterface(v)
			}'''
        op_prefix = "Op"

    target = "case " + op_prefix + "MapDel:"
    if target in content:
        content = content.replace(target, mgetc_code + "\n\t\t" + target)

    with open(filename, "w") as f:
        f.write(content)

add_mgetc_to_vm("vm.go", "OpMapDel")
add_mgetc_to_vm("neoex_vm.go", "NeoOpMapDel", is_neo=True)

def update_compilers():
    # Update VMCompiler
    with open("vm_compiler.go", "r") as f:
        content = f.read()
    if "OpMapGetConst" not in content:
        content = re.sub(r'case \*MemberCallExpression:\n\s+err := c\.walk\(n\.Object\)',
                        r'''case *MemberCallExpression:
		if n.Method == "get" && len(n.Arguments) == 1 {
			if lit, ok := n.Arguments[0].(*StringLiteral); ok {
				err := c.walk(n.Object)
				if err != nil { return err }
				c.emit(OpMapGetConst, c.addConstant(Value{Type: ValString, Str: lit.Value}))
				return nil
			}
		}
		err := c.walk(n.Object)''', content)
        with open("vm_compiler.go", "w") as f:
            f.write(content)

    # Update NeoCompiler
    with open("neoex_compiler.go", "r") as f:
        content = f.read()
    if "NeoOpMapGetConst" not in content:
        # This one is trickier due to my previous partial changes.
        # I'll just look for method == "get"
        new_get_path = r'''	case "get":
		if numArgs != 1 { return compilationValue{}, fmt.Errorf("get expects 1 argument") }
		if firstArgConst && firstArgVal.Type == ValString {
			c.emit(NeoOpMapGetConst, c.addConstant(firstArgVal))
		} else {
			if firstArgConst { c.emitPush(firstArgVal) }
			c.emit(NeoOpMapGet, 0)
		}'''
        content = re.sub(r'case "get":\n\s+if numArgs != 1 \{.*?c\.emit\(NeoOpMapGet, 0\)\n\s+\}', new_get_path, content, flags=re.DOTALL)
        with open("neoex_compiler.go", "w") as f:
            f.write(content)

update_compilers()

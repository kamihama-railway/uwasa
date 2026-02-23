import re

def fix_neo_compiler(content):
    new_func = r'''func (c *NeoCompiler) parseMemberCallExpression(left compilationValue) (compilationValue, error) {
	if left.isConst {
		return compilationValue{}, fmt.Errorf("member call subject must be an identifier")
	}

	lastInst := c.instructions[len(c.instructions)-1]
	if lastInst.Op != NeoOpGetGlobal {
		return compilationValue{}, fmt.Errorf("member call subject must be an identifier")
	}

	c.nextToken() // cur is identifier (method name)
	if c.curToken.Type != TokenIdent {
		return compilationValue{}, fmt.Errorf("expected method name after '.'")
	}
	method := c.curToken.Literal

	if c.peekToken.Type != TokenLParen {
		return compilationValue{}, fmt.Errorf("expected '(' after method name")
	}
	c.nextToken() // cur is '('

	numArgs := 0
	var firstArgConst bool
	var firstArgVal Value
	if c.peekToken.Type != TokenRParen {
		c.nextToken()
		val, err := c.parseExpression(LOWEST)
		if err != nil {
			return compilationValue{}, err
		}
		firstArgConst = val.isConst
		firstArgVal = val.val
		numArgs++
		for c.peekToken.Type == TokenComma {
			if firstArgConst {
				c.emitPush(firstArgVal)
				firstArgConst = false
			}
			c.nextToken()
			c.nextToken()
			val, err = c.parseExpression(LOWEST)
			if err != nil {
				return compilationValue{}, err
			}
			if val.isConst {
				c.emitPush(val.val)
			}
			numArgs++
		}
	}
	if c.peekToken.Type != TokenRParen {
		return compilationValue{}, fmt.Errorf("expected ')', got %s", c.peekToken.Type)
	}
	c.nextToken()

	switch method {
	case "get":
		if numArgs != 1 {
			return compilationValue{}, fmt.Errorf("get expects 1 argument")
		}
		if firstArgConst && firstArgVal.Type == ValString {
			c.emit(NeoOpMapGetConst, c.addConstant(firstArgVal))
		} else {
			if firstArgConst {
				c.emitPush(firstArgVal)
			}
			c.emit(NeoOpMapGet, 0)
		}
	case "set":
		if firstArgConst {
			c.emitPush(firstArgVal)
		}
		if numArgs != 2 {
			return compilationValue{}, fmt.Errorf("set expects 2 arguments")
		}
		c.emit(NeoOpMapSet, 0)
	case "has":
		if firstArgConst {
			c.emitPush(firstArgVal)
			firstArgConst = false
		}
		if numArgs != 1 {
			return compilationValue{}, fmt.Errorf("has expects 1 argument")
		}
		c.emit(NeoOpMapHas, 0)
	case "del":
		if firstArgConst {
			c.emitPush(firstArgVal)
			firstArgConst = false
		}
		if numArgs != 1 {
			return compilationValue{}, fmt.Errorf("del expects 1 argument")
		}
		c.emit(NeoOpMapDel, 0)
	default:
		return compilationValue{}, fmt.Errorf("unknown map method: %s", method)
	}

	return compilationValue{isConst: false}, nil
}'''
    # We use a broad regex to catch the whole function regardless of internal formatting
    content = re.sub(r'func \(c \*NeoCompiler\) parseMemberCallExpression\(left compilationValue\) \(compilationValue, error\) \{.*?\}\n\nfunc \(c \*NeoCompiler\) parseIdentifier', new_func + '\n\nfunc (c *NeoCompiler) parseIdentifier', content, flags=re.DOTALL)
    return content

with open("neoex_compiler.go", "r") as f:
    content = f.read()
content = fix_neo_compiler(content)
with open("neoex_compiler.go", "w") as f:
    f.write(content)

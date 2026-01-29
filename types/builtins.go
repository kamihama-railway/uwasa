package types

import (
	"bytes"
	"fmt"
	"sync"
)

type BuiltinFunc func(args ...any) (any, error)

var BufferPool = sync.Pool{
	New: func() any {
		return new(bytes.Buffer)
	},
}

var Builtins = map[string]BuiltinFunc{
	"concat": func(args ...any) (any, error) {
		// 1. Pre-calculate total length
		totalLen := 0
		argStrings := make([]string, len(args))
		for i, arg := range args {
			switch v := arg.(type) {
			case string:
				argStrings[i] = v
			case int64:
				argStrings[i] = fmt.Sprintf("%d", v)
			case float64:
				argStrings[i] = fmt.Sprintf("%g", v)
			case bool:
				argStrings[i] = fmt.Sprintf("%v", v)
			default:
				argStrings[i] = fmt.Sprintf("%v", v)
			}
			totalLen += len(argStrings[i])
		}

		// 2. Use pooled buffer
		buf := BufferPool.Get().(*bytes.Buffer)
		buf.Reset()
		buf.Grow(totalLen)
		for _, s := range argStrings {
			buf.WriteString(s)
		}
		res := buf.String()
		BufferPool.Put(buf)
		return res, nil
	},
}

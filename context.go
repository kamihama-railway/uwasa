// Copyright (c) 2026 WJQserver, Kamihama Railway Group. All rights reserved.
// Licensed under the GNU Affero General Public License, version 3.0 (the "AGPL").

package uwasa

import (
	"sync"
)

// Context 定义了变量操作的接口
type Context interface {
	Get(name string) (val any, exists bool)
	Set(name string, value any) error
}

// MapContext 是 Context 接口的一个简单实现
type MapContext struct {
	vars map[string]any
}

var contextPool = sync.Pool{
	New: func() any {
		return &MapContext{}
	},
}

func NewMapContext(vars map[string]any) *MapContext {
	if vars == nil {
		vars = make(map[string]any)
	}
	ctx := contextPool.Get().(*MapContext)
	ctx.vars = vars
	return ctx
}

func (c *MapContext) Get(name string) (any, bool) {
	val, exists := c.vars[name]
	return val, exists
}

func (c *MapContext) Set(name string, value any) error {
	c.vars[name] = value
	return nil
}

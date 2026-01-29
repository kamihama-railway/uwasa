// Copyright (c) 2026 WJQserver, Kamihama Railway Group. All rights reserved.
// Licensed under the GNU Affero General Public License, version 3.0 (the "AGPL").

package uwasa

import (
	"github.com/kamihama-railway/uwasa/types"
	"sync"
)

type Context = types.Context
type MapContext = types.MapContext

var contextPool = sync.Pool{
	New: func() any {
		return types.NewMapContext(nil)
	},
}

func NewMapContext(vars map[string]any) *MapContext {
	ctx := contextPool.Get().(*MapContext)
	ctx.Reset(vars)
	return ctx
}

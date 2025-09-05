// Copyright 2025 HAProxy Technologies LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
package diff

import (
	"github.com/haproxytech/go-method-gen/internal/common"
	"github.com/haproxytech/go-method-gen/internal/data"
)

func DiffGeneratorForNodeWithDiff(node *data.TypeNode, ctx *data.Ctx) bool {
	if !node.HasDiff {
		return false
	}
	var diffImplementation string
	if node.IsForField() {
		diffImplementation = ctx.LeftSideComparison + "." + node.Name + ".Diff(" + ctx.RightSideComparison + "." + node.Name + ")"
	} else {
		diffImplementation = ctx.LeftSideComparison + ".Diff(" + ctx.RightSideComparison + ")"
	}
	ctxDiff := &data.Ctx{
		DiffImplementation:         diffImplementation,
		ObjectKind:                 data.KindToString(node.Kind),
		ObjectNameToHaveGeneration: node.Name,
		Imports:                    node.Imports,
	}
	ctx.SubCtxs = append(ctx.SubCtxs, ctxDiff)
	return true
}

type DiffCtx struct {
	Overrides map[string]common.OverrideFuncs
}

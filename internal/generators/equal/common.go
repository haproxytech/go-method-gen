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
package equal

import (
	"github.com/haproxytech/go-method-gen/internal/common"
	"github.com/haproxytech/go-method-gen/internal/data"
)

func EqualGeneratorForNodeWithEqual(node *data.TypeNode, ctx *data.Ctx) bool {
	if !node.HasEqual {
		return false
	}
	var equalImplementation, unequalImplementation string
	if node.IsForField() {
		equalImplementation = ctx.LeftSideComparison + "." + node.Name + ".Equal(" + ctx.RightSideComparison + "." + node.Name + ")"
	} else {
		equalImplementation = ctx.LeftSideComparison + ".Equal(" + ctx.RightSideComparison + ")"
	}
	unequalImplementation = "!" + equalImplementation
	ctxEqual := &data.Ctx{
		InequalImplementation:      unequalImplementation,
		EqualImplementation:        equalImplementation,
		ObjectKind:                 data.KindToString(node.Kind),
		ObjectNameToHaveGeneration: node.Name,
		Imports:                    node.Imports,
		DefinedType:                true,
	}
	ctx.SubCtxs = append(ctx.SubCtxs, ctxEqual)
	return true
}

type EqualCtx struct {
	Overrides map[string]common.OverrideFuncs
}

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
	"strings"

	"github.com/haproxytech/go-method-gen/internal/data"
)

func EqualGeneratorStruct(node *data.TypeNode, ctx *data.Ctx, equalCtx EqualCtx) {
	if EqualGeneratorForNodeWithEqual(node, ctx) {
		return
	}

	ctxEqual := &data.Ctx{
		ObjectKind:                 data.KindToString(node.Kind),
		ObjectNameToHaveGeneration: node.Name,
		LeftSideComparison:         "rec",
		RightSideComparison:        "obj",
		EqualFuncName:              "Equal",
		PkgPath:                    node.PkgPath,
		Pkg:                        strings.Split(node.PackagedType, ".")[0],
		Type:                       node.Type,
	}
	ctx.SubCtxs = append(ctx.SubCtxs, ctxEqual)

	for _, field := range node.Fields {
		Generate(field, ctxEqual, equalCtx)
	}

	ctxEqual.Imports = map[string]struct{}{}

	implementation := strings.Builder{}

	numSubCtxs := len(ctxEqual.SubCtxs)
	for i, subCtx := range ctxEqual.SubCtxs {
		if subCtx.Err {
			continue
		}
		if subCtx.ObjectKind != "Struct" {
			for imp, marker := range subCtx.Imports {
				ctxEqual.Imports[imp] = marker
			}
		}
		if i != 0 && i < numSubCtxs {
			implementation.WriteString(" && \n")
		}
		switch {
		case subCtx.EqualFuncName == "Equal":
			implementation.WriteString(ctxEqual.LeftSideComparison + "." +
				subCtx.ObjectNameToHaveGeneration + "." + subCtx.EqualFuncName + "(" +
				ctxEqual.RightSideComparison + "." + subCtx.ObjectNameToHaveGeneration + ")")
		// case subCtx.EqualFuncName != "" && node.HasEqual:
		case subCtx.EqualFuncName != "":
			implementation.WriteString(subCtx.EqualFuncName + "(" + ctxEqual.LeftSideComparison + "." +
				subCtx.ObjectNameToHaveGeneration + "," +
				ctxEqual.RightSideComparison + "." + subCtx.ObjectNameToHaveGeneration + ")")
		default:
			implementation.WriteString(subCtx.EqualImplementation)
		}

	}
	ctxEqual.EqualImplementation = implementation.String()
}

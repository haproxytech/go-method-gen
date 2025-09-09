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
	"strings"

	"github.com/haproxytech/go-method-gen/internal/data"
)

func DiffGeneratorStruct(node *data.TypeNode, ctx *data.Ctx, diffCtx DiffCtx) {
	if DiffGeneratorForNodeWithDiff(node, ctx) {
		return
	}

	ctxDiff := &data.Ctx{
		ObjectKind:                 data.KindToString(node.Kind),
		ObjectNameToHaveGeneration: node.Name,
		LeftSideComparison:         "rec",
		RightSideComparison:        "obj",
		DiffFuncName:               "Diff",
		PkgPath:                    node.PkgPath,
		Pkg:                        strings.Split(node.PackagedType, ".")[0],
		Type:                       node.Type,
	}

	ctx.SubCtxs = append(ctx.SubCtxs, ctxDiff)
	for _, field := range node.Fields {
		Generate(field, ctxDiff, diffCtx)
	}

	ctxDiff.Imports = map[string]struct{}{}

	implementation := strings.Builder{}

	numSubCtxs := len(ctxDiff.SubCtxs)
	for i, subCtx := range ctxDiff.SubCtxs {
		if subCtx.Err {
			continue
		}

		for imp, marker := range subCtx.Imports {
			ctxDiff.Imports[imp] = marker
		}

		if i != 0 && i < numSubCtxs {
			implementation.WriteString("\n")
		}
		keySeparator := "."
		if subCtx.ObjectKind == data.KindToString(data.Slice) ||
			subCtx.ObjectKind == data.KindToString(data.Map) {
			keySeparator = ""
		}
		key := "diffKey"
		if subCtx.ObjectNameToHaveGeneration != "" {
			key = "\"" + subCtx.ObjectNameToHaveGeneration + keySeparator + `"+` + key
		}
		switch {
		case subCtx.DiffFuncName == "Diff":
			implementation.WriteString("for diffKey, diffValue:= range " + ctxDiff.LeftSideComparison + "." +
				subCtx.ObjectNameToHaveGeneration + "." + subCtx.DiffFuncName + "(" +
				ctxDiff.RightSideComparison + "." + subCtx.ObjectNameToHaveGeneration + ") {\n" +
				"\tdiff[" + key + "] = diffValue\n}")
		// case subCtx.DiffFuncName != "" && node.HasDiff:
		case subCtx.DiffFuncName != "":

			implementation.WriteString("for diffKey, diffValue:= range " + subCtx.DiffFuncName + "(" + ctxDiff.LeftSideComparison + "." +
				subCtx.ObjectNameToHaveGeneration + "," +
				ctxDiff.RightSideComparison + "." + subCtx.ObjectNameToHaveGeneration + ") {\n" +
				"\tdiff[" + key + "] = diffValue\n}")
		default:
			implementation.WriteString(subCtx.DiffImplementation)
		}

	}
	ctxDiff.DiffImplementation = implementation.String()
}

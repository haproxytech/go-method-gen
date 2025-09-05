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
	"fmt"
	"strings"
	"text/template"

	"github.com/haproxytech/go-method-gen/internal/data"
	"github.com/haproxytech/go-method-gen/internal/utils"
)

var builtinDiffTemplateTxt = `if {{ .LeftSideComparison }}.{{ .FieldName }} != {{ .RightSideComparison }}.{{ .FieldName }} {
	diff["{{ .FieldName }}"] = []interface{}{ {{ .LeftSideComparison }}.{{ .FieldName }}, {{ .RightSideComparison }}.{{ .FieldName }} }
}`

var diffBuiltinTemplate = template.Must(template.New("DiffBuiltinTemplate").Parse(builtinDiffTemplateTxt))

func DiffGeneratorBuiltin(node *data.TypeNode, ctx *data.Ctx, diffCtx DiffCtx) {
	if node.PkgPath == "" {
		DiffGeneratorBuiltinRaw(node, ctx, diffCtx)
		return
	}
	DiffGeneratorDefinedBuiltin(node, ctx, diffCtx)
}

func DiffGeneratorDefinedBuiltin(node *data.TypeNode, ctx *data.Ctx, diffCtx DiffCtx) {
	if node.Kind != data.Builtin {
		// TODO log error
	}
	if DiffGeneratorForNodeWithDiff(node, ctx) {
		return
	}
	parameterType := data.GetTypeFromNode(node)
	diffFuncName := utils.DiffFuncName(parameterType)
	ctxDiff := &data.Ctx{
		ObjectKind:                 data.KindToString(node.Kind),
		ObjectNameToHaveGeneration: node.Name,
		LeftSideComparison:         "x",
		RightSideComparison:        "y",
		DiffFuncName:               "Diff",
		PkgPath:                    node.PkgPath,
		Pkg:                        strings.Split(node.PackagedType, ".")[0],
		Type:                       node.Type,
		DefinedType:                true,
		DiffImplementation:         diffFuncName + "(x, y)",
		Imports:                    node.Imports,
	}
	if ctxDiff.Imports == nil {
		ctxDiff.Imports = map[string]struct{}{}
	}
	ctxDiff.Imports["fmt"] = struct{}{}
	ctx.SubCtxs = append(ctx.SubCtxs, ctxDiff)

	ctxDiffImpl := &data.Ctx{
		ObjectKind:                 data.KindToString(node.Kind),
		ObjectNameToHaveGeneration: node.Name,
		LeftSideComparison:         "x",
		RightSideComparison:        "y",
		DiffFuncName:               diffFuncName,
		PkgPath:                    node.PkgPath,
		Pkg:                        strings.Split(node.PackagedType, ".")[0],
		Type:                       node.Type,
		Imports:                    node.Imports,
	}
	name := "self"
	if node.Name != "" {
		name = node.Name
	}
	ctxDiff.SubCtxs = append(ctxDiff.SubCtxs, ctxDiffImpl)
	ctxDiffImpl.DiffImplementation = fmt.Sprintf(`func %s (x, y %s) map[string][]interface{} {
		diff := make(map[string][]interface{})
		if x != y {
			diff["%s"] = []interface{}{x, y}
		}
		return diff
}`, diffFuncName, parameterType, name)
}

func DiffGeneratorBuiltinRaw(node *data.TypeNode, ctx *data.Ctx, diffCtx DiffCtx) {
	diffImplementation := strings.Builder{}
	args := map[string]string{
		"LeftSideComparison":  ctx.LeftSideComparison,
		"RightSideComparison": ctx.RightSideComparison,
		"FieldName":           node.Name,
	}
	diffBuiltinTemplate.Execute(&diffImplementation, args)

	ctxDiff := &data.Ctx{
		DiffImplementation:         diffImplementation.String(),
		ObjectNameToHaveGeneration: node.Name,
		ObjectKind:                 data.KindToString(node.Kind),
	}
	ctx.SubCtxs = append(ctx.SubCtxs, ctxDiff)
}

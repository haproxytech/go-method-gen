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
	"text/template"

	"github.com/haproxytech/go-method-gen/internal/data"
	"github.com/haproxytech/go-method-gen/internal/utils"
)

const equalBuiltinDefinedTemplateTxt = `func {{.EqualFuncName}}(x,y {{.ParameterType}}) bool {
	return x == y
}`

var equalBuiltinDefinedTemplate = template.Must(template.New("EqualBuiltinDefinedTemplate").Parse(equalBuiltinDefinedTemplateTxt))

func EqualGeneratorBuiltin(node *data.TypeNode, ctx *data.Ctx, equalCtx EqualCtx) {
	if node.PkgPath == "" {
		EqualGeneratorBuiltinRaw(node, ctx, equalCtx)
		return
	}
	EqualGeneratorBuiltinDefined(node, ctx, equalCtx)
}

func EqualGeneratorBuiltinDefined(node *data.TypeNode, ctx *data.Ctx, equalCtx EqualCtx) {
	if node.Kind != data.Builtin {
		// TODO log error
	}
	if EqualGeneratorForNodeWithEqual(node, ctx) {
		return
	}
	ctxEqual := &data.Ctx{
		ObjectKind:                 data.KindToString(node.Kind),
		ObjectNameToHaveGeneration: node.Name,
		LeftSideComparison:         "x",
		RightSideComparison:        "y",
		EqualFuncName:              "Equal",
		PkgPath:                    node.PkgPath,
		Pkg:                        strings.Split(node.PackagedType, ".")[0],
		Type:                       node.Type,
		DefinedType:                true,
		Imports:                    node.Imports,
	}
	parameterType := data.GetTypeFromNode(node)
	equalFuncName := utils.EqualFuncName(parameterType)
	ctxEqualImpl := &data.Ctx{
		ObjectKind:                 data.KindToString(node.Kind),
		ObjectNameToHaveGeneration: node.Name,
		LeftSideComparison:         "x",
		RightSideComparison:        "y",
		PkgPath:                    node.PkgPath,
		Pkg:                        strings.Split(node.PackagedType, ".")[0],
		Type:                       node.Type,
		EqualFuncName:              equalFuncName,
	}

	var sb strings.Builder
	equalBuiltinDefinedTemplate.Execute(&sb, map[string]string{
		data.EqualFuncNameDataMap: equalFuncName,
		data.ParameterTypeDataMap: parameterType,
	})
	ctxEqualImpl.EqualImplementation = sb.String()
	ctx.SubCtxs = append(ctx.SubCtxs, ctxEqual)
	ctxEqual.SubCtxs = append(ctxEqual.SubCtxs, ctxEqualImpl)
	ctxEqual.EqualImplementation = ctxEqual.SubCtxs[0].EqualFuncName + "(" + ctxEqual.LeftSideComparison + ", " + ctxEqual.RightSideComparison + ")"
}

func EqualGeneratorBuiltinRaw(node *data.TypeNode, ctx *data.Ctx, equalCtx EqualCtx) {
	var equalImplementation, unequalImplementation string
	if node.IsForType() {
		equalImplementation = ctx.LeftSideComparison + " == " + ctx.RightSideComparison
		unequalImplementation = ctx.LeftSideComparison + " != " + ctx.RightSideComparison
	} else {
		equalImplementation = ctx.LeftSideComparison + "." + node.Name + " == " + ctx.RightSideComparison + "." + node.Name
		unequalImplementation = ctx.LeftSideComparison + "." + node.Name + " != " + ctx.RightSideComparison + "." + node.Name
	}
	ctxEqual := &data.Ctx{
		EqualImplementation:        equalImplementation,
		InequalImplementation:      unequalImplementation,
		ObjectNameToHaveGeneration: node.Name,
		ObjectKind:                 data.KindToString(node.Kind),
	}
	ctx.SubCtxs = append(ctx.SubCtxs, ctxEqual)
}

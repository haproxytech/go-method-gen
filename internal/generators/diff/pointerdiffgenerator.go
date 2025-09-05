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
	"text/template"

	"github.com/haproxytech/go-method-gen/internal/data"
)

const diffPointerRawTemplateTxt = `func {{.DiffFuncName}}(x, y {{.ParameterType}}) map[string][]interface{}  {
	diff := make(map[string][]interface{})
` + diffPointerDefinedTemplateTxt + `
}`

const diffPointerDefinedTemplateTxt = `if x == nil && y == nil {
		return diff
	}
	{{ if .NodeName}}
	key := "{{ .NodeName }}"
	{{ else }}
	key := "*{{ .SubType }}"
	{{ end }}
	switch {
	case x == nil:
		diff[key] = []interface{}{x, *y}
		return diff
	case y == nil:
		diff[key] = []interface{}{*x, y}
		return diff
	}

	{{ if  (eq .IsBuiltinSubNode "true") }}
	if *x != *y{
		diff[key] = []interface{}{x, y}
	}
	{{ else }}
	for diffKey, diffValue := range {{.DiffElement}} {
		diff[key+"."+diffKey]=diffValue
	}
	{{ end }}
	return diff`

var diffPointerRawTemplate = template.Must(template.New("DiffPointerRawTemplate").Parse(diffPointerRawTemplateTxt))

func DiffGeneratorPointer(node *data.TypeNode, ctx *data.Ctx, diffCtx DiffCtx) {
	if node.Type == "" {
		DiffGeneratorRawPointer(node, ctx, diffCtx)
		return
	}
	DiffGeneratorDefinedPointer(node, ctx, diffCtx)
}

func DiffGeneratorDefinedPointer(node *data.TypeNode, ctx *data.Ctx, diffCtx DiffCtx) {
	if node.Kind != data.Pointer {
		// TODO log error
	}
	if DiffGeneratorForNodeWithDiff(node, ctx) {
		return
	}
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
		Imports:                    node.Imports,
	}
	ctx.SubCtxs = append(ctx.SubCtxs, ctxDiff)
	DiffGeneratorRawPointer(node, ctxDiff, diffCtx)
	ctxDiff.Err = ctxDiff.SubCtxs[0].Err
	data.ApplyTemplateForDiff(node, ctxDiff, diffPointerRawTemplate)
	ctxDiff.DiffImplementation = ctxDiff.SubCtxs[0].DiffFuncName + "(x, y)"
}

func DiffGeneratorRawPointer(node *data.TypeNode, ctx *data.Ctx, diffCtx DiffCtx) {
	if node.Kind != data.Pointer {
		// TODO log error
	}
	subNode := node.SubNode
	if subNode == nil {
		// TODO log error
	}
	ctxDiff := &data.Ctx{
		ObjectNameToHaveGeneration: node.Name,
		Imports:                    node.Imports,
		LeftSideComparison:         "x",
		RightSideComparison:        "y",
		ObjectKind:                 data.KindToString(node.Kind),
	}
	ctx.SubCtxs = append(ctx.SubCtxs, ctxDiff)
	Generate(subNode, ctxDiff, diffCtx)
	ctxDiff.Err = ctxDiff.SubCtxs[0].Err
	data.ApplyTemplateForDiff(node, ctxDiff, diffPointerRawTemplate)
}

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
	"text/template"

	"github.com/haproxytech/go-method-gen/internal/common"
	"github.com/haproxytech/go-method-gen/internal/data"
)

const diffPointerRawTemplateTxt = `func {{.DiffFuncName}}(x, y {{.ParameterType}}, opts ...eqdiff.GoMethodGenOptions) map[string][]interface{}  {
	diff := make(map[string][]interface{})
` + diffPointerDefinedTemplateTxt + `
}`

const diffPointerDefinedTemplateTxt = `if x == nil && y == nil {
		return diff
	}

	{{ if (eq .IsBuiltinSubNode "true") }}
	switch {
	case x == nil:
		diff[""] = []interface{}{x, *y}
		return diff
	case y == nil:
		diff[""] = []interface{}{*x, y}
		return diff
	}

	if *x != *y {
		diff[""] = []interface{}{*x, *y}
	}

	return diff
	{{ else if and (ne .InnerDiffFuncName "") (ne .InnerDiffFuncName "Diff") }}
	switch {
	case x == nil:
		return {{.InnerDiffFuncName}}(nil, *y)
	case y == nil:
		return {{.InnerDiffFuncName}}(*x, nil)
	}

	for diffKey, diffValue := range {{.DiffElement}} {
		if diffKey != "" {
			diffKey = "." + diffKey
		}
		diff[diffKey] = diffValue
	}

	return diff
	{{ else }}
	switch {
	case x == nil:
		diff[""] = []interface{}{x, *y}
		return diff
	case y == nil:
		diff[""] = []interface{}{*x, y}
		return diff
	}

	for diffKey, diffValue := range {{.DiffElement}} {
		if diffKey != "" && diffKey[0] != '.' && diffKey[0] != '[' {
			diffKey = "." + diffKey
		}
		diff[diffKey] = diffValue
	}

	return diff
	{{ end }}`

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
	subNodeKind := ""
	if node.SubNode != nil {
		subNodeKind = data.KindToString(node.SubNode.Kind)
	}
	ctxDiff := &data.Ctx{
		ObjectKind:                 data.KindToString(node.Kind),
		ObjectNameToHaveGeneration: node.Name,
		SubNodeKind:                subNodeKind,
		LeftSideComparison:         "*x",
		RightSideComparison:        "*y",
		DiffFuncName:               "Diff",
		PkgPath:                    node.PkgPath,
		Pkg:                        common.GetPackage(node),
		Type:                       node.Type,
		DefinedType:                true,
		Imports:                    node.Imports,
	}
	ctx.SubCtxs = append(ctx.SubCtxs, ctxDiff)
	DiffGeneratorRawPointer(node, ctxDiff, diffCtx)
	ctxDiff.Err = ctxDiff.SubCtxs[0].Err
	data.ApplyTemplateForDiff(node, ctxDiff, diffPointerRawTemplate)
	ctxDiff.DiffImplementation = ctxDiff.SubCtxs[0].DiffFuncName + "(x, y, opts...)"
}

func DiffGeneratorRawPointer(node *data.TypeNode, ctx *data.Ctx, diffCtx DiffCtx) {
	if node.Kind != data.Pointer {
		// TODO log error
	}
	subNode := node.SubNode
	if subNode == nil {
		// TODO log error
	}
	subNodeKind := ""
	if subNode != nil {
		subNodeKind = data.KindToString(subNode.Kind)
	}
	ctxDiff := &data.Ctx{
		ObjectNameToHaveGeneration: node.Name,
		Imports:                    node.Imports,
		LeftSideComparison:         "*x",
		RightSideComparison:        "*y",
		ObjectKind:                 data.KindToString(node.Kind),
		SubNodeKind:                subNodeKind,
	}
	ctx.SubCtxs = append(ctx.SubCtxs, ctxDiff)
	Generate(subNode, ctxDiff, diffCtx)
	ctxDiff.Err = ctxDiff.SubCtxs[0].Err
	data.ApplyTemplateForDiff(node, ctxDiff, diffPointerRawTemplate)
}

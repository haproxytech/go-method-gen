package equal

import (
	"text/template"

	"github.com/haproxytech/eqdiff/internal/data"
)

var equalPointerTemplateTxt = `func {{.EqualFuncName}}(x, y {{.ParameterType}}) bool {
	if x == nil || y == nil {
		return x == y
	}
	return {{.EqualityTest}}
}`

var equalPointerTemplate = template.Must(template.New("EqualPointerTemplate").Parse(equalPointerTemplateTxt))

func EqualGeneratorPointer(node *data.TypeNode, ctx *data.Ctx, equalCtx EqualCtx) {
	if node.Type == "" {
		EqualGeneratorPointerRawType(node, ctx, equalCtx)
		return
	}
	EqualGeneratorPointerDefinedType(node, ctx, equalCtx)
}

func EqualGeneratorPointerDefinedType(node *data.TypeNode, ctx *data.Ctx, equalCtx EqualCtx) {
	if node.Kind != data.Pointer {
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
		Type:                       node.Type,
		DefinedType:                true,
		Imports:                    node.Imports,
	}
	ctx.SubCtxs = append(ctx.SubCtxs, ctxEqual)
	EqualGeneratorPointerRawType(node, ctxEqual, equalCtx)
	ctxEqual.EqualImplementation = ctxEqual.SubCtxs[0].EqualFuncName + "(x, y)"
}

func EqualGeneratorPointerRawType(node *data.TypeNode, ctx *data.Ctx, equalCtx EqualCtx) {
	if node.Kind != data.Pointer {
		// TODO log error
	}
	subNode := node.SubNode
	if subNode == nil {
		// TODO log error
	}
	ctxEqual := &data.Ctx{
		ObjectNameToHaveGeneration: node.Name,
		LeftSideComparison:         "x",
		RightSideComparison:        "y",
		ObjectKind:                 data.KindToString(node.Kind),
		Imports:                    node.Imports,
		Type:                       node.Type,
		PkgPath:                    node.PkgPath,
	}
	ctx.SubCtxs = append(ctx.SubCtxs, ctxEqual)
	Generate(subNode, ctxEqual, equalCtx)
	ctxEqual.Err = ctxEqual.SubCtxs[0].Err
	data.ApplyTemplateForEqual(node, ctxEqual, equalPointerTemplate)
}

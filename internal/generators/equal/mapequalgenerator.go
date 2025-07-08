package equal

import (
	"text/template"

	"github.com/haproxytech/eqdiff/internal/data"
)

const equalMapRawTemplateTxt = `func {{.EqualFuncName}}(x, y {{.ParameterType}}) bool {
	if len(x) != len(y) {
		return false
	}

	for kx, vx := range x {
		if vy, exists := y[kx]; !exists || {{.InequalityTest}} {
			return false
		}
	}

	return true
}`

var equalMapRawTemplate = template.Must(template.New("EqualMapRawTemplate").Parse(equalMapRawTemplateTxt))

func EqualGeneratorMap(node *data.TypeNode, ctx *data.Ctx, equalCtx EqualCtx) {
	if node.Type == "" {
		EqualGeneratorRawMap(node, ctx, equalCtx)
		return
	}
	EqualGeneratorDefinedMap(node, ctx, equalCtx)
}

func EqualGeneratorRawMap(node *data.TypeNode, ctx *data.Ctx, equalCtx EqualCtx) {
	if node.Kind != data.Map {
		// TODO log error
	}
	subNode := node.SubNode
	if subNode == nil {
		// TODO log error
	}
	ctxEqual := &data.Ctx{
		ObjectNameToHaveGeneration: node.Name,
		LeftSideComparison:         "vx",
		RightSideComparison:        "vy",
		ObjectKind:                 data.KindToString(node.Kind),
		Imports:                    node.Imports,
		//
		Type:    node.Type,
		PkgPath: node.PkgPath,
	}
	ctx.SubCtxs = append(ctx.SubCtxs, ctxEqual)
	Generate(subNode, ctxEqual, equalCtx)
	data.ApplyTemplateForEqual(node, ctxEqual, equalMapRawTemplate)
}

func EqualGeneratorDefinedMap(node *data.TypeNode, ctx *data.Ctx, equalCtx EqualCtx) {
	if node.Kind != data.Map {
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
	EqualGeneratorRawMap(node, ctxEqual, equalCtx)
	ctxEqual.EqualImplementation = ctxEqual.SubCtxs[0].EqualFuncName + "(x, y)"

}

package equal

import (
	"text/template"

	"github.com/haproxytech/eqdiff/internal/data"
)

var equalArrayTemplateTxt = `func {{.EqualFuncName}}(x, y {{.ParameterType}}) bool {
	for i := range x {
        if {{.InequalityTest}} {
            return false
        }
    }
    return true
}`

var equalArrayTemplate = template.Must(template.New("EqualArrayTemplate").Parse(equalArrayTemplateTxt))

func EqualGeneratorArray(node *data.TypeNode, ctx *data.Ctx, pkgsForGeneration map[string]struct{}) {
	if node.Type == "" {
		EqualGeneratorRawArray(node, ctx, pkgsForGeneration)
		return
	}
	EqualGeneratorDefinedArray(node, ctx, pkgsForGeneration)
}

func EqualGeneratorDefinedArray(node *data.TypeNode, ctx *data.Ctx, pkgsForGeneration map[string]struct{}) {
	if node.Kind != data.Array {
		// TODO log error
	}
	if EqualGeneratorForNodeWithEqual(node, ctx) {
		return
	}

	ctxEqual := &data.Ctx{
		ObjectKind:                 data.KindToString(node.Kind),
		ObjectNameToHaveGeneration: node.Name,
		LeftSideComparison:         "x[i]",
		RightSideComparison:        "y[i]",
		EqualFuncName:              "Equal",
		PkgPath:                    node.PkgPath,
		Type:                       node.Type,
		DefinedType:                true,
		Imports:                    node.Imports,
	}
	ctx.SubCtxs = append(ctx.SubCtxs, ctxEqual)
	EqualGeneratorRawArray(node, ctxEqual, pkgsForGeneration)
	ctxEqual.EqualImplementation = ctxEqual.SubCtxs[0].EqualFuncName + "(x, y)"
}

func EqualGeneratorRawArray(node *data.TypeNode, ctx *data.Ctx, pkgsForGeneration map[string]struct{}) {
	if node.Kind != data.Array {
		// TODO log error
	}
	subNode := node.SubNode
	if subNode == nil {
		// TODO log error
	}
	ctxEqual := &data.Ctx{
		ObjectNameToHaveGeneration: node.Name,
		LeftSideComparison:         "x[i]",
		RightSideComparison:        "y[i]",
		Imports:                    node.Imports,
	}
	ctx.SubCtxs = append(ctx.SubCtxs, ctxEqual)
	EqualGenerator(subNode, ctxEqual, pkgsForGeneration)
	data.ApplyTemplateForEqual(node, ctxEqual, equalArrayTemplate)
}

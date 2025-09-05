package equal

import (
	"strings"
	"text/template"

	"github.com/haproxytech/go-method-gen/internal/data"
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

func EqualGeneratorArray(node *data.TypeNode, ctx *data.Ctx, equalCtx EqualCtx) {
	if node.Type == "" {
		EqualGeneratorRawArray(node, ctx, equalCtx)
		return
	}
	EqualGeneratorDefinedArray(node, ctx, equalCtx)
}

func EqualGeneratorDefinedArray(node *data.TypeNode, ctx *data.Ctx, equalCtx EqualCtx) {
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
		Pkg:                        strings.Split(node.PackagedType, ".")[0],
		Type:                       node.Type,
		DefinedType:                true,
		Imports:                    node.Imports,
	}
	ctx.SubCtxs = append(ctx.SubCtxs, ctxEqual)
	EqualGeneratorRawArray(node, ctxEqual, equalCtx)
	ctxEqual.EqualImplementation = ctxEqual.SubCtxs[0].EqualFuncName + "(x, y)"
}

func EqualGeneratorRawArray(node *data.TypeNode, ctx *data.Ctx, equalCtx EqualCtx) {
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
	Generate(subNode, ctxEqual, equalCtx)
	data.ApplyTemplateForEqual(node, ctxEqual, equalArrayTemplate)
}

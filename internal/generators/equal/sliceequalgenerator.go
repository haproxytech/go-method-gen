package equal

import (
	"text/template"

	"github.com/haproxytech/eqdiff/internal/data"
)

var equalSliceRawTemplateTxt = `func {{.EqualFuncName}}(x, y {{.ParameterType}}) bool {
	if len(x) != len(y) {
		return false
	}

	for i, vx := range x {
		vy := y[i]
		if {{.InequalityTest}} {
			return false
		}
	}

	return true
}`

var equalSliceRawTemplate = template.Must(template.New("EqualSliceRawTemplate").Parse(equalSliceRawTemplateTxt))

func EqualGeneratorSlice(node *data.TypeNode, ctx *data.Ctx, pkgsForGeneration map[string]struct{}) {
	if node.Type == "" {
		EqualGeneratorSliceRawType(node, ctx, pkgsForGeneration)
		return
	}
	EqualGeneratorSliceDefinedType(node, ctx, pkgsForGeneration)
}

func EqualGeneratorSliceRawType(node *data.TypeNode, ctx *data.Ctx, pkgsForGeneration map[string]struct{}) {
	if node.Kind != data.Slice {
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
		Type:                       node.Type,
		PkgPath:                    node.PkgPath,
	}
	ctx.SubCtxs = append(ctx.SubCtxs, ctxEqual)
	EqualGenerator(subNode, ctxEqual, pkgsForGeneration)
	ctxEqual.Err = ctxEqual.SubCtxs[0].Err
	data.ApplyTemplateForEqual(node, ctxEqual, equalSliceRawTemplate)
}

func EqualGeneratorSliceDefinedType(node *data.TypeNode, ctx *data.Ctx, pkgsForGeneration map[string]struct{}) {
	if node.Kind != data.Slice {
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
	EqualGeneratorSliceRawType(node, ctxEqual, pkgsForGeneration)
	ctxEqual.EqualImplementation = ctxEqual.SubCtxs[0].EqualFuncName + "(x, y)"
}

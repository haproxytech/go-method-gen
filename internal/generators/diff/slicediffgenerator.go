package diff

import (
	"text/template"

	"github.com/haproxytech/eqdiff/internal/data"
)

var diffSliceRawTemplateTxt = `func {{.DiffFuncName}}(x, y {{.ParameterType}}) map[string][]interface{}  {
	diff := make(map[string][]interface{})
	lenX := len(x)
	lenY := len(y)

	if (x == nil && y == nil) || (lenX ==0 && lenY ==0) {
		return diff
	}
	
	if x == nil {
		return map[string][]interface{}{"": {nil, y}}
	}
	
	if y == nil {
		return map[string][]interface{}{"": {x, nil}}
	}

	for i := 0; i < lenX && i < lenY; i++ {
		key := fmt.Sprintf("[%d]",i)
		vx, vy := x[i], y[i]

		{{ if  (eq .IsBuiltinSubNode "true") }}
		if vx != vy {
			diff[key] = []interface{}{vx, vy}
		}
		{{ else }}
		for diffKey, diffValue := range {{.DiffElement}} {
			diff[key+"."+diffKey]=diffValue
		}
		{{ end }}
	}

	for i := lenY; i < lenX; i++ {
		key := fmt.Sprintf("[%d]",i)
		diff[key] = []interface{}{x[i], nil}
	}


	for i := lenX; i < lenY; i++ {
		key := fmt.Sprintf("[%d]",i)
		diff[key] = []interface{}{nil, y[i]}
	}

    return diff
}`

var diffSliceRawTemplate = template.Must(template.New("DiffSliceRawTemplate").Parse(diffSliceRawTemplateTxt))

func DiffGeneratorSlice(node *data.TypeNode, ctx *data.Ctx, pkgsForGeneration map[string]struct{}) {
	if node.Type == "" {
		DiffGeneratorSliceRawType(node, ctx, pkgsForGeneration)
		return
	}
	DiffGeneratorSliceDefinedType(node, ctx, pkgsForGeneration)
}

func DiffGeneratorSliceDefinedType(node *data.TypeNode, ctx *data.Ctx, pkgsForGeneration map[string]struct{}) {
	if node.Kind != data.Slice {
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
		Type:                       node.Type,
		DefinedType:                true,
		Imports:                    node.Imports,
	}
	if ctxDiff.Imports == nil {
		ctxDiff.Imports = map[string]struct{}{}
	}
	ctxDiff.Imports["fmt"] = struct{}{}
	ctx.SubCtxs = append(ctx.SubCtxs, ctxDiff)
	DiffGeneratorSliceRawType(node, ctxDiff, pkgsForGeneration)
	ctxDiff.Err = ctxDiff.SubCtxs[0].Err
	ctxDiff.DiffImplementation = ctxDiff.SubCtxs[0].DiffFuncName + "(x, y)"
}

func DiffGeneratorSliceRawType(node *data.TypeNode, ctx *data.Ctx, pkgsForGeneration map[string]struct{}) {
	if node.Kind != data.Slice {
		// TODO log error
	}
	subNode := node.SubNode
	if subNode == nil {
		// TODO log error
	}
	ctxDiff := &data.Ctx{
		ObjectNameToHaveGeneration: node.Name,
		LeftSideComparison:         "vx",
		RightSideComparison:        "vy",
		ObjectKind:                 data.KindToString(node.Kind),
		Imports:                    node.Imports,
		Type:                       node.Type,
		PkgPath:                    node.PkgPath,
	}
	if ctxDiff.Imports == nil {
		ctxDiff.Imports = map[string]struct{}{}
	}
	ctxDiff.Imports["fmt"] = struct{}{}
	ctx.SubCtxs = append(ctx.SubCtxs, ctxDiff)
	DiffGenerator(subNode, ctxDiff, pkgsForGeneration)
	ctxDiff.Err = ctxDiff.SubCtxs[0].Err
	data.ApplyTemplateForDiff(node, ctxDiff, diffSliceRawTemplate)
}

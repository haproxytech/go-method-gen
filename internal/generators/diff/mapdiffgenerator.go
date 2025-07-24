package diff

import (
	"strings"
	"text/template"

	"github.com/haproxytech/eqdiff/internal/data"
)

const diffMapRawTemplateTxt = `func {{.DiffFuncName}}(x, y {{.ParameterType}}) map[string][]interface{}  {
	diff := make(map[string][]interface{})
` + diffMapDefinedTemplateTxt + `
}`

const diffMapDefinedTemplateTxt = `if (x == nil && y == nil) || (len(x) ==0 && len(y) ==0) {
		return diff
	}

	if x == nil {
		return map[string][]interface{}{"": {nil, y}}
	}

	if y == nil {
		return map[string][]interface{}{"": {x, nil}}
	}

	for kx,vx := range x {
		key := fmt.Sprintf("[%v]",kx)
		vy := y[kx]
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
	for ky,vy := range y {
		key := fmt.Sprintf("[%v]",ky)
		if _,found := diff[key]; found {
			continue
		}

		vx := x[ky]
		{{ if  (eq .IsBuiltinSubNode "true") }}
		if vx != vy {
			diff[key] = []interface{}{vx, vy}
		}
		{{ else }}
		for diffKey, diffValue := range {{.DiffElement}} {
			diff[key+"."+diffKey]= []interface{}{ diffValue[1], diffValue[0]}
		}
		{{ end }}

	}
    return diff`

var diffMapRawTemplate = template.Must(template.New("DiffMapRawTemplate").Parse(diffMapRawTemplateTxt))

func DiffGeneratorMap(node *data.TypeNode, ctx *data.Ctx, diffCtx DiffCtx) {
	if node.Type == "" {
		DiffGeneratorRawMap(node, ctx, diffCtx)
		return
	}
	DiffGeneratorDefinedMap(node, ctx, diffCtx)
}

func DiffGeneratorRawMap(node *data.TypeNode, ctx *data.Ctx, diffCtx DiffCtx) {
	if node.Kind != data.Map {
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
	}
	if ctxDiff.Imports == nil {
		ctxDiff.Imports = map[string]struct{}{}
	}
	ctxDiff.Imports["fmt"] = struct{}{}
	ctx.SubCtxs = append(ctx.SubCtxs, ctxDiff)
	Generate(subNode, ctxDiff, diffCtx)
	ctxDiff.Err = ctxDiff.SubCtxs[0].Err
	data.ApplyTemplateForDiff(node, ctxDiff, diffMapRawTemplate)
}

func DiffGeneratorDefinedMap(node *data.TypeNode, ctx *data.Ctx, diffCtx DiffCtx) {
	if node.Kind != data.Map {
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
	if ctxDiff.Imports == nil {
		ctxDiff.Imports = map[string]struct{}{}
	}
	ctxDiff.Imports["fmt"] = struct{}{}
	ctx.SubCtxs = append(ctx.SubCtxs, ctxDiff)
	DiffGeneratorRawMap(node, ctxDiff, diffCtx)
	ctxDiff.Err = ctxDiff.SubCtxs[0].Err
	ctxDiff.DiffImplementation = ctxDiff.SubCtxs[0].DiffFuncName + "(x, y)"
}

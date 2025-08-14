package diff

import (
	"strings"
	"text/template"

	"github.com/haproxytech/gomethodgen/internal/data"
)

var diffArrayTemplateTxt = `func {{.DiffFuncName}}(x, y {{.ParameterType}}) map[string][]interface{}  {
	diff := make(map[string][]interface{})
	for i, vx := range x {
		key := fmt.Sprintf("[%d]{{ .SubType }}",i)
		vy := y[i]
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
    return diff
}`

var diffArrayTemplate = template.Must(template.New("DiffArrayTemplate").Parse(diffArrayTemplateTxt))

func DiffGeneratorArray(node *data.TypeNode, ctx *data.Ctx, diffCtx DiffCtx) {
	if node.Type == "" {
		DiffGeneratorArrayRawType(node, ctx, diffCtx)
		return
	}
	DiffGeneratorArrayDefinedType(node, ctx, diffCtx)
}

func DiffGeneratorArrayDefinedType(node *data.TypeNode, ctx *data.Ctx, diffCtx DiffCtx) {
	if node.Kind != data.Array {
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
	DiffGeneratorArrayRawType(node, ctxDiff, diffCtx)
	ctxDiff.Err = ctxDiff.SubCtxs[0].Err
	ctxDiff.DiffImplementation = ctxDiff.SubCtxs[0].DiffFuncName + "(x, y)"

}

func DiffGeneratorArrayRawType(node *data.TypeNode, ctx *data.Ctx, diffCtx DiffCtx) {
	if node.Kind != data.Array {
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
		Imports:                    node.Imports,
	}
	if ctxDiff.Imports == nil {
		ctxDiff.Imports = map[string]struct{}{}
	}
	ctxDiff.Imports["fmt"] = struct{}{}
	ctx.SubCtxs = append(ctx.SubCtxs, ctxDiff)
	if subNode.Kind != data.Builtin {
		Generate(subNode, ctxDiff, diffCtx)
	}
	data.ApplyTemplateForDiff(node, ctxDiff, diffArrayTemplate)
}

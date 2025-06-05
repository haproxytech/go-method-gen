package diff

import "github.com/haproxytech/eqdiff/internal/data"

func DiffGeneratorFunc(node *data.TypeNode, ctx *data.Ctx, pkgsForGeneration map[string]struct{}) {
	if node.Kind != data.Func {
		// TODO log error
	}
	ctxDiff := &data.Ctx{
		ObjectNameToHaveGeneration: node.Name,
		ObjectKind:                 data.KindToString(node.Kind),
		Imports:                    node.Imports,
		Err:                        true,
	}
	ctx.SubCtxs = append(ctx.SubCtxs, ctxDiff)
}

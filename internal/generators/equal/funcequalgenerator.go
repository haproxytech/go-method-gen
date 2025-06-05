package equal

import "github.com/haproxytech/eqdiff/internal/data"

func EqualGeneratorFunc(node *data.TypeNode, ctx *data.Ctx, pkgsForGeneration map[string]struct{}) {
	if node.Kind != data.Func {
		// TODO log error
	}
	ctxEqual := &data.Ctx{
		ObjectNameToHaveGeneration: node.Name,
		ObjectKind:                 data.KindToString(node.Kind),
		Imports:                    node.Imports,
		Err:                        true,
	}
	ctx.SubCtxs = append(ctx.SubCtxs, ctxEqual)
}

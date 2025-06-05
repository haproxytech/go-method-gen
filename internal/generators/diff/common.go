package diff

import "github.com/haproxytech/eqdiff/internal/data"

func DiffGeneratorForNodeWithDiff(node *data.TypeNode, ctx *data.Ctx) bool {
	if !node.HasDiff {
		return false
	}
	var diffImplementation string
	if node.IsForField() {
		diffImplementation = ctx.LeftSideComparison + "." + node.Name + ".Diff(" + ctx.RightSideComparison + "." + node.Name + ")"
	} else {
		diffImplementation = ctx.LeftSideComparison + ".Diff(" + ctx.RightSideComparison + ")"
	}
	ctxDiff := &data.Ctx{
		DiffImplementation:         diffImplementation,
		ObjectKind:                 data.KindToString(node.Kind),
		ObjectNameToHaveGeneration: node.Name,
		Imports:                    node.Imports,
	}
	ctx.SubCtxs = append(ctx.SubCtxs, ctxDiff)
	return true
}

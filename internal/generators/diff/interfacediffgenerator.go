package diff

import "github.com/haproxytech/gomethodgen/internal/data"

func DiffGeneratorInterface(node *data.TypeNode, ctx *data.Ctx, diffCtx DiffCtx) {
	if node.Kind != data.Interface {
		// TODO log error
	}

	var equalImplementation, unequalImplementation string
	if node.IsForType() {
		equalImplementation = ctx.LeftSideComparison + " == " + ctx.RightSideComparison
	} else {
		equalImplementation = ctx.LeftSideComparison + "." + node.Name + " == " + ctx.RightSideComparison + "." + node.Name
	}
	unequalImplementation = "!" + equalImplementation
	ctxDiff := &data.Ctx{
		ObjectNameToHaveGeneration: node.Name,
		EqualImplementation:        equalImplementation,
		InequalImplementation:      unequalImplementation,
		ObjectKind:                 data.KindToString(node.Kind),
		Imports:                    node.Imports,
		Err:                        true,
	}
	ctx.SubCtxs = append(ctx.SubCtxs, ctxDiff)
}

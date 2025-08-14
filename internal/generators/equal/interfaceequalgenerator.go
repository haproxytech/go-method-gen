package equal

import "github.com/haproxytech/gomethodgen/internal/data"

func EqualGeneratorInterface(node *data.TypeNode, ctx *data.Ctx, equalCtx EqualCtx) {
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
	ctxEqual := &data.Ctx{
		ObjectNameToHaveGeneration: node.Name,
		EqualImplementation:        equalImplementation,
		InequalImplementation:      unequalImplementation,
		ObjectKind:                 data.KindToString(node.Kind),
		Imports:                    node.Imports,
		Err:                        true,
	}
	ctx.SubCtxs = append(ctx.SubCtxs, ctxEqual)
}

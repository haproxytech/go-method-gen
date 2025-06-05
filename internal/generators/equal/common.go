package equal

import "github.com/haproxytech/eqdiff/internal/data"

func EqualGeneratorForNodeWithEqual(node *data.TypeNode, ctx *data.Ctx) bool {
	if !node.HasEqual {
		return false
	}
	var equalImplementation, unequalImplementation string
	if node.IsForField() {
		equalImplementation = ctx.LeftSideComparison + "." + node.Name + ".Equal(" + ctx.RightSideComparison + "." + node.Name + ")"
	} else {
		equalImplementation = ctx.LeftSideComparison + ".Equal(" + ctx.RightSideComparison + ")"
	}
	unequalImplementation = "!" + equalImplementation
	ctxEqual := &data.Ctx{
		InequalImplementation:      unequalImplementation,
		EqualImplementation:        equalImplementation,
		ObjectKind:                 data.KindToString(node.Kind),
		ObjectNameToHaveGeneration: node.Name,
		Imports:                    node.Imports,
		DefinedType:                true,
	}
	ctx.SubCtxs = append(ctx.SubCtxs, ctxEqual)
	return true
}

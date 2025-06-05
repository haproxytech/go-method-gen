package equal

import (
	"github.com/haproxytech/eqdiff/internal/data"
)

func EqualGenerator(node *data.TypeNode, ctx *data.Ctx, pkgsForGeneration map[string]struct{}) {
	if node == nil {
		return
	}
	switch node.Kind {
	case data.Struct:
		EqualGeneratorStruct(node, ctx, pkgsForGeneration)
	case data.Builtin:
		EqualGeneratorBuiltin(node, ctx, pkgsForGeneration)
	case data.Array:
		EqualGeneratorArray(node, ctx, pkgsForGeneration)
	case data.Slice:
		EqualGeneratorSlice(node, ctx, pkgsForGeneration)
	case data.Map:
		EqualGeneratorMap(node, ctx, pkgsForGeneration)
	case data.Interface:
		EqualGeneratorInterface(node, ctx, pkgsForGeneration)
	}
}

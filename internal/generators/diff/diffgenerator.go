package diff

import (
	"github.com/haproxytech/eqdiff/internal/data"
)

func DiffGenerator(node *data.TypeNode, ctx *data.Ctx, pkgsForGeneration map[string]struct{}) {
	if node == nil {
		return
	}
	switch node.Kind {
	case data.Struct:
		DiffGeneratorStruct(node, ctx, pkgsForGeneration)
	case data.Builtin:
		DiffGeneratorBuiltin(node, ctx, pkgsForGeneration)
	case data.Array:
		DiffGeneratorArray(node, ctx, pkgsForGeneration)
	case data.Slice:
		DiffGeneratorSlice(node, ctx, pkgsForGeneration)
	case data.Map:
		DiffGeneratorMap(node, ctx, pkgsForGeneration)
	case data.Interface:
		DiffGeneratorInterface(node, ctx, pkgsForGeneration)
	}
}

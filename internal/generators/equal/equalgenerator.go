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
	}
}

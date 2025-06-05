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
	}
}

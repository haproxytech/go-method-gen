package diff

import (
	"strings"

	"github.com/haproxytech/eqdiff/internal/data"
	"github.com/haproxytech/eqdiff/internal/utils"
)

func Generate(node *data.TypeNode, ctx *data.Ctx, diffCtx DiffCtx) {
	if node == nil {
		return
	}
	if node.Err {
		return
	}

	packagedType := node.PkgPath + "." + node.Type
	override, hasOverride := diffCtx.Overrides[packagedType]

	if hasOverride && override.Diff != nil {
		fn := override.Diff
		ctxDiff := &data.Ctx{
			ObjectKind:                 data.KindToString(node.Kind),
			ObjectNameToHaveGeneration: node.Name,
			LeftSideComparison:         "x",
			RightSideComparison:        "y",
			PkgPath:                    node.PkgPath,
			Pkg:                        strings.Split(node.PackagedType, ".")[0],
			Type:                       node.Type,
		}
		ctx.SubCtxs = append(ctx.SubCtxs, ctxDiff)
		if node.UpNode == nil {
			ctxDiff.DiffFuncName = fn.Name
			ctxDiff.DiffImplementation =
				"for diffKey, diffValue:= range " +
					utils.ExtractPkg(fn.Pkg) + "." + fn.Name + "(rec, obj)" + "{\n" +
					"\tdiff[\"" + node.Type + "\"] = diffValue\n}"
		} else {
			ctxDiff.DiffFuncName = utils.ExtractPkg(fn.Pkg) + "." + fn.Name
		}

		if ctxDiff.Imports == nil {
			ctxDiff.Imports = make(map[string]struct{})
		}
		ctxDiff.Imports[fn.Pkg] = struct{}{}
		return
	}
	switch node.Kind {
	case data.Struct:
		DiffGeneratorStruct(node, ctx, diffCtx)
	case data.Builtin:
		DiffGeneratorBuiltin(node, ctx, diffCtx)
	case data.Array:
		DiffGeneratorArray(node, ctx, diffCtx)
	case data.Slice:
		DiffGeneratorSlice(node, ctx, diffCtx)
	case data.Map:
		DiffGeneratorMap(node, ctx, diffCtx)
	case data.Interface:
		DiffGeneratorInterface(node, ctx, diffCtx)
	case data.Pointer:
		DiffGeneratorPointer(node, ctx, diffCtx)
	case data.Func:
		DiffGeneratorFunc(node, ctx, diffCtx)
	}
}

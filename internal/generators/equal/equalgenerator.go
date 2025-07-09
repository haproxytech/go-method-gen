package equal

import (
	"github.com/haproxytech/eqdiff/internal/data"
	"github.com/haproxytech/eqdiff/internal/utils"
)

func Generate(node *data.TypeNode, ctx *data.Ctx, equalCtx EqualCtx) {
	if node == nil {
		return
	}
	if node.Err {
		return
	}

	packagedType := node.PkgPath + "." + node.Type
	override, hasOverride := equalCtx.Overrides[packagedType]
	if hasOverride && override.Equal != nil {
		fn := override.Equal
		ctxEqual := &data.Ctx{
			ObjectKind:                 data.KindToString(node.Kind),
			ObjectNameToHaveGeneration: node.Name,
			LeftSideComparison:         "x",
			RightSideComparison:        "y",
			PkgPath:                    node.PkgPath,
			Type:                       node.Type,
		}
		ctx.SubCtxs = append(ctx.SubCtxs, ctxEqual)
		if node.UpNode == nil {
			ctxEqual.EqualFuncName = fn.Name
			ctxEqual.EqualImplementation = utils.ExtractPkg(fn.Pkg) + "." + fn.Name + "(rec, obj)"
		} else {
			ctxEqual.EqualFuncName = utils.ExtractPkg(fn.Pkg) + "." + fn.Name
		}

		if ctxEqual.Imports == nil {
			ctxEqual.Imports = make(map[string]struct{})
		}
		ctxEqual.Imports[fn.Pkg] = struct{}{}
		return
	}

	switch node.Kind {
	case data.Struct:
		EqualGeneratorStruct(node, ctx, equalCtx)
	case data.Builtin:
		EqualGeneratorBuiltin(node, ctx, equalCtx)
	case data.Array:
		EqualGeneratorArray(node, ctx, equalCtx)
	case data.Slice:
		EqualGeneratorSlice(node, ctx, equalCtx)
	case data.Map:
		EqualGeneratorMap(node, ctx, equalCtx)
	case data.Interface:
		EqualGeneratorInterface(node, ctx, equalCtx)
	case data.Pointer:
		EqualGeneratorPointer(node, ctx, equalCtx)
	case data.Func:
		EqualGeneratorFunc(node, ctx, equalCtx)
	}
}

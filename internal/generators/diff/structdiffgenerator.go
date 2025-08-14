package diff

import (
	"strings"

	"github.com/haproxytech/gomethodgen/internal/data"
)

func DiffGeneratorStruct(node *data.TypeNode, ctx *data.Ctx, diffCtx DiffCtx) {
	if DiffGeneratorForNodeWithDiff(node, ctx) {
		return
	}

	ctxDiff := &data.Ctx{
		ObjectKind:                 data.KindToString(node.Kind),
		ObjectNameToHaveGeneration: node.Name,
		LeftSideComparison:         "rec",
		RightSideComparison:        "obj",
		DiffFuncName:               "Diff",
		PkgPath:                    node.PkgPath,
		Pkg:                        strings.Split(node.PackagedType, ".")[0],
		Type:                       node.Type,
	}

	ctx.SubCtxs = append(ctx.SubCtxs, ctxDiff)
	for _, field := range node.Fields {
		Generate(field, ctxDiff, diffCtx)
	}

	ctxDiff.Imports = map[string]struct{}{}

	implementation := strings.Builder{}

	numSubCtxs := len(ctxDiff.SubCtxs)
	for i, subCtx := range ctxDiff.SubCtxs {
		if subCtx.Err {
			continue
		}
		if subCtx.ObjectKind != data.KindToString(data.Struct) {
			for imp, marker := range subCtx.Imports {
				ctxDiff.Imports[imp] = marker
			}
		}
		if i != 0 && i < numSubCtxs {
			implementation.WriteString("\n")
		}
		keySeparator := "."
		if subCtx.ObjectKind == data.KindToString(data.Slice) ||
			subCtx.ObjectKind == data.KindToString(data.Map) {
			keySeparator = ""
		}
		key := "diffKey"
		if subCtx.ObjectNameToHaveGeneration != "" {
			key = "\"" + subCtx.ObjectNameToHaveGeneration + keySeparator + `"+` + key
		}
		switch {
		case subCtx.DiffFuncName == "Diff":
			implementation.WriteString("for diffKey, diffValue:= range " + ctxDiff.LeftSideComparison + "." +
				subCtx.ObjectNameToHaveGeneration + "." + subCtx.DiffFuncName + "(" +
				ctxDiff.RightSideComparison + "." + subCtx.ObjectNameToHaveGeneration + ") {\n" +
				"\tdiff[" + key + "] = diffValue\n}")
		// case subCtx.DiffFuncName != "" && node.HasDiff:
		case subCtx.DiffFuncName != "":

			implementation.WriteString("for diffKey, diffValue:= range " + subCtx.DiffFuncName + "(" + ctxDiff.LeftSideComparison + "." +
				subCtx.ObjectNameToHaveGeneration + "," +
				ctxDiff.RightSideComparison + "." + subCtx.ObjectNameToHaveGeneration + ") {\n" +
				"\tdiff[" + key + "] = diffValue\n}")
		default:
			implementation.WriteString(subCtx.DiffImplementation)
		}

	}
	ctxDiff.DiffImplementation = implementation.String()
}

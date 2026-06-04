// Copyright 2025 HAProxy Technologies LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
package diff

import "github.com/haproxytech/go-method-gen/internal/data"

var diffInterfaceTemplateTxt = `func DiffInterface( x,y interface{}, opts ...eqdiff.GoMethodGenOptions) map[string][]interface{} {
	var opt *eqdiff.GoMethodGenOptions
	if len(opts) > 0 {
		opt = &opts[0]
	}
	diffs := map[string][]interface{}{}
	if opt != nil && !opt.CompareInterfaces {
		return diffs
	}

	if x == nil && y == nil {
		return diffs
	}

	if x == nil {
		return map[string][]interface{}{"": {nil, y}}
	}

	if y == nil {
		return map[string][]interface{}{"": {x, nil}}
	}

	diff := cmp.Diff(x, y)
	if diff != "" {
		diffs[""] = []interface{}{x, y, diff}
	}
	return diffs
}
`

func DiffGeneratorInterface(node *data.TypeNode, ctx *data.Ctx, diffCtx DiffCtx) {
	if node.Kind != data.Interface {
		// TODO log error
	}
	if !diffCtx.EnableCompareInterfaces {
		return
	}
	if node.Imports == nil {
		node.Imports = make(map[string]struct{})
	}
	node.Imports["github.com/google/go-cmp/cmp"] = struct{}{}

	ctxDiff := &data.Ctx{
		ObjectNameToHaveGeneration: node.Name,
		ObjectKind:                 data.KindToString(node.Kind),
		Imports:                    node.Imports,
		DiffImplementation:         diffInterfaceTemplateTxt,
		DiffFuncName:               "DiffInterface",
	}
	ctx.SubCtxs = append(ctx.SubCtxs, ctxDiff)
}

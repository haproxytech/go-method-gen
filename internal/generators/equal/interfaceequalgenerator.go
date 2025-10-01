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
package equal

import (
	"strings"

	"github.com/haproxytech/go-method-gen/internal/data"
)

var equalInterfaceTemplateTxt = `func EqualInterface( x,y interface{}, opts ...eqdiff.GoMethodGenOptions) bool {

	var opt *eqdiff.GoMethodGenOptions
	if len(opts) > 0 {
		opt = &opts[0]
	}

	if (x== nil) != (y== nil) {
		if opt == nil || !opt.TreatNilNotAsEmpty {
			return true
		}
		return false
	}

	if opt == nil || !opt.CompareInterfaces {
		return true
	}

	return reflect.DeepEqual(x,y)
}
`

func EqualGeneratorInterface(node *data.TypeNode, ctx *data.Ctx, equalCtx EqualCtx) {
	if node.Kind != data.Interface {
		// TODO log error
	}

	var equalImplementation, unequalImplementation string

	if equalCtx.EnableCompareInterfaces {
		equalImplementation = equalInterfaceTemplateTxt

	} else {
		equalImplementation = ctx.LeftSideComparison + " == " + ctx.RightSideComparison
	}

	if node.Imports == nil {
		node.Imports = make(map[string]struct{})
	}
	node.Imports["reflect"] = struct{}{}
	ctxEqual := &data.Ctx{
		ObjectNameToHaveGeneration: node.Name,
		ObjectKind:                 data.KindToString(node.Kind),
		Imports:                    node.Imports,
		EqualFuncName:              "EqualInterface",
		EqualImplementation:        equalImplementation,
		InequalImplementation:      unequalImplementation,
		Type:                       node.Type,
		PkgPath:                    node.PkgPath,
		Pkg:                        strings.Split(node.PackagedType, ".")[0],
	}
	ctx.SubCtxs = append(ctx.SubCtxs, ctxEqual)
}

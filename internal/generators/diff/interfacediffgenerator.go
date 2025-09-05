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

func DiffGeneratorInterface(node *data.TypeNode, ctx *data.Ctx, diffCtx DiffCtx) {
	if node.Kind != data.Interface {
		// TODO log error
	}

	var equalImplementation, unequalImplementation string
	if node.IsForType() {
		equalImplementation = ctx.LeftSideComparison + " == " + ctx.RightSideComparison
	} else {
		equalImplementation = ctx.LeftSideComparison + "." + node.Name + " == " + ctx.RightSideComparison + "." + node.Name
	}
	unequalImplementation = "!" + equalImplementation
	ctxDiff := &data.Ctx{
		ObjectNameToHaveGeneration: node.Name,
		EqualImplementation:        equalImplementation,
		InequalImplementation:      unequalImplementation,
		ObjectKind:                 data.KindToString(node.Kind),
		Imports:                    node.Imports,
		Err:                        true,
	}
	ctx.SubCtxs = append(ctx.SubCtxs, ctxDiff)
}

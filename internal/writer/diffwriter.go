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
package writer

import (
	"bytes"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/haproxytech/go-method-gen/internal/data"
)

// diffTemplateRawTxt defines the Go function template for generating a Diff method
// when the type is a struct. The generated function builds a diff map by executing
// the provided implementation code inside it.
const diffTemplateRawTxt = `func ({{.LeftSideComparison}} {{.Type}}) Diff({{.RightSideComparison}} {{.Type}}) map[string][]interface{} {
	diff := make(map[string][]interface{})
	{{.DiffImplementation}}
	return diff
}
`

// diffTemplateDefinedTxt defines the Go function template for generating a Diff method
// for defined types (type aliases). In this case, the implementation is expected to
// return the diff map directly, so no initialization code is included.
const diffTemplateDefinedTxt = `func ({{.LeftSideComparison}} {{.Type}}) Diff({{.RightSideComparison}} {{.Type}}) map[string][]interface{} {
	return {{.DiffImplementation}}
}
`

// diffTemplateRaw is the parsed template object for struct-based Diff generation.
var diffTemplateRaw = template.Must(template.New("DiffTemplate").Parse(diffTemplateRawTxt))

// diffTemplateDefined is the parsed template object for defined-type Diff generation.
var diffTemplateDefined = template.Must(template.New("DiffTemplateDefined").Parse(diffTemplateDefinedTxt))

// WriteDiffFiles generates Go files containing Diff methods based on the provided
// code generation context (`ctx`). It organizes generated code by output file and package.
//
// Parameters:
//   - dir: Base directory where files will be written
//   - file: Initial target file path (may be overridden based on type and package)
//   - files: Map of file paths to a map of code sections ("Package", "Imports", "Diff")
//   - ctx: Code generation context containing metadata and generated implementations
//
// Behavior:
//   - Skips generation if Diff function name or implementation is empty, or if there was an error.
//   - For struct types or defined types, generates a dedicated Go file with the full Diff function.
//   - For other cases, appends the Diff implementation to an existing entry in the `files` map.
//   - Recursively processes any sub-contexts to handle nested or related types.
func WriteDiffFiles(dir, file string, files map[string]map[string]string, ctx data.Ctx) error {
	// Skip if no Diff function name
	if ctx.DiffFuncName == "" {
		return nil
	}
	// Skip if no Diff implementation
	if ctx.DiffImplementation == "" {
		return nil
	}
	// Skip if an error occurred in context generation
	if ctx.Err {
		return nil
	}

	// Special handling for struct types and defined types
	if ctx.ObjectKind == data.KindToString(data.Struct) || ctx.DefinedType {
		// Build output file path based on package path and type name
		file = filepath.Join(dir, ctx.PkgPath, strings.ToLower(ctx.Type)+"_diff_generated.go")

		// Prepare template arguments
		args := map[string]string{
			"LeftSideComparison":  ctx.LeftSideComparison,
			"RightSideComparison": ctx.RightSideComparison,
			"Type":                ctx.Type,
			"DiffImplementation":  ctx.DiffImplementation,
		}

		// Render the template
		contents := bytes.Buffer{}
		diffTemplate := diffTemplateRaw
		if ctx.DefinedType {
			diffTemplate = diffTemplateDefined
		}
		err := diffTemplate.Execute(&contents, args)
		if err != nil {
			return err
		}

		// Build imports section if needed
		var importsClause string
		if len(ctx.Imports) > 0 {
			imports := bytes.Buffer{}
			for imp := range ctx.Imports {
				imports.WriteString("\"" + imp + "\"\n")
			}
			importsClause = "import (\n" + imports.String() + ")"
		}
		// Store generated code in files map
		files[file] = map[string]string{
			"Package": "package " + ctx.Pkg,
			"Imports": importsClause,
			"Diff":    contents.String(),
		}
		// Recursively process sub-contexts
		for _, subCtx := range ctx.SubCtxs {
			WriteDiffFiles(dir, file, files, *subCtx)
		}
		return nil
	}

	// For non-struct and non-defined types: append to an existing file entry
	implementations := files[file]
	if implementations == nil {
		implementations = map[string]string{}
		files[file] = implementations
	}
	implementations[ctx.DiffFuncName] = ctx.DiffImplementation

	// Recursively process sub-contexts
	for _, subCtx := range ctx.SubCtxs {
		WriteDiffFiles(dir, file, files, *subCtx)
	}
	return nil
}

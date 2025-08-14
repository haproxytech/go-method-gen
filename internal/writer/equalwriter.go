package writer

import (
	"bytes"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/haproxytech/eqdiff/internal/data"
)

const equalTemplateTxt = `func ({{.LeftSideComparison}} {{.Type}}) Equal({{.RightSideComparison}} {{.Type}}) bool {
	return {{.EqualImplementation}}
}
`

// Pre-parse and store the template for generating Equal functions.
// The template expects placeholders for left/right comparison variable names,
// the type name, and the actual comparison implementation.
var equalTemplate = template.Must(template.New("EqualTemplate").Parse(equalTemplateTxt))

// WriteEqualFiles generates Go source code for Equal functions based on the given context
// and writes them into an in-memory map of files.
//
// Parameters:
//   - dir: base directory where files should be placed.
//   - file: the current target filename (may be overridden based on context).
//   - files: an in-memory structure mapping filenames to a map of code sections (Package, Imports, Equal, etc.).
//   - ctx: a data.Ctx object containing type metadata and the Equal implementation.
//
// Behavior:
//   - If the context has no Equal function name or implementation, nothing is written.
//   - If the type is a struct or a defined type, a dedicated "_equal_generated.go" file is created
//     for that type, including package declaration, imports, and the Equal method body.
//   - If the type is not a struct/defined type, the Equal implementation is appended to the existing file entry.
//   - This function is recursive: it processes all sub-contexts in ctx.SubCtxs.
func WriteEqualFiles(dir, file string, files map[string]map[string]string, ctx data.Ctx) error {

	// Skip generation if Equal function name or implementation is missing
	if ctx.EqualFuncName == "" {
		return nil
	}
	if ctx.EqualImplementation == "" {
		return nil
	}
	if ctx.Err {
		return nil
	}

	// Case: Structs or explicitly defined types get their own file
	if ctx.ObjectKind == data.KindToString(data.Struct) || ctx.DefinedType {
		file = filepath.Join(dir, ctx.PkgPath, strings.ToLower(ctx.Type)+"_equal_generated.go")

		// Prepare the template arguments for Equal function generation
		args := map[string]string{
			"LeftSideComparison":  ctx.LeftSideComparison,
			"RightSideComparison": ctx.RightSideComparison,
			"Type":                ctx.Type,
			"EqualImplementation": ctx.EqualImplementation,
		}
		// Render the Equal function template into a buffer
		contents := bytes.Buffer{}
		err := equalTemplate.Execute(&contents, args)
		if err != nil {
			return err
		}

		// Build the import clause if the context has imports
		var importsClause string
		if len(ctx.Imports) > 0 {
			imports := bytes.Buffer{}
			for imp := range ctx.Imports {
				imports.WriteString("\"" + imp + "\"\n")
			}
			importsClause = "import (\n" + imports.String() + ")"
		}

		// Store the generated content in the in-memory file map
		files[file] = map[string]string{
			"Package": "package " + ctx.Pkg,
			"Imports": importsClause,
			"Equal":   contents.String(),
		}

		// Recursively process sub-contexts
		for _, subCtx := range ctx.SubCtxs {
			WriteEqualFiles(dir, file, files, *subCtx)
		}
		return nil
	}

	// Case: Append Equal implementation to an existing file (non-struct, non-defined types)
	implementations := files[file]
	if implementations == nil {
		implementations = map[string]string{}
		files[file] = implementations
	}
	implementations[ctx.EqualFuncName] = ctx.EqualImplementation

	// Recursively process sub-contexts
	for _, subCtx := range ctx.SubCtxs {
		WriteEqualFiles(dir, file, files, *subCtx)
	}
	return nil
}

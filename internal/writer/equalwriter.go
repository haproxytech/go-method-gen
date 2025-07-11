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

var equalTemplate = template.Must(template.New("EqualTemplate").Parse(equalTemplateTxt))

func WriteEqualFiles(dir, file string, files map[string]map[string]string, ctx data.Ctx) error {

	if ctx.EqualFuncName == "" {
		return nil
	}
	if ctx.EqualImplementation == "" {
		return nil
	}
	if ctx.Err {
		return nil
	}

	if ctx.ObjectKind == data.KindToString(data.Struct) || ctx.DefinedType {
		file = filepath.Join(dir, ctx.PkgPath, strings.ToLower(ctx.Type)+"_equal_generated.go")
		pathParts := strings.Split(ctx.PkgPath, "/")
		args := map[string]string{
			"LeftSideComparison":  ctx.LeftSideComparison,
			"RightSideComparison": ctx.RightSideComparison,
			"Type":                ctx.Type,
			"EqualImplementation": ctx.EqualImplementation,
		}
		contents := bytes.Buffer{}
		err := equalTemplate.Execute(&contents, args)
		if err != nil {
			return err
		}
		var importsClause string
		if len(ctx.Imports) > 0 {
			imports := bytes.Buffer{}
			for imp := range ctx.Imports {
				imports.WriteString("\"" + imp + "\"\n")
			}
			importsClause = "import (\n" + imports.String() + ")"
		}
		files[file] = map[string]string{
			"Package": "package " + pathParts[len(pathParts)-1],
			"Imports": importsClause,
			"Equal":   contents.String(),
		}
		for _, subCtx := range ctx.SubCtxs {
			WriteEqualFiles(dir, file, files, *subCtx)
		}
		return nil
	}

	implementations := files[file]
	if implementations == nil {
		implementations = map[string]string{}
		files[file] = implementations
	}
	implementations[ctx.EqualFuncName] = ctx.EqualImplementation

	for _, subCtx := range ctx.SubCtxs {
		WriteEqualFiles(dir, file, files, *subCtx)
	}
	return nil
}

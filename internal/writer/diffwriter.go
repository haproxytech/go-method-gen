package writer

import (
	"bytes"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/haproxytech/eqdiff/internal/data"
)

const diffTemplateRawTxt = `func ({{.LeftSideComparison}} {{.Type}}) Diff({{.RightSideComparison}} {{.Type}}) map[string][]interface{} {
	diff := make(map[string][]interface{})
	{{.DiffImplementation}}
	return diff
}
`

const diffTemplateDefinedTxt = `func ({{.LeftSideComparison}} {{.Type}}) Diff({{.RightSideComparison}} {{.Type}}) map[string][]interface{} {
	return {{.DiffImplementation}}
}
`

var diffTemplateRaw = template.Must(template.New("DiffTemplate").Parse(diffTemplateRawTxt))
var diffTemplateDefined = template.Must(template.New("DiffTemplateDefined").Parse(diffTemplateDefinedTxt))

func WriteDiffFiles(dir, file string, files map[string]map[string]string, ctx data.Ctx) error {
	if ctx.DiffFuncName == "" {
		return nil
	}
	if ctx.DiffImplementation == "" {
		return nil
	}
	if ctx.Err {
		return nil
	}

	if ctx.ObjectKind == data.KindToString(data.Struct) || ctx.DefinedType {
		file = filepath.Join(dir, ctx.PkgPath, strings.ToLower(ctx.Type)+"_diff_generated.go")
		args := map[string]string{
			"LeftSideComparison":  ctx.LeftSideComparison,
			"RightSideComparison": ctx.RightSideComparison,
			"Type":                ctx.Type,
			"DiffImplementation":  ctx.DiffImplementation,
		}
		contents := bytes.Buffer{}
		diffTemplate := diffTemplateRaw
		if ctx.DefinedType {
			diffTemplate = diffTemplateDefined
		}
		err := diffTemplate.Execute(&contents, args)
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
			"Package": "package " + ctx.Pkg,
			"Imports": importsClause,
			"Diff":    contents.String(),
		}
		for _, subCtx := range ctx.SubCtxs {
			WriteDiffFiles(dir, file, files, *subCtx)
		}
		return nil
	}

	implementations := files[file]
	if implementations == nil {
		implementations = map[string]string{}
		files[file] = implementations
	}
	implementations[ctx.DiffFuncName] = ctx.DiffImplementation

	for _, subCtx := range ctx.SubCtxs {
		WriteDiffFiles(dir, file, files, *subCtx)
	}
	return nil
}

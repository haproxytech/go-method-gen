package eqdiff

import (
	"bytes"
	"fmt"
	"go/format"
	"os"
	"path/filepath"
	"reflect"
	"strings"

	"github.com/haproxytech/eqdiff/internal/data"
	"github.com/haproxytech/eqdiff/internal/generators/diff"
	"github.com/haproxytech/eqdiff/internal/generators/equal"
	"github.com/haproxytech/eqdiff/internal/parser"
	"github.com/haproxytech/eqdiff/internal/writer"
	imp "golang.org/x/tools/imports"
)

type Options struct {
	OutputDir string
	Package   string
	Filename  string
	Prefix    string
}

func Generate(types []reflect.Type, opts Options) error {

	roots := []*data.TypeNode{}
	dir := opts.OutputDir

	// First we parse the type to get their description by reflection
	for _, typ := range types {
		root := &data.TypeNode{}
		roots = append(roots, root)
		parser.Parse(root, typ, typ.PkgPath(), map[string]struct{}{})
	}

	funcsByPkg := map[string]map[string]struct{}{}              // pkg -> func
	setEqualsFuncsByBaseDir := map[string]map[string]struct{}{} // baseDir -> Equalxx funcName
	setDiffsFuncsByBaseDir := map[string]map[string]struct{}{}  // baseDir -> Diffxx funcName
	for _, root := range roots {
		ctx := &data.Ctx{LeftSideComparison: "rec", RightSideComparison: "obj"}
		if root.HasEqual {
			continue
		}
		equal.EqualGenerator(root, ctx, map[string]struct{}{})
		if len(ctx.SubCtxs) == 1 {
			contents := map[string]map[string]string{} // file -> func -> implementation
			writer.WriteEqualFiles(dir, "", contents, *ctx.SubCtxs[0])
			for file, funcs := range contents {
				for funName := range funcs {
					if funName == "Equal" || !strings.HasPrefix(funName, "Equal") {
						continue
					}
					basedirContent := filepath.Dir(file)
					equalsfuncs := setEqualsFuncsByBaseDir[basedirContent]
					if equalsfuncs == nil {
						equalsfuncs = map[string]struct{}{}
						setEqualsFuncsByBaseDir[basedirContent] = equalsfuncs

					}
					if _, exists := equalsfuncs[funName]; exists {
						delete(funcs, funName)
					}
					equalsfuncs[funName] = struct{}{}
				}
			}
			for file, funcs := range contents {
				baseDir := filepath.Dir(file)
				pkgfuncs, pkgExists := funcsByPkg[baseDir]
				if !pkgExists {
					pkgfuncs = map[string]struct{}{}
					funcsByPkg[baseDir] = pkgfuncs
				}

				var sb bytes.Buffer
				err := os.MkdirAll(baseDir, 0o755)
				if err != nil {
					return err
				}
				pkg := funcs["Package"]
				sb.WriteString(pkg + "\n")
				delete(funcs, "Package")
				imports := funcs["Imports"]
				sb.WriteString(imports + "\n")
				delete(funcs, "Imports")
				var hasFunc bool
				for _, fun := range funcs {
					if _, funExists := pkgfuncs[fun]; funExists {
						continue
					}
					hasFunc = true
					pkgfuncs[fun] = struct{}{}
					sb.WriteString(fun + "\n")
				}
				if hasFunc {
					formattedCode, errFormat := format.Source(sb.Bytes())
					if errFormat != nil {
						fmt.Println(errFormat.Error())
						os.WriteFile(file, sb.Bytes(), 0o644)
						//return errFormat
					} else {
						os.WriteFile(file, formattedCode, 0o644)
					}

				}
			}
		}
		ctx = &data.Ctx{LeftSideComparison: "rec", RightSideComparison: "obj"}
		if root.HasDiff {
			continue
		}
		diff.DiffGenerator(root, ctx, map[string]struct{}{})
		if len(ctx.SubCtxs) == 1 {
			contents := map[string]map[string]string{} // file -> func -> implementation
			writer.WriteDiffFiles(dir, "", contents, *ctx.SubCtxs[0])
			for file, funcs := range contents {
				for funName := range funcs {
					if funName == "Diff" || !strings.HasPrefix(funName, "Diff") {
						continue
					}
					basedirContent := filepath.Dir(file)
					diffsfuncs := setDiffsFuncsByBaseDir[basedirContent]
					if diffsfuncs == nil {
						diffsfuncs = map[string]struct{}{}
						setDiffsFuncsByBaseDir[basedirContent] = diffsfuncs

					}
					if _, exists := diffsfuncs[funName]; exists {
						delete(funcs, funName)
					}
					diffsfuncs[funName] = struct{}{}
				}
			}
			for file, funcs := range contents {
				baseDir := filepath.Dir(file)
				pkgfuncs, pkgExists := funcsByPkg[baseDir]
				if !pkgExists {
					pkgfuncs = map[string]struct{}{}
					funcsByPkg[baseDir] = pkgfuncs
				}

				var sb bytes.Buffer
				err := os.MkdirAll(baseDir, 0o755)
				if err != nil {
					return err
				}
				pkg := funcs["Package"]
				sb.WriteString(pkg + "\n")
				delete(funcs, "Package")
				imports := funcs["Imports"]
				sb.WriteString(imports + "\n")
				delete(funcs, "Imports")
				var hasFunc bool
				for _, fun := range funcs {
					if _, funExists := pkgfuncs[fun]; funExists {
						continue
					}
					hasFunc = true
					pkgfuncs[fun] = struct{}{}
					sb.WriteString(fun + "\n")
				}
				if hasFunc {
					formattedCode, errFormat := imp.Process("", sb.Bytes(), nil)
					if errFormat != nil {
						fmt.Printf("file: %s, err: %s\n", file, errFormat.Error())
						fmt.Println(string(sb.Bytes()))
						return errFormat
					}
					os.WriteFile(file, formattedCode, 0o644)
				}
			}
		}
	}
	return nil
}

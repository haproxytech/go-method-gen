package main

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"go/types"
	"log"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/haproxytech/eqdiff/internal/utils"
	"golang.org/x/tools/go/packages"
)

const generatedMainTemplate = `
package main

import (
	"fmt"
	"os"
	"reflect"
	{{range $imp, $alias := .Imports}}{{if ne $alias ""}}
	{{$alias}} "{{$imp}}"{{else}}
	"{{$imp}}"{{end}}{{end}}
	"github.com/haproxytech/eqdiff/pkg/eqdiff"
)

func main() {
	{{range $_, $typeSpec := .TypeSpecs}}{{if $typeSpec.IsAliasType}}
	var {{$typeSpec.AliasTypeVar}} {{$typeSpec.PackagedType}}
	{{end}}{{end}}

	types := []reflect.Type{
		{{range $_, $typeSpec := .TypeSpecs}}{{if $typeSpec.IsAliasType}}
		reflect.TypeOf({{$typeSpec.AliasTypeVar}}),
		{{else}}
		reflect.TypeOf({{$typeSpec.PackagedType}}{}),
		{{end}}{{end}}
	}

	err := eqdiff.Generate(types, eqdiff.Options{
		OutputDir: {{printf "%q" .OutputDir}},
		OverridesFile: {{printf "%q" .OverridesPath}},
		HeaderPath: {{printf "%q" .HeaderPath}},
	})
	if err != nil {
		fmt.Println("Generation error:", err)
		os.Exit(1)
	}
}
`

type TemplateData struct {
	Imports       map[string]string
	TypeSpecs     map[string]TypeSpec
	OutputDir     string
	OverridesPath string
	HeaderPath    string
}

type TypeSpec struct {
	FullName     string // github.com/haproxytech/client-native/v5/models.SpoeScope@v5.1.15
	Version      string // v5.1.15
	Package      string // github.com/haproxytech/client-native/v5/models
	PackagedType string // models.SpoeScope
	Type         string // SpoeScope
	ImportName   string // models
	PackageAlias string // models, or alias if invalid package name
	IsAliasType  bool   // true if alias
	AliasTypeVar string
}

func main() {
	outputDir := "./generated"
	var typeArgs []string
	var keepTemp, debug bool
	var replaceEqdiffPath, overridesPath, headerPath string
	var extraReplaces []string
	var seenOutputDir, seenKeepTemp, seenDebug,
		seenHeader, seenReplace, seenOverrides bool
	var scanPath string
	var seenScan bool

	for _, arg := range os.Args[1:] {
		switch {
		case strings.HasPrefix(arg, "--scan="):
			if seenScan {
				exit("Error: --scan specified more than once")
			}
			scanPath = strings.TrimPrefix(arg, "--scan=")
			seenScan = true
		case strings.HasPrefix(arg, "--output-dir="):
			if seenOutputDir {
				exit("Error: --output-dir specified more than once")
			}
			outputDir = strings.TrimPrefix(arg, "--output-dir=")
			if outputDir == "" {
				exit("Error: --output-dir value cannot be empty")
			}
			seenOutputDir = true

		case arg == "--keep-temp":
			if seenKeepTemp {
				exit("Error: --keep-temp specified more than once")
			}
			keepTemp = true
			seenKeepTemp = true

		case arg == "--debug":
			if seenDebug {
				exit("Error: --debug specified more than once")
			}
			debug = true
			seenDebug = true

		case strings.HasPrefix(arg, "--replace-eqdiff="):
			if seenReplace {
				exit("Error: --replace-eqdiff specified more than once")
			}
			replaceEqdiffPath = strings.TrimPrefix(arg, "--replace-eqdiff=")
			seenReplace = true
		case strings.HasPrefix(arg, "--overrides="):
			if seenOverrides {
				exit("Error: --overrides specified more than once")
			}
			overridesPath = strings.TrimPrefix(arg, "--overrides=")
			seenOverrides = true
		case strings.HasPrefix(arg, "--replace="):
			extraReplaces = append(extraReplaces, strings.TrimPrefix(arg, "--replace="))
		case strings.HasPrefix(arg, "--header-file="):
			if seenHeader {
				exit("Error: --header-file specified more than once")
			}
			headerPath = strings.TrimPrefix(arg, "--header-file=")
			seenHeader = true
		case strings.HasPrefix(arg, "--"):
			exit(fmt.Sprintf("Error: unknown option: %s", arg))

		default:
			typeArgs = append(typeArgs, arg)
		}
	}

	if len(typeArgs) > 0 && seenScan {
		exit("Error: you cannot provide types as arguments and use --scan=<path> at the same time")
	}

	if len(typeArgs) == 0 && !seenScan {
		exit("Error: you must provide either types as arguments or use --scan=<path>")
	}

	if debug {
		fmt.Println("▶️ Debug mode ON")
		fmt.Println("\u2022 Parsed args:")
		fmt.Printf("  - outputDir: %s\n", outputDir)
		fmt.Printf("  - keepTemp: %v\n", keepTemp)
		fmt.Printf("  - typeArgs: %v\n", typeArgs)
		fmt.Printf("  - replaceEqdiffPath: %s\n", replaceEqdiffPath)
		fmt.Printf("  - overridesPath: %s\n", overridesPath)
		fmt.Printf("  - extraReplaces: %v\n", extraReplaces)
	}

	var moduleName string
	if seenScan {
		cmd := exec.Command("go", "list", "-m")
		cmd.Dir = scanPath
		out, err := cmd.Output()
		check(err)
		moduleName = strings.TrimSpace(string(out))
	} else {
		modName, err := exec.Command("go", "list", "-m").Output()
		check(err)
		moduleName = strings.TrimSpace(string(modName))
	}

	if debug {
		fmt.Printf("\u2022 Detected Go module: %s\n", moduleName)
	}

	//typeSpecs := map[string]string{}
	typeSpecs := map[string]TypeSpec{}
	//definedTypeVariables := map[string]string{}
	var importsWithVersion []string

	// map import path -> alias (alias = "" means no alias)
	imports := make(map[string]string)
	importSet := make(map[string]bool)

	if seenScan {
		importsScan, specs, modRoot, err := scanTypes(scanPath, moduleName)
		check(err)
		extraReplaces = append(extraReplaces, moduleName+":"+modRoot)
		for _, imp := range importsScan {
			importSet[imp] = true
		}

		importsWithVersion = importsScan
		for _, spec := range specs {
			//typeSpecs[spec] = spec + "{}"
			typeSpecs[spec.FullName] = spec
			if !importSet[spec.Package] {
				importSet[spec.Package] = true
				pkg := spec.Package
				if spec.Version != "" {
					pkg += "@" + spec.Version
				}
				importsWithVersion = append(importsWithVersion, pkg)
				// Alias detection for dash in package base name
				imports[spec.Package] = spec.PackageAlias
			}
		}

		// Fill alias map for scanned imports:
		for _, imp := range importsScan {
			imports[imp] = utils.AliasImport(filepath.Base(imp))
		}

		if debug {
			fmt.Println("• Scanned types:")
			for _, t := range typeSpecs {
				fmt.Printf("  - %s\n", t)
			}
			fmt.Println("• From imports:")
			for _, imp := range importsScan {
				a := imports[imp]
				if a != "" {
					fmt.Printf("  - %s (alias %s)\n", imp, a)
				} else {
					fmt.Printf("  - %s\n", imp)
				}
			}
		}
	}

	for _, full := range typeArgs {
		typeSpec, err := parseTypeSpec(full)
		if err != nil {
			exit(err.Error())
		}
		typeSpecs[full] = typeSpec
		if !importSet[typeSpec.Package] {
			importSet[typeSpec.Package] = true
			pkg := typeSpec.Package
			if typeSpec.Version != "" {
				pkg += "@" + typeSpec.Version
			}
			importsWithVersion = append(importsWithVersion, pkg)
			// Alias detection for dash in package base name
			imports[typeSpec.Package] = typeSpec.PackageAlias
		}
	}

	// Add imports and types from typeArgs:
	// for _, full := range typeArgs {
	// 	typeSpec, err := parseTypeSpec(full)
	// 	if err != nil {
	// 		exit(err.Error())
	// 	}
	// 	var version string
	// 	at := strings.LastIndex(full, "@")
	// 	if at != -1 {
	// 		version = full[at+1:]
	// 		full = full[:at]
	// 	}
	// 	typeSpec.Version = version
	// 	dot := strings.LastIndex(full, ".")
	// 	if dot == -1 || dot == len(full)-1 {
	// 		exit(fmt.Sprintf("Invalid type format: %s (expected importpath.TypeName)", full))
	// 	}
	// 	importPath := full[:dot]
	// 	typeSpec.Package = importPath
	// 	importWithVersion := importPath
	// 	if version != "" {
	// 		importWithVersion += "@" + version
	// 	}
	// 	typeSpec.PackagedType = full[dot+1:]

	// 	if !importSet[importPath] {
	// 		importSet[importPath] = true
	// 		importsWithVersion = append(importsWithVersion, importWithVersion)

	// 		// Alias detection for dash in package base name
	// 		imports[importPath] = utils.AliasImport(filepath.Base(importPath))
	// 	}
	// 	pkgAlias := imports[importPath]
	// 	if pkgAlias == "" {
	// 		pkgAlias = filepath.Base(importPath)
	// 	}
	// 	//typeSpecs[originalFull] = fmt.Sprintf("%s.%s{}", pkgAlias, typeName)
	// 	//typeSpecs = append(typeSpecs,fmt.Sprintf("%s.%s", pkgAlias, typeName))
	// }

	absOutputDir := filepath.Join(cwd(), outputDir)
	clearOutputDir(absOutputDir, debug)

	if debug {
		fmt.Println("• Final import alias map:")
		for imp, alias := range imports {
			if alias != "" {
				fmt.Printf("  %s as %s\n", imp, alias)
			} else {
				fmt.Printf("  %s\n", imp)
			}
		}
		fmt.Println("• Type specs:")
		for t := range typeSpecs {
			fmt.Println("  -", t)
		}
	}

	data := TemplateData{
		Imports:       imports,
		TypeSpecs:     typeSpecs,
		OutputDir:     absOutputDir,
		OverridesPath: overridesPath,
		HeaderPath:    headerPath,
	}

	tmpDir := filepath.Join(cwd(), ".eqdiff-tmp")
	err := os.MkdirAll(tmpDir, 0o755)
	check(err)

	if !keepTemp {
		defer os.RemoveAll(tmpDir)
	} else {
		fmt.Println("Temporary files kept at:", tmpDir)
	}

	generateGoModWithReplaces(tmpDir, replaceEqdiffPath, extraReplaces, debug)
	addGoGetDeps(tmpDir, importsWithVersion, debug)

	// for _, full := range typeArgs {
	// 	isDefinedAlias, err := isDefinedAlias(full)
	// 	if err != nil {
	// 		exit(err.Error())
	// 	}
	// 	if isDefinedAlias {
	// 		delete(data.TypeSpecs, full)
	// 	}
	// 	fmt.Println("isDefinedAlias:", isDefinedAlias)
	// }
	generateMainGo(tmpDir, data, debug)

	cmd := exec.Command("go", "run", ".")
	cmd.Dir = tmpDir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if debug {
		fmt.Printf("\u2022 Executing: go run . (cwd = %s)\n", tmpDir)
	}
	check(cmd.Run())
}

// cwd returns the current working directory or exits on error.
func cwd() string {
	dir, err := os.Getwd()
	check(err)
	return dir
}

// generateGoModWithReplaces creates a temporary go.mod file with optional replace directives.
func generateGoModWithReplaces(tmpDir, replaceEqdiffPath string, extraReplaces []string, debug bool) {
	goModPath := filepath.Join(tmpDir, "go.mod")
	goModContent := "module eqdiff-tmp\n\ngo 1.21\n"
	if replaceEqdiffPath != "" {
		goModContent += fmt.Sprintf("\nreplace github.com/haproxytech/eqdiff => %s\n", replaceEqdiffPath)
	}
	for _, repl := range extraReplaces {
		parts := strings.SplitN(repl, ":", 2)
		if len(parts) != 2 {
			exit(fmt.Sprintf("Invalid replace syntax: %s (expected 'module => path')", repl))
		}
		goModContent += fmt.Sprintf("replace %s => %s\n", strings.TrimSpace(parts[0]), strings.TrimSpace(parts[1]))
	}
	err := os.WriteFile(goModPath, []byte(goModContent), 0644)
	check(err)

	if debug {
		fmt.Println("\u2022 go.mod created")
		if replaceEqdiffPath != "" {
			fmt.Printf("\u2022 Added replace directive for eqdiff => %s\n", replaceEqdiffPath)
		}
		for _, repl := range extraReplaces {
			fmt.Printf("• Added replace directive: %s\n", repl)
		}
	}
}

// addGoGetDeps runs 'go get' on required dependencies and tidies up the module.
// It ensures all necessary packages are downloaded in the temp workspace.
func addGoGetDeps(tmpDir string, importsWithVersion []string, debug bool) {
	for _, pkg := range importsWithVersion {
		cmd := exec.Command("go", "get", pkg)
		cmd.Dir = tmpDir
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if debug {
			fmt.Printf("\u2022 Running: go get %s\n", pkg)
		}
		check(cmd.Run())
	}

	cmd := exec.Command("go", "get", "github.com/haproxytech/eqdiff/pkg/eqdiff")
	cmd.Dir = tmpDir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if debug {
		fmt.Println("\u2022 Running: go get github.com/haproxytech/eqdiff/pkg/eqdiff")
	}
	check(cmd.Run())

	cmd = exec.Command("go", "mod", "tidy")
	cmd.Dir = tmpDir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if debug {
		fmt.Println("\u2022 Running: go mod tidy")
	}
	check(cmd.Run())
}

// generateMainGo creates a main.go file inside the temp directory.
// This file serves as the actual generator which will invoke eqdiff.Generate with provided types.
func generateMainGo(tmpDir string, data TemplateData, debug bool) {
	mainPath := filepath.Join(tmpDir, "main.go")
	file, err := os.Create(mainPath)
	check(err)

	var buf bytes.Buffer
	tmpl := template.Must(template.New("main").Parse(generatedMainTemplate))
	err = tmpl.Execute(&buf, data)
	check(err)

	if debug {
		fmt.Println("• Generated main.go content:")
		fmt.Println(strings.Repeat("-", 60))
		fmt.Println(buf.String())
		fmt.Println(strings.Repeat("-", 60))
	}

	_, err = file.Write(buf.Bytes())
	check(err)
	file.Close()
}

// check exits the program with an error message if err is not nil.
func check(err error) {
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error:", err)
		os.Exit(1)
	}
}

// exit prints an error message and exits with code 1.
func exit(msg string) {
	fmt.Fprintln(os.Stderr, msg)
	os.Exit(1)
}

// clearOutputDir deletes and recreates the given output directory.
func clearOutputDir(path string, debug bool) {
	err := os.RemoveAll(path)
	check(err)
	err = os.MkdirAll(path, 0o755)
	check(err)
	if debug {
		fmt.Printf("• Cleared and recreated output directory: %s\n", path)
	}
}

// scanTypes parses all Go source files in the given directory (scanPath),
// and returns the import paths and a list of discovered type names in that package and the module root.
// It uses the moduleName and the relative path from the module root to compute the full import path.
func scanTypes(scanPath, moduleName string) ([]string, []TypeSpec, string, error) {
	absScanPath, err := filepath.Abs(scanPath)
	if err != nil {
		return nil, nil, "", err
	}

	modRoot, err := findModuleRoot(absScanPath)
	if err != nil {
		return nil, nil, modRoot, fmt.Errorf("could not find module root: %w", err)
	}

	relPath, err := filepath.Rel(modRoot, absScanPath)
	if err != nil {
		return nil, nil, modRoot, fmt.Errorf("cannot determine path relative to module root: %w", err)
	}

	importPath := moduleName
	if relPath != "." {
		importPath = moduleName + "/" + filepath.ToSlash(relPath)
	}

	fset := token.NewFileSet()
	pkgs, err := parser.ParseDir(fset, scanPath, nil, 0)
	if err != nil {
		return nil, nil, modRoot, err
	}

	var typeSpecs []TypeSpec

	for _, pkg := range pkgs {
		for _, file := range pkg.Files {
			packageName := file.Name.Name
			for _, decl := range file.Decls {
				genDecl, ok := decl.(*ast.GenDecl)
				if !ok || genDecl.Tok != token.TYPE {
					continue
				}
				for _, spec := range genDecl.Specs {
					typeSpecNode, ok := spec.(*ast.TypeSpec)
					if !ok {
						continue
					}

					typeName := typeSpecNode.Name.Name
					packagedType := fmt.Sprintf("%s.%s", packageName, typeName)
					fullName := fmt.Sprintf("%s.%s", importPath, typeName)
					dirName := filepath.Base(importPath)
					alias := utils.AliasImport(dirName)

					isAlias, _ := isDefinedAlias(fmt.Sprintf("%s.%s", importPath, typeName))
					var aliasVar string
					if isAlias {
						aliasVar = utils.GenerateAliasVarName(packagedType) // exemple: models.Scope => modelsScope
					}
					typeSpec := TypeSpec{
						FullName:     fullName,
						Version:      "",
						Package:      importPath,
						PackagedType: packagedType,
						Type:         typeName,
						ImportName:   packageName,
						PackageAlias: alias,
						IsAliasType:  isAlias,
						AliasTypeVar: aliasVar,
					}
					typeSpecs = append(typeSpecs, typeSpec)
					log.Printf("typeSpec : %+v", typeSpec)
				}
			}
		}
	}

	return []string{importPath}, typeSpecs, modRoot, nil
}

// findModuleRoot traverses parent directories upwards to locate the nearest go.mod file.
// It returns the path to the module root.
func findModuleRoot(path string) (string, error) {
	path, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}
	for {
		if _, err := os.Stat(filepath.Join(path, "go.mod")); err == nil {
			return path, nil
		}
		parent := filepath.Dir(path)
		if parent == path {
			return "", fmt.Errorf("go.mod not found in any parent of %s", path)
		}
		path = parent
	}
}

func isDefinedAlias(full string) (bool, error) {
	items := strings.SplitN(full, "@", 2)
	packagedType := items[0]
	version := ""
	if len(items) == 2 {
		version = items[1]
	}
	lastDot := strings.LastIndex(packagedType, ".")
	if lastDot == -1 {
		return false, fmt.Errorf("invalid type path: %s", full)
	}
	rawImportPath := packagedType[:lastDot]
	typeName := packagedType[lastDot+1:]

	// Gérer le suffixe @version
	importPath := rawImportPath

	// Déterminer le module à go get (chemin sans le dernier composant)
	modulePath := path.Dir(importPath)
	goGetArg := modulePath
	if version != "" {
		goGetArg += "@" + version
	}

	// Toujours exécuter un go get pour s'assurer que le module est dans go.mod
	cmd := exec.Command("go", "get", goGetArg)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return false, fmt.Errorf("failed to go get module: %w", err)
	}

	// Charger le package
	cfg := &packages.Config{
		Mode: packages.NeedTypes | packages.NeedTypesInfo | packages.NeedImports | packages.NeedName,
	}
	pkgs, err := packages.Load(cfg, importPath)
	if err != nil {
		return false, err
	}
	if packages.PrintErrors(pkgs) > 0 {
		return false, fmt.Errorf("failed to load package %s", importPath)
	}
	if len(pkgs) == 0 {
		return false, fmt.Errorf("no packages found for %s", importPath)
	}
	pkg := pkgs[0]

	obj := pkg.Types.Scope().Lookup(typeName)
	if obj == nil {
		return false, fmt.Errorf("type %s not found in %s", typeName, importPath)
	}
	typeObj, ok := obj.(*types.TypeName)
	if !ok {
		return false, fmt.Errorf("%s is not a named type", typeName)
	}
	named, ok := typeObj.Type().(*types.Named)
	if !ok {
		return false, fmt.Errorf("%s is not a named type", typeName)
	}

	underlying := named.Underlying()
	if _, ok := underlying.(*types.Struct); ok {
		return false, nil // struct complet
	}
	return true, nil // alias (vers un primitif ou autre)
}

func parseTypeSpec(full string) (TypeSpec, error) {
	var version string
	var base string

	parts := strings.SplitN(full, "@", 2)
	base = parts[0]
	if len(parts) == 2 {
		version = parts[1]
	}

	lastDot := strings.LastIndex(base, ".")
	if lastDot == -1 {
		return TypeSpec{}, fmt.Errorf("invalid type path: %s", full)
	}

	pkgPath := base[:lastDot]
	typeName := base[lastDot+1:]
	importName := path.Base(pkgPath)
	alias := utils.AliasPkg(importName)

	isAlias, err := isDefinedAlias(full)
	if err != nil {
		return TypeSpec{}, fmt.Errorf("could not determine if type is defined alias: %w", err)
	}
	var aliasVar string
	if isAlias {
		aliasVar = utils.GenerateAliasVarName(importName + "." + typeName)
	}

	return TypeSpec{
		FullName:     full,
		Version:      version,
		Package:      pkgPath,
		PackagedType: importName + "." + typeName,
		Type:         typeName,
		ImportName:   importName,
		PackageAlias: alias,
		IsAliasType:  isAlias,
		AliasTypeVar: aliasVar,
	}, nil
}

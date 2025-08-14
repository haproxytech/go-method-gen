package main

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"go/types"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/haproxytech/eqdiff/internal/utils"
	"golang.org/x/tools/go/packages"
)

// generatedMainTemplate is the source of the ephemeral generator we write to the
// temp workspace (tmpDir/main.go). It imports all target packages, builds a
// []reflect.Type from the discovered/declared type specs, and runs eqdiff.Generate.
//
// NOTE:
//   - We inject the current working directory ({{.Cwd}}) and explicitly chdir in
//     the generated main so any relative paths provided by the caller work as expected.
//   - For alias types (i.e., non-struct named types), we need a *value* to call
//     reflect.TypeOf on. We therefore declare a zero variable of the alias type
//     and use reflect.TypeOf(var) rather than reflect.TypeOf(T{}).
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
	err := os.Chdir("{{.Cwd}}")
	if err != nil {
		fmt.Println("Failed to change working directory:", err)
		os.Exit(1)
	}
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

	err = eqdiff.Generate(types, eqdiff.Options{
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

// TemplateData is the data model passed to generatedMainTemplate.
type TemplateData struct {
	// Imports maps "import path" -> "alias". An empty alias means no local alias.
	Imports map[string]string
	// TypeSpecs contains all the types that will be fed to eqdiff.Generate.
	TypeSpecs map[string]TypeSpec
	// Paths/flags for the generator.
	OutputDir     string
	OverridesPath string
	HeaderPath    string
	// Cwd is injected into the generated main and used for os.Chdir.
	Cwd string
}

// TypeSpec describes a single target type, including import/package metadata
// and whether the named type is an alias (i.e., not a struct).
type TypeSpec struct {
	FullName     string // e.g., github.com/haproxytech/client-native/v5/models.SpoeScope@v5.1.15
	Version      string // e.g., v5.1.15 (optional @version suffix)
	Package      string // e.g., github.com/haproxytech/client-native/v5/models
	PackagedType string // e.g., models.SpoeScope
	Type         string // e.g., SpoeScope
	ImportName   string // e.g., models (base(import path))
	PackageAlias string // alias used in the generated import if needed (dash, etc.)
	IsAliasType  bool   // true if the type is an alias (i.e., underlying is not a struct)
	AliasTypeVar string // variable name used in the template for alias types
}

// main parses CLI flags, resolves inputs (types or scan), prepares a temp
// workspace (go.mod, go get, generated main), and runs `go run .` to launch
// eqdiff in the ephemeral module.
func main() {
	// Defaults & holders for parsed flags.
	outputDir := "./generated"
	var typeArgs []string
	var keepTemp, debug bool
	var replaceEqdiffPath, overridesPath, headerPath string
	var extraReplaces []string
	var seenOutputDir, seenKeepTemp, seenDebug,
		seenHeader, seenReplace, seenOverrides bool
	var scanPath string
	var seenScan bool
	// --- Argument parsing ---
	// We accept either explicit types as CLI args or a --scan=<path> to discover them.
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
			// Positional args are treated as fully-qualified type identifiers:
			// "<import.path>.Type[@version]"
			typeArgs = append(typeArgs, arg)
		}
	}

	if len(typeArgs) > 0 && seenScan {
		exit("Error: you cannot provide types as arguments and use --scan=<path> at the same time")
	}

	if len(typeArgs) == 0 && !seenScan {
		exit("Error: you must provide either types as arguments or use --scan=<path>")
	}

	// --- Debug dump of parsed args ---
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
	// --- Resolve module context for --scan (or fall back to current module) ---
	var moduleName, absScanPath, modRoot, relPath string
	// if scan option is used we modify working directory  ...
	if seenScan {
		cmd := exec.Command("go", "list", "-m")
		cmd.Dir = scanPath
		out, err := cmd.Output()
		check(err)
		moduleName = strings.TrimSpace(string(out))
		// Compute the relative import path within the module root.
		absScanPath, err = filepath.Abs(scanPath)
		check(err)
		modRoot, err = findModuleRoot(absScanPath)
		check(err)
		relPath, err = filepath.Rel(modRoot, absScanPath)
		check(err)
		// Add a replace so the ephemeral module can import the local scanning module.
		extraReplaces = append(extraReplaces, moduleName+":"+modRoot)
	} else {
		// ... otherwise we use the current working directory.
		modName, err := exec.Command("go", "list", "-m").Output()
		check(err)
		moduleName = strings.TrimSpace(string(modName))
	}

	if debug {
		fmt.Printf("\u2022 Detected Go module: %s\n", moduleName)
	}
	// Accumulators for types/imports we must add to the temporary module.
	typeSpecs := map[string]TypeSpec{}
	var importsWithVersion []string

	// imports: map(import path -> alias). alias="" means "no alias in import".
	imports := make(map[string]string)
	// importSet helps avoid duplicating the same import path.
	importSet := make(map[string]bool)

	// Prepare output directory now; generator will write files there.
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

	// --- Create and (optionally) keep the temp workspace ---
	tmpDir := filepath.Join(cwd(), ".eqdiff-tmp")
	err := os.MkdirAll(tmpDir, 0o755)
	check(err)

	if !keepTemp {
		defer os.RemoveAll(tmpDir)
	} else {
		fmt.Println("Temporary files kept at:", tmpDir)
	}

	// --- Write go.mod with replaces (eqdiff + user-provided replaces) ---
	generateGoModWithReplaces(tmpDir, replaceEqdiffPath, extraReplaces, debug)
	// We'll reuse these across lookups to avoid repeated 'go get' and package loads.
	modulesGoGet := make(map[string]struct{})
	pkgsInfo := map[string][]*packages.Package{}

	// --- Resolve types from explicit CLI arguments ---
	for _, full := range typeArgs {
		typeSpec, err := parseTypeSpec(full, modulesGoGet, pkgsInfo)
		if err != nil {
			exit(err.Error())
		}
		typeSpecs[full] = typeSpec
		// Record the import path and an alias if needed; track a possibly pinned version.
		if !importSet[typeSpec.Package] {
			importSet[typeSpec.Package] = true
			pkg := typeSpec.Package
			if typeSpec.Version != "" {
				pkg += "@" + typeSpec.Version
			}
			importsWithVersion = append(importsWithVersion, pkg)
			// Heuristic aliasing (e.g., if package base name contains '-').
			imports[typeSpec.Package] = typeSpec.PackageAlias
		}
	}
	// --- If --scan was requested, discover types in the package and add them ---
	if seenScan {
		importsScan, specs, err := scanTypes(scanPath, moduleName, relPath)
		check(err)
		// Track all discovered imports and reuse them for go get.
		for _, imp := range importsScan {
			importSet[imp] = true
		}

		importsWithVersion = importsScan
		// Add discovered types and imports (ensuring aliases are set).
		for _, spec := range specs {
			typeSpecs[spec.FullName] = spec
			if !importSet[spec.Package] {
				importSet[spec.Package] = true
				pkg := spec.Package
				if spec.Version != "" {
					pkg += "@" + spec.Version
				}
				importsWithVersion = append(importsWithVersion, pkg)
				imports[spec.Package] = spec.PackageAlias
			}
		}

		// For all scanned imports, ensure alias map is filled even if not already set.
		for _, imp := range importsScan {
			imports[imp] = utils.AliasImport(filepath.Base(imp))
		}

		if debug {
			fmt.Println("• Scanned types:")
			for _, t := range typeSpecs {
				fmt.Printf("  - %+v\n", t)
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
	// --- Render the generated main.go into tmpDir ---
	data := TemplateData{
		Imports:       imports,
		TypeSpecs:     typeSpecs,
		OutputDir:     absOutputDir,
		OverridesPath: overridesPath,
		HeaderPath:    headerPath,
		Cwd:           cwd(),
	}
	generateMainGo(tmpDir, data, debug)
	// --- Fetch deps into the temp module and tidy ---
	addGoGetDeps(tmpDir, importsWithVersion, debug)

	// --- Run the ephemeral generator (go run . in tmpDir) ---
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
// Kept as a tiny helper to reduce error handling noise at callsites.
func cwd() string {
	dir, err := os.Getwd()
	check(err)
	return dir
}

// generateGoModWithReplaces creates a temporary go.mod file inside tmpDir.
// It always writes a new module named "eqdiff-tmp", sets a recent Go version,
// and applies replace directives:
//
//   - If replaceEqdiffPath is provided, eqdiff is replaced to that local path.
//   - Every item in extraReplaces must be "module:path" and is translated into
//     "replace module => path".
func generateGoModWithReplaces(tmpDir, replaceEqdiffPath string, extraReplaces []string, debug bool) {
	goModPath := filepath.Join(tmpDir, "go.mod")
	goModContent := "module eqdiff-tmp\n\ngo 1.21\n"
	if replaceEqdiffPath != "" {
		goModContent += fmt.Sprintf("\nreplace github.com/haproxytech/eqdiff => %s\n", replaceEqdiffPath)
	}
	for _, repl := range extraReplaces {
		// Accept "module:path" (we normalize to a replace directive).
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

// addGoGetDeps runs 'go get' for all imports we collected (possibly with versions),
// ensures eqdiff runtime dependency is present, and finally runs 'go mod tidy'.
// This warms the temp module with everything the generated main.go will need.
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

// generateMainGo materializes the generatedMainTemplate into tmpDir/main.go.
// The rendered program is the small wrapper that calls eqdiff.Generate.
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

// check aborts the program if err is non-nil, printing the error to stderr.
// Centralizing this avoids repeating the same boilerplate all over the file.
func check(err error) {
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error:", err)
		os.Exit(1)
	}
}

// exit prints a message to stderr and exits with status code 1.
// Use this for deliberate user-facing argument/validation errors.
func exit(msg string) {
	fmt.Fprintln(os.Stderr, msg)
	os.Exit(1)
}

// clearOutputDir forcefully recreates the output directory (removing any
// previous content). This is intentional to guarantee a clean generation run.
func clearOutputDir(path string, debug bool) {
	err := os.RemoveAll(path)
	check(err)
	err = os.MkdirAll(path, 0o755)
	check(err)
	if debug {
		fmt.Printf("• Cleared and recreated output directory: %s\n", path)
	}
}

// scanTypes walks all Go files inside scanPath, gathers declared type names,
// prunes any that are only used as dependencies by other types (so we emit
// only "top-level" exported types), and returns both the import path of the
// package and the list of TypeSpec entries to generate.
//
// The import path is computed as "<moduleName>/<relPath>" where relPath is the
// path from the module root to scanPath, normalized to forward slashes.
func scanTypes(scanPath, moduleName, relPath string) ([]string, []TypeSpec, error) {
	importPath := moduleName
	if relPath != "." {
		importPath = moduleName + "/" + filepath.ToSlash(relPath)
	}

	fset := token.NewFileSet()
	pkgs, err := parser.ParseDir(fset, scanPath, nil, parser.AllErrors)
	if err != nil {
		return nil, nil, err
	}

	allTypes := make(map[string]*ast.TypeSpec)
	dependencies := make(map[string]map[string]bool)

	// Collect declared types and their (shallow) dependencies.
	// We only track dependency *names* here, which is enough to drop "leaf" deps later.
	for _, pkg := range pkgs {
		for _, file := range pkg.Files {
			for _, decl := range file.Decls {
				genDecl, ok := decl.(*ast.GenDecl)
				if !ok || genDecl.Tok != token.TYPE {
					continue
				}
				for _, spec := range genDecl.Specs {
					typeSpec, ok := spec.(*ast.TypeSpec)
					if !ok {
						continue
					}
					typeName := typeSpec.Name.Name
					allTypes[typeName] = typeSpec

					dependencies[typeName] = make(map[string]bool)
					findDepsInExpr(typeSpec.Type, dependencies[typeName])
				}
			}
		}
	}

	// Compute which types are referenced by other types; those are considered
	// internal dependencies and are filtered out from the final list.
	used := make(map[string]bool)
	for _, deps := range dependencies {
		for dep := range deps {
			used[dep] = true
		}
	}

	var typeSpecs []TypeSpec
	alias := filepath.Base(importPath)
	// Prefer a safe alias for import (e.g., to handle names with dashes).
	pkgAlias := utils.AliasImport(alias)
	if pkgAlias == "" {
		pkgAlias = alias
	}

	modules := make(map[string]struct{})
	pkgsInfo := map[string][]*packages.Package{}
	for typeName := range allTypes {
		if used[typeName] {
			// Skip types that are only referenced by others (dependencies).
			continue
		}

		fullName := fmt.Sprintf("%s.%s", importPath, typeName)
		// Determine if the named type is an alias (non-struct) or a proper struct.
		isAlias, _ := isDefinedAlias(fullName, modules, pkgsInfo)
		packaged := fmt.Sprintf("%s.%s", pkgAlias, typeName)
		varName := utils.GenerateAliasVarName(packaged)

		typeSpecs = append(typeSpecs, TypeSpec{
			FullName:     fullName,
			Version:      "",
			Package:      importPath,
			PackagedType: packaged,
			Type:         typeName,
			ImportName:   alias,
			PackageAlias: pkgAlias,
			IsAliasType:  isAlias,
			AliasTypeVar: varName,
		})
	}

	return []string{importPath}, typeSpecs, nil
}

// findModuleRoot climbs up from 'path' until it finds a directory containing a
// go.mod file and returns that directory path. It fails if no go.mod is found.
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

// isDefinedAlias determines whether a fully qualified type (e.g.,
// "github.com/foo/bar/pkg.Type[@v1.2.3]") is an alias type (i.e., not a struct).
//
// Implementation details:
//   - We optionally 'go get' the module path (dirname of import path) at a pinned
//     version to ensure it is present in the temp module.
//   - We load the package with x/tools/go/packages to access type info.
//   - If the named type's underlying type is *types.Struct → NOT an alias.
//     Otherwise → alias (e.g., alias to a primitive or another non-struct type).
func isDefinedAlias(full string, modulesGoGet map[string]struct{}, pkgsInfo map[string][]*packages.Package) (bool, error) {
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

	// 'importPath' is the package path that contains 'typeName'.
	importPath := rawImportPath

	// Determine the module path to 'go get' by trimming the last path element.
	modulePath := path.Dir(importPath)
	goGetArg := modulePath
	if version != "" {
		goGetArg += "@" + version
	}
	// Avoid running 'go get' multiple times for the same module@version.
	if _, alreadyGot := modulesGoGet[goGetArg]; !alreadyGot {
		cmd := exec.Command("go", "get", goGetArg)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			return false, fmt.Errorf("failed to go get module: %w", err)
		}
		modulesGoGet[goGetArg] = struct{}{}
	}
	// Load the package with full type info.
	cfg := &packages.Config{
		Mode: packages.NeedTypes | packages.NeedTypesInfo | packages.NeedImports | packages.NeedName,
	}
	var pkgs []*packages.Package
	if pkgsInfo[importPath] != nil {
		pkgs = pkgsInfo[importPath]
	} else {
		var err error
		pkgs, err = packages.Load(cfg, importPath)
		if err != nil {
			return false, err
		}
		pkgsInfo[importPath] = pkgs
	}
	if packages.PrintErrors(pkgs) > 0 {
		return false, fmt.Errorf("failed to load package %s", importPath)
	}
	if len(pkgs) == 0 {
		return false, fmt.Errorf("no packages found for %s", importPath)
	}
	pkg := pkgs[0]
	// Lookup the type symbol by name and inspect its underlying type.
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
	return true, nil // Alias (e.g., alias to primitive or other non-struct type).
}

// parseTypeSpec parses a fully qualified type spec of the form
// "<import.path>.Type[@version]" and returns a TypeSpec containing import
// information, alias detection, and optional generated variable name for alias types.
func parseTypeSpec(full string, modulesGoGet map[string]struct{}, pkgsInfo map[string][]*packages.Package) (TypeSpec, error) {
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

	// Determine alias-ness; we ignore the error here because we still want a
	// partially filled spec even if loading fails (caller will handle errors).
	isAlias, _ := isDefinedAlias(full, modulesGoGet, pkgsInfo)
	var aliasVar string
	if isAlias {
		// Variable used in the template to get reflect.TypeOf(aliasVar).
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

// findDepsInExpr walks a type expression and records referenced identifiers'
// names into 'deps'. This is a shallow name-based dependency tracking used by
// scanTypes to drop types that are only used by other types.
func findDepsInExpr(expr ast.Expr, deps map[string]bool) {
	switch t := expr.(type) {
	case *ast.Ident:
		deps[t.Name] = true

	case *ast.SelectorExpr:
		deps[t.Sel.Name] = true

	case *ast.StarExpr:
		findDepsInExpr(t.X, deps)

	case *ast.ArrayType:
		findDepsInExpr(t.Elt, deps)

	case *ast.MapType:
		findDepsInExpr(t.Key, deps)
		findDepsInExpr(t.Value, deps)

	case *ast.StructType:
		for _, field := range t.Fields.List {
			findDepsInExpr(field.Type, deps)
		}
	}
}

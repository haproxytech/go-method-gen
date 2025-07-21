package main

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/haproxytech/eqdiff/internal/common"
	yaml "gopkg.in/yaml.v3"
)

const generatedMainTemplate = `
package main

import (
	"fmt"
	"os"
	"reflect"
	{{range .Imports}}"{{.}}"
	{{end}}
	"github.com/haproxytech/eqdiff/pkg/eqdiff"
)

func main() {
	types := []reflect.Type{
		{{range .TypeSpecs}}reflect.TypeOf({{.}}{}),
		{{end}}
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
	Imports       []string
	TypeSpecs     []string
	OutputDir     string
	OverridesPath string
	HeaderPath    string
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

	var typeSpecs []string
	var importsWithVersion []string

	var imports []string
	importSet := make(map[string]bool)

	if seenScan {
		importsScan, specs, err := scanTypes(scanPath, moduleName)
		imports = importsScan
		check(err)
		for i := range imports {
			importSet[imports[i]] = true
		}
		typeSpecs = append([]string{}, specs...)

		importsWithVersion = imports

		if debug {
			fmt.Println("• Scanned types:")
			for _, t := range typeSpecs {
				fmt.Printf("  - %s\n", t)
			}
			fmt.Println("• From imports:")
			for _, imp := range imports {
				fmt.Printf("  - %s\n", imp)
			}
		}
	}

	for _, full := range typeArgs {
		var version string
		at := strings.LastIndex(full, "@")
		if at != -1 {
			version = full[at+1:]
			full = full[:at]
		}

		dot := strings.LastIndex(full, ".")
		if dot == -1 || dot == len(full)-1 {
			exit(fmt.Sprintf("Invalid type format: %s (expected importpath.TypeName)", full))
		}
		importPath := full[:dot]
		importWithVersion := importPath
		if version != "" {
			importWithVersion += "@" + version
		}
		typeName := full[dot+1:]

		if !importSet[importPath] {
			imports = append(imports, importPath)
			importsWithVersion = append(importsWithVersion, importWithVersion)
			importSet[importPath] = true
		}
		pkgAlias := filepath.Base(importPath)
		typeSpecs = append(typeSpecs, fmt.Sprintf("%s.%s", pkgAlias, typeName))
	}

	absOutputDir := filepath.Join(cwd(), outputDir)
	clearOutputDir(absOutputDir, debug)
	if debug {
		fmt.Println("• Final import paths:")
		for _, imp := range imports {
			fmt.Println("  -", imp)
		}
		fmt.Println("• Type specs:")
		for _, t := range typeSpecs {
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

	generateMainGo(tmpDir, data, debug)

	generateGoModWithReplaces(tmpDir, replaceEqdiffPath, extraReplaces, debug)
	addGoGetDeps(tmpDir, importsWithVersion, debug)

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
// Used to redirect module paths for local development or testing.
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

// GetTypeAndPackage splits a fully qualified type string into import path and type name.
// e.g., "github.com/foo/bar.TypeName" → ("github.com/foo/bar", "TypeName")
func GetTypeAndPackage(packagedType string) (string, string) {
	dot := strings.LastIndex(packagedType, ".")
	if dot == -1 || dot == len(packagedType)-1 {
		exit(fmt.Sprintf("Invalid type format: %s (expected importpath.TypeName)", packagedType))
	}
	return packagedType[:dot], packagedType[dot+1:]
}

// LoadOverridesYaml reads and parses a YAML file that defines override functions.
func LoadOverridesYaml(path string) (map[string]common.OverrideFuncs, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read overrides file: %w", err)
	}
	var parsed map[string]common.OverrideFuncs
	if err := yaml.Unmarshal(data, &parsed); err != nil {
		return nil, fmt.Errorf("failed to parse overrides YAML: %w", err)
	}
	return parsed, nil
}

// scanTypes parses all Go source files in the given directory (scanPath),
// and returns the import path and a list of discovered type names in that package.
// It uses the moduleName and the relative path from the module root to compute the full import path.
func scanTypes(scanPath, moduleName string) ([]string, []string, error) {
	absScanPath, err := filepath.Abs(scanPath)
	if err != nil {
		return nil, nil, err
	}

	modRoot, err := findModuleRoot(absScanPath)
	if err != nil {
		return nil, nil, fmt.Errorf("could not find module root: %w", err)
	}

	relPath, err := filepath.Rel(modRoot, absScanPath)
	if err != nil {
		return nil, nil, fmt.Errorf("cannot determine path relative to module root: %w", err)
	}

	// Assemble the full import path for the scanned directory
	importPath := moduleName
	if relPath != "." {
		importPath = moduleName + "/" + filepath.ToSlash(relPath)
	}

	fset := token.NewFileSet()
	pkgs, err := parser.ParseDir(fset, scanPath, nil, 0)
	if err != nil {
		return nil, nil, err
	}

	var typeSpecs []string
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
					typeSpecs = append(typeSpecs, fmt.Sprintf("%s.%s", filepath.Base(importPath), typeSpec.Name.Name))
				}
			}
		}
	}

	return []string{importPath}, typeSpecs, nil
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

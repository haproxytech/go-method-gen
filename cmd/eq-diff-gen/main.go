package main

import (
	"bytes"
	"fmt"
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
		Overrides: {{printf "%q" .OverridesPath}},
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
}

func main() {
	outputDir := "./generated"
	var typeArgs []string
	var keepTemp, debug bool
	var replaceEqdiffPath, overridesPath string
	var seenOutputDir, seenKeepTemp, seenDebug, seenReplace, seenOverrides bool

	for _, arg := range os.Args[1:] {
		switch {
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
		case strings.HasPrefix(arg, "--"):
			exit(fmt.Sprintf("Error: unknown option: %s", arg))

		default:
			typeArgs = append(typeArgs, arg)
		}
	}

	if len(typeArgs) == 0 {
		exit("Error: at least one importpath.Type must be provided")
	}

	if debug {
		fmt.Println("▶️ Debug mode ON")
		fmt.Println("\u2022 Parsed args:")
		fmt.Printf("  - outputDir: %s\n", outputDir)
		fmt.Printf("  - keepTemp: %v\n", keepTemp)
		fmt.Printf("  - typeArgs: %v\n", typeArgs)
		fmt.Printf("  - replaceEqdiffPath: %s\n", replaceEqdiffPath)
		fmt.Printf("  - overridesPath: %s\n", overridesPath)
	}

	modName, err := exec.Command("go", "list", "-m").Output()
	check(err)
	moduleName := strings.TrimSpace(string(modName))
	if debug {
		fmt.Printf("\u2022 Detected Go module: %s\n", moduleName)
	}

	var imports []string
	var typeSpecs []string
	importSet := make(map[string]bool)

	for _, full := range typeArgs {
		dot := strings.LastIndex(full, ".")
		if dot == -1 || dot == len(full)-1 {
			exit(fmt.Sprintf("Invalid type format: %s (expected importpath.TypeName)", full))
		}
		importPath := full[:dot]
		typeName := full[dot+1:]

		if !importSet[importPath] {
			imports = append(imports, importPath)
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
	}

	tmpDir := filepath.Join(cwd(), ".eqdiff-tmp")
	err = os.MkdirAll(tmpDir, 0o755)
	check(err)

	if !keepTemp {
		defer os.RemoveAll(tmpDir)
	} else {
		fmt.Println("Temporary files kept at:", tmpDir)
	}

	generateMainGo(tmpDir, data, debug)

	generateGoMod(tmpDir, replaceEqdiffPath, debug)
	addGoGetDeps(tmpDir, imports, debug)

	cmd := exec.Command("go", "run", ".")
	cmd.Dir = tmpDir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if debug {
		fmt.Printf("\u2022 Executing: go run . (cwd = %s)\n", tmpDir)
	}
	check(cmd.Run())
}

func cwd() string {
	dir, err := os.Getwd()
	check(err)
	return dir
}

func generateGoMod(tmpDir, replaceEqdiffPath string, debug bool) {
	goModPath := filepath.Join(tmpDir, "go.mod")
	goModContent := "module eqdiff-tmp\n\ngo 1.21\n"
	if replaceEqdiffPath != "" {
		goModContent += fmt.Sprintf("\nreplace github.com/haproxytech/eqdiff => %s\n", replaceEqdiffPath)
	}
	err := os.WriteFile(goModPath, []byte(goModContent), 0644)
	check(err)

	if debug {
		fmt.Println("\u2022 go.mod created")
		if replaceEqdiffPath != "" {
			fmt.Printf("\u2022 Added replace directive for eqdiff => %s\n", replaceEqdiffPath)
		}
	}
}

func addGoGetDeps(tmpDir string, imports []string, debug bool) {
	for _, pkg := range imports {
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

func check(err error) {
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error:", err)
		os.Exit(1)
	}
}

func exit(msg string) {
	fmt.Fprintln(os.Stderr, msg)
	os.Exit(1)
}

func clearOutputDir(path string, debug bool) {
	err := os.RemoveAll(path)
	check(err)
	err = os.MkdirAll(path, 0o755)
	check(err)
	if debug {
		fmt.Printf("• Cleared and recreated output directory: %s\n", path)
	}
}

func contains(s []string, str string) bool {
	for _, v := range s {
		if v == str {
			return true
		}
	}
	return false
}

func GetTypeAndPackage(packagedType string) (string, string) {
	dot := strings.LastIndex(packagedType, ".")
	if dot == -1 || dot == len(packagedType)-1 {
		exit(fmt.Sprintf("Invalid type format: %s (expected importpath.TypeName)", packagedType))
	}
	return packagedType[:dot], packagedType[dot+1:]
}

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

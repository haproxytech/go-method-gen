package main

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"text/template"
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
		OutputDir: "{{.OutputDir}}",
	})
	if err != nil {
		fmt.Println("Generation error:", err)
		os.Exit(1)
	}
}
`

type TemplateData struct {
	Imports   []string
	TypeSpecs []string
	OutputDir string
}

func main() {
	outputDir := "./generated"
	var typeArgs []string
	var keepTemp, debug bool
	var seenOutputDir, seenKeepTemp, seenDebug bool

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
		fmt.Println("• Parsed args:")
		fmt.Printf("  - outputDir: %s\n", outputDir)
		fmt.Printf("  - keepTemp: %v\n", keepTemp)
		fmt.Printf("  - typeArgs: %v\n", typeArgs)
	}

	// Get current module name
	modName, err := exec.Command("go", "list", "-m").Output()
	check(err)
	moduleName := strings.TrimSpace(string(modName))
	if debug {
		fmt.Printf("• Detected Go module: %s\n", moduleName)
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
		Imports:   imports,
		TypeSpecs: typeSpecs,
		OutputDir: outputDir,
	}

	cwd, err := os.Getwd()
	check(err)

	tmpDir := filepath.Join(cwd, ".eqdiff-tmp")
	err = os.MkdirAll(tmpDir, 0o755)
	check(err)

	if !keepTemp {
		defer os.RemoveAll(tmpDir)
	} else {
		fmt.Println("Temporary files kept at:", tmpDir)
	}

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

	// Run the generated Go code
	cmd := exec.Command("go", "run", "./.eqdiff-tmp")
	cmd.Dir = cwd
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if debug {
		fmt.Printf("• Executing: go run ./.eqdiff-tmp (cwd = %s)\n", cwd)
	}
	check(cmd.Run())
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

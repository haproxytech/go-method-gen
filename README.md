# eq-diff-gen – Struct Comparison and Merge Code Generator for Go

`eq-diff-gen` is a development utility that generates `Equal`, `Diff`, and `Merge` functions for Go structs using reflection. It is designed to reduce boilerplate and promote consistent comparison and merging behavior across complex structures.

This tool is useful for applications involving configuration merging, object synchronization, and state comparison.

---

## Features

* **Generate equality functions**: Check deep equality between two struct instances.
* **Generate diff functions**: Return field-level differences between structs.
* **Custom field overrides**: Provide fine-grained diff/equality behavior via YAML override files.
* **Header injection**: Add license or documentation header to generated code.
* **Module path replacement**: Use local module paths for `eqdiff` or any dependency.
* **CLI compatible**: Usable as a standalone binary or as a scriptable tool in CI/CD.
* **Directory scanning**: Automatically scan a directory for Go types with --scan.

---

## Example Generated Functions

### Equal

The generated `Equal` function checks for deep equality across fields:

```go
func (rec StructA) Equal(obj StructA) bool {
	return EqualMapStringString(rec.maps, obj.maps) &&
		EqualMapIntPointerA(rec.mapA, obj.mapA)
}
```

### Diff

The generated `Diff` function returns a list of changed fields:

```go
func (rec StructA) Diff(obj StructA) map[string][]interface{} {
	diff := make(map[string][]interface{})
	for diffKey, diffValue := range DiffMapStringString(rec.maps, obj.maps) {
		diff["maps"+diffKey] = diffValue
	}
	for diffKey, diffValue := range DiffMapIntPointerA(rec.mapA, obj.mapA) {
		diff["mapA"+diffKey] = diffValue
	}
	return diff
}
```

---

## Installation

```bash
go install github.com/<TBD>/eq-diff-gen/cmd/eq-diff-gen/@latest
```

Or build from source:

```bash
git clone https://github.com/<TBD>/eq-diff-gen
cd eq-diff-gen
go build -o eq-diff-gen
```

---

## Usage

```bash
eq-diff-gen [flags] importpath.TypeName [...]
```

You must provide fully-qualified type paths (importpath.TypeName), unless using --scan.

Alternatively, use the --scan flag to automatically discover all types in a directory:
```bash
eq-diff-gen --scan=./path/to/pkg [flags]
```
⚠️ --scan is exclusive with explicit type arguments. You must use one or the other, not both.

Options:
|option|functionality|
|--|--|
--output-dir=DIR|Path to write generated code (default: ./generated)  |
--keep-temp	|Keep temporary working files (.eqdiff-tmp) for inspection  |
--debug	|Enable verbose output (shows parsed args, generated code, etc.)  |
--replace-eqdiff=DIR|	Use a local path for the eqdiff module  |
--replace=MOD:LOCALPATH|	Add a replace directive in the generated go.mod (can be used multiple times)  |
--overrides=FILE.yaml|YAML file to override diff/equal logic for specific fields  |
--header-file=PATH|Optional Go file to prepend as header in generated output  |
--scan=DIR|	Scan a directory to extract all types (exclusive with type arguments) |

You must provide fully-qualified type paths (`importpath.TypeName`) if not using scan option.

---
## Type Argument Format

Each type must be specified using the following format:

`importpath.TypeName`

You may also include an optional version suffix using Go module semantics:

`importpath.TypeName@version`

### Examples of valid inputs

|Input	|Description|
|-------|-------------|
|github.com/example/project/config.StructConfig|Uses the current module version available in your environment|
|github.com/example/project/config.StructConfig@v1.4.2|Forces resolution of the module at v1.4.2|
|my/local/module/pkg.MyType|Local import path (must be resolvable via go get or replace)|
|my/local/module/pkg.MyType@latest|Gets the latest version from the module proxy|

💡 Note: The version suffix applies only to the module part of the import path.
The tool will extract the type name from the final segment after the last dot (.) and use Go reflection to generate equality, diff code.

---

## Example Use Case

```bash
eq-diff-gen --keep-temp --debug --replace-eqdiff=/home/user/dev/eqdiff github.com/example/project/config.StructConfig
```

This generates methods like `Equal`, `Diff`, and `Merge` for `StructConfig`, stores intermediate files, outputs debug logs, and uses a local path for the `eqdiff` module.

---
## Custom Function Overrides (via --overrides)

You can override the default code generation for specific types by providing a YAML file via the --overrides=FILE.yaml option.

This is useful when:

* You want to reuse hand-written comparison or diff logic,

* The type is not safe or meaningful to reflect on (e.g. generics, external types),

* You need optimized or domain-specific equality logic.

## YAML Structure

Each key in the YAML map must be a fully-qualified type path (importpath.TypeName), and must define either or both of:

    equal: Custom equality function

    diff: Custom diff function

Each function override must provide:

    pkg: the import path of the package containing the function

    name: the name of the function

### Example overrides.yaml
```
github.com/myorg/myproject/pkg/structs.StructA:
  equal:
    pkg: "github.com/myorg/myproject/pkg/structs/funcs"
    name: "EqualStructA"
  diff:
    pkg: "github.com/myorg/myproject/pkg/structs/funcs"
    name: "DiffStructA"

github.com/myorg/data/v5/models.Acls:
  equal:
    pkg: "github.com/myorg/myproject/pkg/structs/funcs"
    name: "EqualAcls"
  diff:
    pkg: "github.com/myorg/myproject/pkg/structs/funcs"
    name: "DiffAcls"
```
This example tells eq-diff-gen to use the custom functions EqualStructA and DiffStructA for StructA, and EqualAcls and DiffAcls for models.Acls.

These functions must have the correct signature, typically:

`func EqualStructA(a, b StructA) bool`  
`func DiffStructA(a, b StructA) map[string][]interface{}`

💡 The specified packages will automatically be imported in the generated file, and the functions will be used instead of auto-generated ones.
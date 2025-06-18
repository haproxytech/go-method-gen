# eq-diff-gen – Struct Comparison and Merge Code Generator for Go

`eq-diff-gen` is a development utility that generates `Equal`, `Diff`, and `Merge` functions for Go structs using reflection. It is designed to reduce boilerplate and promote consistent comparison and merging behavior across complex structures.

This tool is useful for applications involving configuration merging, object synchronization, and state comparison.

---

## Features

* **Generate equality functions**: Check deep equality between two struct instances.
* **Generate diff functions**: Return field-level differences between structs.
* **Generate merge functions**: Support both shallow and deep in-place merging of fields.
* **Custom merge logic**: Automatically delegates merge behavior to `Merge` methods if defined <TODO>.
* **CLI compatible**: Can be used as a standalone binary in development workflows.

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
go install github.com/youruser/eq-diff-gen@latest
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

Options:

* `--keep-temp` – Keep intermediate files like the generated `main.go`
* `--output-dir DIR` – Specify where to write the generated code
* `--debug` – Enable verbose debug logging for troubleshooting or inspection

You must provide fully-qualified type paths (`importpath.TypeName`).

---

## Example Use Case

```bash
eq-diff-gen --keep-temp --debug github.com/example/project/config.StructConfig
```

This generates methods like `Equal`, `Diff`, and `Merge` for `StructConfig`, stores intermediate files, and outputs debug logs.

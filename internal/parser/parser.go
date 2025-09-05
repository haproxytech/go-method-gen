package parser

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/haproxytech/go-method-gen/internal/data"
	"github.com/haproxytech/go-method-gen/internal/utils"
)

// typesToSkip lists fully-qualified type names that should not be parsed.
// These are usually Kubernetes meta types automatically embedded in resources
// and not relevant for equality/diff generation.
var typesToSkip = map[string]struct{}{
	"v1.TypeMeta":   {},
	"v1.ObjectMeta": {},
	"v1.ListMeta":   {},
}

// TypeAlreadyVisited checks if a type has already been processed in the current parsing context.
// It prevents infinite recursion when parsing self-referential or cyclic types.
//
// typ: the reflect.Type to check
// fqnTypesProcessed: a map storing fully-qualified names (package + type) of already parsed types
// returns true if the type was already visited, false otherwise
func TypeAlreadyVisited(typ reflect.Type, fqnTypesProcessed map[string]struct{}) bool {
	if typ.Name() == "" {
		return false
	}
	// Build the fully-qualified name
	fqnType := typ.PkgPath() + "." + typ.Name()

	// If found, it has already been visited
	if _, found := fqnTypesProcessed[fqnType]; found {
		return true
	}

	// Otherwise, mark it as visited
	fqnTypesProcessed[fqnType] = struct{}{}
	return false
}

// Parse recursively analyzes the given reflect.Type and fills the provided TypeNode
// with its structural and metadata information.
//
// node: the current TypeNode to populate
// typ: the reflect.Type being parsed
// pkg: the package of the parent referer
// fqnTypesProcessed: map to track already parsed types (avoiding recursion loops)
func Parse(node *data.TypeNode, typ reflect.Type, pkg string, fqnTypesProcessed map[string]struct{}) {
	kind := typ.Kind()
	switch kind {
	case reflect.Array:
		ParseArray(node, typ, pkg, fqnTypesProcessed)
	case reflect.Slice:
		ParseSlice(node, typ, pkg, fqnTypesProcessed)
	case reflect.Ptr:
		ParsePointer(node, typ, pkg, fqnTypesProcessed)
	case reflect.Struct:
		ParseStructure(node, typ, pkg, fqnTypesProcessed)
	case reflect.Map:
		ParseMap(node, typ, pkg, fqnTypesProcessed)
	case reflect.Interface:
		ParseInterface(node, typ, pkg, fqnTypesProcessed)
	case reflect.Func:
		ParseFunc(node, typ, pkg)
	}
	if kind == reflect.String || (kind > reflect.Invalid && kind <= reflect.Complex128) {
		ParseBuiltin(node, pkg, typ)
	}
}

// ParseBuiltin handles built-in Go types (string, int, bool, etc.).
// It sets the node kind to Builtin and determines if the type belongs to the same package.
func ParseBuiltin(node *data.TypeNode, pkg string, typ reflect.Type) {
	DefaultParsing(node, typ)
	node.Kind = data.Builtin
	node.SamePkgAsReferer = true
	if node.PkgPath != "" {
		node.SamePkgAsReferer = pkg == node.PkgPath
	}
}

// ParseInterface handles interface types.
// It marks the node as an interface, sets SamePkgAsReferer, and flags Err=true (unsupported for equality).
func ParseInterface(node *data.TypeNode, typ reflect.Type, pkg string, typesProcessed map[string]struct{}) {
	DefaultParsing(node, typ)
	node.Kind = data.Interface
	node.SamePkgAsReferer = pkg == node.PkgPath
	node.Err = true
}

// ParseInterface handles interface types.
// It marks the node as an interface, sets SamePkgAsReferer, and flags Err=true (unsupported for equality).
func ParseStructure(node *data.TypeNode, typ reflect.Type, pkg string, typesProcessed map[string]struct{}) {
	DefaultParsing(node, typ)
	node.Kind = data.Struct
	node.SamePkgAsReferer = pkg == node.PkgPath
	pkg = node.PkgPath
	// Initialize imports set
	node.Imports = map[string]struct{}{}
	if !node.SamePkgAsReferer {
		node.Imports[node.PkgPath] = struct{}{}
	}

	// Avoid re-parsing types we've already seen
	if TypeAlreadyVisited(typ, typesProcessed) {
		return
	}
	// Only parse fields if the struct has no custom Equal method
	if !node.HasEqual {
		StructFieldsEqual(node, typ, pkg, typesProcessed)
	}
	// Err will be true only if all fields have Err set to true
	node.Err = true
	for _, field := range node.Fields {
		node.Err = node.Err && field.Err
	}
}

// ParseFunc handles function types.
// It marks them as unsupported (Err=true).
func ParseFunc(node *data.TypeNode, typ reflect.Type, pkg string) {
	DefaultParsing(node, typ)
	node.Kind = data.Func
	node.Err = true
}

// ParseMap handles map types.
// It parses both the key type and value type recursively and merges their imports.
func ParseMap(node *data.TypeNode, typ reflect.Type, pkg string, typesProcessed map[string]struct{}) {
	DefaultParsing(node, typ)
	node.Kind = data.Map
	node.MapKeyType = typ.Key().Name()
	// Parse the map value type
	mapType := typ.Elem()
	mapNode := &data.TypeNode{
		UpNode: node,
	}
	node.SubNode = mapNode
	Parse(mapNode, mapType, pkg, typesProcessed)
	// Update PkgPath depending on whether the type is named or anonymous
	node.PkgPath = typ.Key().PkgPath()
	if node.Type != "" {
		node.PkgPath = typ.PkgPath()
	}
	node.PackagedType = typ.Key().String()
	node.SamePkgAsReferer = pkg == node.PkgPath
	// Merge imports from the value type (SubNode) and key type
	node.Imports = map[string]struct{}{}
	for subNodeImport := range node.SubNode.Imports {
		node.Imports[subNodeImport] = struct{}{}
	}
	if !node.SamePkgAsReferer && node.PkgPath != "" {
		node.Imports[node.PkgPath] = struct{}{}
	}
	node.Err = mapNode.Err
}

// ParseArray handles fixed-length array types.
// It parses the element type recursively.
func ParseArray(node *data.TypeNode, typ reflect.Type, pkg string, typesProcessed map[string]struct{}) {
	DefaultParsing(node, typ)
	node.Kind = data.Array
	node.Len = typ.Len()
	arrayType := typ.Elem()
	arrayNode := &data.TypeNode{
		UpNode: node,
	}
	node.SubNode = arrayNode
	Parse(arrayNode, arrayType, pkg, typesProcessed)
	// Propagate PkgPath and imports from the element type
	node.PkgPath = node.SubNode.PkgPath
	if node.Type != "" {
		node.PkgPath = typ.PkgPath()
	}
	node.Imports = node.SubNode.Imports
	node.Err = arrayNode.Err
}

// ParseSlice handles slice types.
// It parses the element type recursively and tracks whether it's in the same package.
func ParseSlice(node *data.TypeNode, typ reflect.Type, pkg string, typesProcessed map[string]struct{}) {
	DefaultParsing(node, typ)
	node.Kind = data.Slice
	sliceType := typ.Elem()
	sliceNode := &data.TypeNode{
		UpNode:           node,
		SamePkgAsReferer: node.Type != "",
	}
	node.SubNode = sliceNode
	if node.Type != "" {
		// Named slice type → same package as referer
		pkg = node.PkgPath
		node.SamePkgAsReferer = true
	}
	Parse(sliceNode, sliceType, pkg, typesProcessed)
	// If package path not yet set, inherit from element type
	if node.PkgPath == "" {
		node.PkgPath = node.SubNode.PkgPath
	}
	node.Imports = node.SubNode.Imports
	node.Err = sliceNode.Err
}

// ParsePointer handles pointer types.
// It parses the pointed-to type recursively and inherits its packaged type and imports.
func ParsePointer(node *data.TypeNode, typ reflect.Type, pkg string, typesProcessed map[string]struct{}) {
	DefaultParsing(node, typ)
	node.Kind = data.Pointer
	pointerType := typ.Elem()
	pointerNode := &data.TypeNode{
		UpNode: node,
	}
	node.SubNode = pointerNode
	Parse(pointerNode, pointerType, pkg, typesProcessed)
	if node.Type == "" {
		node.PkgPath = node.SubNode.PkgPath
	}
	node.PackagedType = node.SubNode.PackagedType
	node.Imports = node.SubNode.Imports
	node.Err = pointerNode.Err
}

// StructFieldsEqual parses the fields of a struct for equality/diff generation.
// It skips certain predefined meta types and parses each remaining field recursively.
func StructFieldsEqual(node *data.TypeNode, typ reflect.Type, pkg string, typesProcessed map[string]struct{}) {
	for i := 0; i < typ.NumField(); i++ {
		fieldType := typ.Field(i)
		// Skip predefined meta types (e.g., Kubernetes ObjectMeta)
		_, toSkip := typesToSkip[fieldType.Type.String()]
		if toSkip {
			continue
		}
		equalNode := &data.TypeNode{
			Name:   fieldType.Name,
			UpNode: node,
		}
		node.Fields = append(node.Fields, equalNode)
		Parse(equalNode, fieldType.Type, pkg, typesProcessed)
	}
}

// DefaultParsing sets the common metadata for a TypeNode from a reflect.Type.
// This includes type name, package path, packaged type string, comparability,
// availability of Equal/Diff methods, and package alias handling.
func DefaultParsing(node *data.TypeNode, typ reflect.Type) {
	node.Type = typ.Name()
	node.PkgPath = typ.PkgPath()
	node.PackagedType = typ.String()
	node.IsComparable = typ.Comparable()
	node.HasEqual = utils.HasEqualFor(typ)
	node.HasDiff = utils.HasDiffFor(typ)
	// Extract package name from the full type string
	pkgAndType := strings.SplitN(node.PackagedType, ".", 2)
	pkg := pkgAndType[0]
	// If there is a package alias, apply it to the packaged type
	if PkgAlias := utils.AliasPkg(pkg); PkgAlias != "" {
		node.PkgAlias = PkgAlias
	}
	if node.PkgAlias != "" {
		typName := pkgAndType[1]
		node.PackagedType = fmt.Sprintf("%s.%s", node.PkgAlias, typName)
	}
}

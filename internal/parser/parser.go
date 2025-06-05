package parser

import (
	"reflect"

	"github.com/haproxytech/eqdiff/internal/data"
	"github.com/haproxytech/eqdiff/internal/utils"
)

var typesToSkip = map[string]struct{}{
	"v1.TypeMeta":   {},
	"v1.ObjectMeta": {},
}

func TypeAlreadyVisited(typ reflect.Type, fqnTypesProcessed map[string]struct{}) bool {
	if typ.Name() == "" {
		return false
	}

	fqnType := typ.PkgPath() + "." + typ.Name()

	if _, found := fqnTypesProcessed[fqnType]; found {
		return true
	}

	fqnTypesProcessed[fqnType] = struct{}{}
	return false
}

func Parse(node *data.TypeNode, typ reflect.Type, pkg string, fqnTypesProcessed map[string]struct{}) {
	kind := typ.Kind()
	switch kind {
	case reflect.Struct:
		ParseStructure(node, typ, pkg, fqnTypesProcessed)
	}
}

func ParseStructure(node *data.TypeNode, typ reflect.Type, pkg string, typesProcessed map[string]struct{}) {
	DefaultParsing(node, typ)
	node.Kind = data.Struct
	node.SamePkgAsReferer = pkg == node.PkgPath
	pkg = node.PkgPath
	node.Imports = map[string]struct{}{}
	if !node.SamePkgAsReferer {
		node.Imports[node.PkgPath] = struct{}{}
	}

	if TypeAlreadyVisited(typ, typesProcessed) {
		return
	}
	if !node.HasEqual {
		StructFieldsEqual(node, typ, pkg, typesProcessed)
	}
}

func StructFieldsEqual(node *data.TypeNode, typ reflect.Type, pkg string, typesProcessed map[string]struct{}) {
	for i := 0; i < typ.NumField(); i++ {
		fieldType := typ.Field(i)
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

func DefaultParsing(node *data.TypeNode, typ reflect.Type) {
	node.Type = typ.Name()
	node.PkgPath = typ.PkgPath()
	node.PackagedType = typ.String()
	node.IsComparable = typ.Comparable()
	node.HasEqual = utils.HasEqualFor(typ)
	node.HasDiff = utils.HasDiffFor(typ)
}

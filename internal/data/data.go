package data

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
	"text/template"

	"github.com/haproxytech/go-method-gen/internal/utils"
)

// Constants used as keys when passing template data to Equal/Diff templates
const (
	ParameterTypeDataMap  = "ParameterType"  // Full type string for parameter
	EqualFuncNameDataMap  = "EqualFuncName"  // Name of the Equal function
	EqualityTestDataMap   = "EqualityTest"   // Expression for equality comparison
	InequalityTestDataMap = "InequalityTest" // Expression for inequality comparison

	DiffFuncNameDataMap = "DiffFuncName"     // Name of the Diff function
	DiffElementMap      = "DiffElement"      // Expression for diffing
	NodeNameMap         = "NodeName"         // Field name
	IsBuiltinSubNodeMap = "IsBuiltinSubNode" // Indicates if sub-node is a builtin type
	SubTypeMap          = "SubType"          // Type of sub-node
)

// Kind represents the kind of a type node (builtin, struct, array, slice, map, etc.)
type Kind int

const (
	Unknown Kind = 1 << iota
	Builtin
	Struct
	Array
	Slice
	Map
	Interface
	Pointer
	Func
)

// MarshalJSON allows Kind to be serialized to JSON as a string
func (k Kind) MarshalJSON() ([]byte, error) {
	return json.Marshal(KindToString(k))
}

// MarshalYAML allows Kind to be serialized to YAML as a string
func (k Kind) MarshalYAML() (interface{}, error) {
	return KindToString(k), nil
}

// TypeNode represents a type or field in the type hierarchy
type TypeNode struct {
	HasEqual         bool           // True if type has an existing Equal method
	HasDiff          bool           // True if type has an existing Diff method
	Name             string         // Field name, empty for root type
	Type             string         // Field type name
	PackagedType     string         // Fully qualified type name including package
	Kind             Kind           // Kind of the type
	IsComparable     bool           // True if type can be compared with ==
	PkgPath          string         // Package path for the type
	PkgAlias         string         // Alias used when importing the package
	SamePkgAsReferer bool           // True if type is in same package as reference
	Fields           []*TypeNode    // Child fields (for structs)
	Len              int            // Array length
	Value            *reflect.Value // Optional pointer to value
	Imports          map[string]struct{}
	MapKeyType       string
	SubNode          *TypeNode
	UpNode           *TypeNode `json:"-"`
	Err              bool
}

// IsForType returns true if this node represents a type (not a field)
func (en *TypeNode) IsForType() bool {
	return !en.IsForField()
}

// IsForField returns true if this node represents a field
func (en *TypeNode) IsForField() bool {
	return en.Name != ""
}

// Ctx holds information needed for code generation
type Ctx struct {
	PkgPath                                 string
	Pkg                                     string
	ObjectNameToHaveGeneration              string
	LeftSideComparison, RightSideComparison string
	EqualImplementation                     string
	InequalImplementation                   string
	DiffImplementation                      string
	EqualFuncName                           string
	DiffFuncName                            string
	DiffElement                             string
	ObjectKind                              string
	Type                                    string
	Imports                                 map[string]struct{}
	Err                                     bool
	DefinedType                             bool
	SubCtxs                                 []*Ctx
}

// KindToString converts a Kind enum to a string
func KindToString(kind Kind) string {
	switch kind {
	case Array:
		return "Array"
	case Slice:
		return "Slice"
	case Map:
		return "Map"
	case Struct:
		return "Struct"
	case Interface:
		return "Interface"
	case Builtin:
		return "Builtin"
	case Pointer:
		return "Pointer"
	case Func:
		return "Func"
	}
	return "Unknown"
}

// ApplyTemplateForEqual applies a text/template to generate the Equal function
// for the given node, storing the result in ctx.
func ApplyTemplateForEqual(node *TypeNode, ctx *Ctx, t *template.Template) {
	args := GetTemplateDataFromSubNodeEqual(node, ctx)
	sb := strings.Builder{}
	t.Execute(&sb, args)
	ctx.EqualFuncName = args[EqualFuncNameDataMap]
	ctx.EqualImplementation = sb.String()
}

// ApplyTemplateForDiff applies a text/template to generate the Diff function
// for the given node, storing the result in ctx.
func ApplyTemplateForDiff(node *TypeNode, ctx *Ctx, t *template.Template) {
	args := GetTemplateDataFromSubNodeDiff(node, ctx)
	sb := strings.Builder{}
	t.Execute(&sb, args)
	ctx.DiffFuncName = args[DiffFuncNameDataMap]
	ctx.DiffImplementation = sb.String()
}

// GetTemplateDataFromSubNodeEqual prepares template variables for generating Equal function
func GetTemplateDataFromSubNodeEqual(node *TypeNode, ctx *Ctx) map[string]string {
	var subValueEqual, subValueUnequal, subType string

	if len(ctx.SubCtxs) == 1 {
		subCtx := ctx.SubCtxs[0]
		subType = subCtx.Type
		equalFuncName := subCtx.EqualFuncName
		switch {
		case (node.SubNode.HasEqual || equalFuncName == "Equal") && node.Kind == Pointer:
			subValueEqual = "(*" + ctx.LeftSideComparison + ").Equal(*" + ctx.RightSideComparison + ")"
			subValueUnequal = "!" + subValueEqual
		case node.HasEqual || equalFuncName == "Equal":
			subValueEqual = ctx.LeftSideComparison + ".Equal(" + ctx.RightSideComparison + ")"
			subValueUnequal = "!" + subValueEqual
		case equalFuncName != "":
			subValueEqual = subCtx.EqualFuncName + "(" + ctx.LeftSideComparison + "," + ctx.RightSideComparison + ")"
			subValueUnequal = "!" + subValueEqual
		case equalFuncName == "" && node.Kind == Pointer:
			subValueEqual = "*" + ctx.LeftSideComparison + " == *" + ctx.RightSideComparison
			subValueUnequal = "*" + ctx.LeftSideComparison + " != *" + ctx.RightSideComparison
		default:
			subValueEqual = subCtx.EqualImplementation
			subValueUnequal = subCtx.InequalImplementation
		}
	}
	parameterType := GetTypeFromNode(node)
	equalFuncName := utils.EqualFuncName(parameterType)
	return map[string]string{
		ParameterTypeDataMap:  parameterType,
		EqualFuncNameDataMap:  equalFuncName,
		EqualityTestDataMap:   subValueEqual,
		InequalityTestDataMap: subValueUnequal,
		SubTypeMap:            subType,
	}
}

// GetTemplateDataFromSubNodeDiff prepares template variables for generating Diff function
func GetTemplateDataFromSubNodeDiff(node *TypeNode, ctx *Ctx) map[string]string {
	var subValueDiff, subType string
	if len(ctx.SubCtxs) == 1 {
		subCtx := ctx.SubCtxs[0]
		subType = subCtx.Type
		diffFuncName := subCtx.DiffFuncName
		switch {
		case (node.SubNode.HasDiff || diffFuncName == "Diff") && node.Kind == Pointer:
			subValueDiff = "(*" + ctx.LeftSideComparison + ").Diff(*" + ctx.RightSideComparison + ")"
		case node.HasDiff || diffFuncName == "Diff":
			subValueDiff = ctx.LeftSideComparison + ".Diff(" + ctx.RightSideComparison + ")"
		case diffFuncName != "":
			subValueDiff = subCtx.DiffFuncName + "(" + ctx.LeftSideComparison + "," + ctx.RightSideComparison + ")"
		default:
			subValueDiff = subCtx.DiffImplementation
		}
	}
	parameterType := GetTypeFromNode(node)
	isBuiltinSubNodeMap := "false"
	if node.SubNode != nil && node.SubNode.Kind == Builtin {
		isBuiltinSubNodeMap = "true"
	}
	diffFuncName := utils.DiffFuncName(parameterType)
	return map[string]string{
		ParameterTypeDataMap: parameterType,
		DiffFuncNameDataMap:  diffFuncName,
		DiffElementMap:       subValueDiff,
		NodeNameMap:          node.Name,
		IsBuiltinSubNodeMap:  isBuiltinSubNodeMap,
		SubTypeMap:           subType,
	}
}

// GetTypeFromNode returns the string representation of a type node
func GetTypeFromNode(node *TypeNode) string {
	if node == nil {
		return ""
	}
	if node.Type != "" && node.Kind != Struct {
		return node.Type
	}
	name := ""
	switch node.Kind {
	case Array:
		name = fmt.Sprintf("[%d]", node.Len) + GetTypeFromNode(node.SubNode)
	case Slice:
		name = "[]" + GetTypeFromNode(node.SubNode)
	case Map:
		var keyType string
		if node.SamePkgAsReferer {
			keyType = node.MapKeyType
		} else {
			keyType = node.PackagedType
		}
		name += "map[" + keyType + "]" + GetTypeFromNode(node.SubNode)
	case Pointer:
		name += "*" + GetTypeFromNode(node.SubNode)
	case Func:
		name = "FuncIsForbidden" // placeholder for unsupported function type
	case Struct:
		if node.SamePkgAsReferer {
			name = node.Type
		} else {
			name = node.PackagedType
		}
	default:
		name = node.Type
	}

	return name
}

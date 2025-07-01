package data

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
	"text/template"

	"github.com/haproxytech/eqdiff/internal/utils"
)

const (
	ParameterTypeDataMap  = "ParameterType"
	EqualFuncNameDataMap  = "EqualFuncName"
	EqualityTestDataMap   = "EqualityTest"
	InequalityTestDataMap = "InequalityTest"

	DiffFuncNameDataMap = "DiffFuncName"
	DiffElementMap      = "DiffElement"
	NodeNameMap         = "NodeName"
	IsBuiltinSubNodeMap = "IsBuiltinSubNode"
	SubTypeMap          = "SubType"
)

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

func (k Kind) MarshalJSON() ([]byte, error) {
	return json.Marshal(KindToString(k))
}

func (k Kind) MarshalYAML() (interface{}, error) {
	return KindToString(k), nil
}

type TypeNode struct {
	HasEqual         bool   // Does it have an existing equal method ?
	HasDiff          bool   // Does it have an existing diff method ?
	Name             string // Field Name, empty for root struct
	Type             string // Type of the the field
	PackagedType     string // Packaged Type of the the field
	Kind             Kind
	IsComparable     bool   // Can we compare with == ?
	PkgPath          string // Package path for the type
	SamePkgAsReferer bool
	Fields           []*TypeNode // Slice of embeded fields
	Len              int         // array length
	Value            *reflect.Value
	Imports          map[string]struct{}
	MapKeyType       string
	SubNode          *TypeNode
	UpNode           *TypeNode `json:"-"`
	Err              bool
}

func (en *TypeNode) IsForType() bool {
	return !en.IsForField()
}

func (en *TypeNode) IsForField() bool {
	return en.Name != ""
}

type Ctx struct {
	PkgPath                                 string
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

func ApplyTemplateForEqual(node *TypeNode, ctx *Ctx, t *template.Template) {
	args := GetTemplateDataFromSubNodeEqual(node, ctx)
	sb := strings.Builder{}
	t.Execute(&sb, args)
	ctx.EqualFuncName = args[EqualFuncNameDataMap]
	ctx.EqualImplementation = sb.String()
}

func ApplyTemplateForDiff(node *TypeNode, ctx *Ctx, t *template.Template) {
	args := GetTemplateDataFromSubNodeDiff(node, ctx)
	sb := strings.Builder{}
	t.Execute(&sb, args)
	ctx.DiffFuncName = args[DiffFuncNameDataMap]
	ctx.DiffImplementation = sb.String()
}

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
		name = "FuncIsForbidden"
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

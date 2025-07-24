package utils

import (
	"encoding/json"
	"log"
	"path"
	"reflect"
	"regexp"
	"strconv"
	"strings"
)

func Fqn(packagedName string) string {
	packagedName = strings.ReplaceAll(packagedName, "[]", "[Slice]")
	re := regexp.MustCompile(`(\[\]|[^a-zA-Z0-9\*]+)`)
	words := re.Split(packagedName, -1)

	for i, word := range words {
		// Array case
		if num, err := strconv.Atoi(word); err == nil {
			words[i] = "Array" + strconv.Itoa(num)
			continue
		}
		// Pointer case
		if strings.HasPrefix(word, "*") {
			originalWord := word
			numPointerStar := strings.Count(word, "*")
			words[i] = strings.Repeat("Pointer", numPointerStar) + capitalize(originalWord[numPointerStar:])
		} else {
			words[i] = capitalize(word)
		}
	}
	return strings.Join(words, "")
}

func EqualFuncName(input string) string {
	return "Equal" + Fqn(input)
}

func DiffFuncName(input string) string {
	return "Diff" + Fqn(input)
}

func capitalize(s string) string {
	if s == "" {
		return s
	}
	return strings.ToUpper(s[:1]) + s[1:]
}

func SerializeJSON[T any](node T) string {
	jsonData, err := json.MarshalIndent(node, "", "  ")
	if err != nil {
		log.Fatalf("JSON Serialization error : %v", err)
	}
	return string(jsonData)
}

func HasEqualFor(typ reflect.Type) bool {
	if typ.PkgPath() == "" {
		return false
	}
	method, found := typ.MethodByName("Equal")
	return found && method.Type.NumIn() == 2 && // one and only one argument
		method.Type.In(0).AssignableTo(typ) && // of same type as node
		method.Type.NumOut() == 1 && // one and only one return value
		method.Type.Out(0).Kind() == reflect.Bool // of type bool
}

func HasDiffFor(typ reflect.Type) bool {
	if typ.PkgPath() == "" {
		return false
	}
	method, found := typ.MethodByName("Diff")

	var correctReturnType bool
	if found {
		outType := method.Type.Out(0)
		// return type of map[string][]interface{}
		if outType.Kind() == reflect.Map {
			keyType := outType.Key()
			valueType := outType.Elem()

			if keyType.Kind() == reflect.String &&
				valueType.Kind() == reflect.Slice &&
				valueType.Elem().Kind() == reflect.Interface {
				correctReturnType = true
			}
		}
	}
	return found && method.Type.NumIn() == 2 && // one and only one argument
		method.Type.In(0).AssignableTo(typ) && // of same type as node
		method.Type.NumOut() == 1 && // one and only one return value
		correctReturnType
}

func ExtractPkg(fullpkg string) string {
	pkg := strings.Split(fullpkg, "/")
	return pkg[len(pkg)-1]
}

func AliasPkg(pkg string) string {
	alias := strings.ReplaceAll(pkg, "-", "_")
	if len(alias) > 0 && alias[0] >= '0' && alias[0] <= '9' {
		alias = "_" + alias
	}
	if alias == pkg {
		return ""
	}
	return alias
}

func AliasImport(importPath string) string {
	return AliasPkg(path.Base(importPath))
}

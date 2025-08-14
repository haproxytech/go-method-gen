package utils

import (
	"encoding/json"
	"log"
	"path"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"unicode"
)

// Fqn generates a "Fully Qualified Name"-like string from a Go type name,
// transforming it into a form suitable for use in generated function names.
// It handles slices, arrays, and pointers by replacing symbols with
// descriptive words (e.g., []MyType -> SliceMyType, **MyType -> PointerPointerMyType).
func Fqn(packagedName string) string {
	// Replace slice notation [] with a placeholder to distinguish it from arrays
	packagedName = strings.ReplaceAll(packagedName, "[]", "[Slice]")
	// Split on non-alphanumeric characters or "[]"
	re := regexp.MustCompile(`(\[\]|[^a-zA-Z0-9\*]+)`)
	words := re.Split(packagedName, -1)

	for i, word := range words {
		// Array case: if the segment is purely numeric, treat it as an array length
		if num, err := strconv.Atoi(word); err == nil {
			words[i] = "Array" + strconv.Itoa(num)
			continue
		}
		// Pointer case: detect leading '*' characters
		if strings.HasPrefix(word, "*") {
			originalWord := word
			numPointerStar := strings.Count(word, "*")
			// Repeat "Pointer" once for each '*' and capitalize the remaining part
			words[i] = strings.Repeat("Pointer", numPointerStar) + capitalize(originalWord[numPointerStar:])
		} else {
			// Normal case: just capitalize the word
			words[i] = capitalize(word)
		}
	}
	return strings.Join(words, "")
}

// EqualFuncName returns the generated Equal function name for a given type name.
func EqualFuncName(input string) string {
	return "Equal" + Fqn(input)
}

// DiffFuncName returns the generated Diff function name for a given type name.
func DiffFuncName(input string) string {
	return "Diff" + Fqn(input)
}

// capitalize returns the input string with its first character in uppercase.
func capitalize(s string) string {
	if s == "" {
		return s
	}
	return strings.ToUpper(s[:1]) + s[1:]
}

// SerializeJSON serializes any Go value to a pretty-printed JSON string.
// If serialization fails, the program will log a fatal error and exit.
func SerializeJSON[T any](node T) string {
	jsonData, err := json.MarshalIndent(node, "", "  ")
	if err != nil {
		log.Fatalf("JSON Serialization error : %v", err)
	}
	return string(jsonData)
}

// HasEqualFor checks whether a given type defines an Equal method
// with the exact signature: func (T) Equal(T) bool.
func HasEqualFor(typ reflect.Type) bool {
	if typ.PkgPath() == "" {
		return false
	}
	method, found := typ.MethodByName("Equal")
	return found && method.Type.NumIn() == 2 && // method has exactly one argument (plus the receiver)
		method.Type.In(0).AssignableTo(typ) && // receiver matches the given type
		method.Type.NumOut() == 1 && // exactly one return value
		method.Type.Out(0).Kind() == reflect.Bool // return type is bool
}

// HasDiffFor checks whether a given type defines a Diff method
// with the exact signature: func (T) Diff(T) map[string][]interface{}.
func HasDiffFor(typ reflect.Type) bool {
	if typ.PkgPath() == "" {
		return false
	}
	method, found := typ.MethodByName("Diff")

	var correctReturnType bool
	if found {
		outType := method.Type.Out(0)
		// Check that return type is map[string][]interface{}
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
	return found && method.Type.NumIn() == 2 && // one argument (plus receiver)
		method.Type.In(0).AssignableTo(typ) && // one argument (plus receiver)
		method.Type.NumOut() == 1 && // single return value
		correctReturnType
}

// ExtractPkg returns the last element of a full Go import path,
// which corresponds to the package name (e.g., "github.com/foo/bar" -> "bar").
func ExtractPkg(fullpkg string) string {
	pkg := strings.Split(fullpkg, "/")
	return pkg[len(pkg)-1]
}

// AliasPkg converts a package name to a valid Go import alias.
// - Replaces hyphens with underscores.
// - If the name starts with a digit, prefixes it with an underscore.
// - Returns an empty string if no alias is necessary.
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

// AliasImport applies AliasPkg to the last path segment of an import path.
func AliasImport(importPath string) string {
	return AliasPkg(path.Base(importPath))
}

// GenerateAliasVarName converts a packaged type name like "models.Backend" or "v1.Backend"
// into a variable-safe Go identifier like "modelsBackend" or "v1Backend".
// Non-letter and non-digit characters are removed.
func GenerateAliasVarName(packagedType string) string {
	var b strings.Builder
	for _, r := range packagedType {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			b.WriteRune(r)
		}
	}
	return b.String()
}

package common

import (
	"strings"

	"github.com/haproxytech/go-method-gen/internal/data"
)

func GetPackage(node *data.TypeNode) string {
	var pkg string
	packagedTypeSplits := strings.Split(node.PackagedType, ".")
	if len(packagedTypeSplits) > 1 {
		pkg = packagedTypeSplits[0]
	}
	return pkg
}

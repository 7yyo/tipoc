package widget

import (
	"embed"
	"strings"
)

//go:embed "catalog"
var catalogPath embed.FS

const catalog = "catalog"

type Catalog struct {
	Value string
}

func (c Catalog) String() string {
	return c.Value
}

func newCatalog(v string) *Catalog {
	return &Catalog{
		Value: v,
	}
}

func isSafety(v string) bool {
	return strings.HasPrefix(v, "6")
}

func isLoadData(v string) bool {
	return strings.HasPrefix(v, "8")
}

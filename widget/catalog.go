package widget

import (
	"embed"
	"io/fs"
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

func readCatalog() (fs.File, error) {
	return catalogPath.Open(catalog)
}

func isSafety(v string) bool {
	return strings.HasPrefix(v, "6")
}

func isLoadData(v string) bool {
	return strings.HasPrefix(v, "8")
}

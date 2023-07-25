package widget

import (
	"embed"
	"io/fs"
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

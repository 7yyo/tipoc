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

func isOnlineDDLAddIndex(v string) bool {
	return strings.HasPrefix(v, "3.5")
}

func isDataDistribution(v string) bool {
	return strings.HasPrefix(v, "1.19")
}

func isSafety(v string) bool {
	return strings.HasPrefix(v, "6")
}

func isLoadDataTPCC(v string) bool {
	return strings.HasPrefix(v, "8.1")
}

func isLoadDataImportInto(v string) bool {
	return strings.HasPrefix(v, "8.2")
}

func isLoadData(v string) bool {
	return strings.HasPrefix(v, "8.3")
}

func isSelectIntoOutFile(v string) bool {
	return strings.HasPrefix(v, "8.4")
}

func isInstallSysBench(v string) bool {
	return strings.HasPrefix(v, "20.1")
}

package server

import (
	"embed"
	"fmt"
	"io/fs"
	"io/ioutil"
	"os"
	"path/filepath"
	"pictorial/log"
	"strings"
)

//go:embed "script/*.sql"
var srt embed.FS

const suffix = ".sql"
const splitLine = "## -"
const tblName = "${TABLE_NAME}"

type script struct {
	name string
	tp   int
	sql  []string
}

func getScript(name string, o int) ([]string, error) {
	fName := fmt.Sprintf("%s%s", name, suffix)
	var fPath string
	var f fs.File
	var err error
	switch o {
	case sql:
		fPath = filepath.Join("script", fName)
		f, err = srt.Open(fPath)
	case other:
		fPath = filepath.Join(others, fName)
		f, err = os.Open(fPath)
	}
	if err != nil {
		log.Logger.Warnf("%s is not exists, skip", fPath)
		return nil, nil
	}
	defer f.Close()
	data, _ := ioutil.ReadAll(f)
	text := string(data)
	if o == sql {
		tName := strings.Split(name, " ")[1]
		text = strings.ReplaceAll(text, tblName, "poc."+tName)
	}
	return strings.Split(text, splitLine), nil
}

func isDisaster(s string) bool {
	return strings.HasPrefix(s, "7.5")
}

func whichOperator(s string) int {
	switch {
	case strings.HasPrefix(s, "5.2"):
		return scaleIn
	case strings.HasPrefix(s, "7.1"):
		return kill
	case strings.HasPrefix(s, "7.2"):
		return dataCorrupted
	case strings.HasPrefix(s, "7.3"):
		return crash
	case strings.HasPrefix(s, "7.4"):
		return recoverSystemd
	case strings.HasPrefix(s, "7.5"):
		return disaster
	default:
		return 0
	}
}

func isOperator(s string) bool {
	return strings.HasPrefix(s, "5.2") ||
		strings.HasPrefix(s, "7.1") ||
		strings.HasPrefix(s, "7.2") ||
		strings.HasPrefix(s, "7.3") ||
		strings.HasPrefix(s, "7.4") ||
		strings.HasPrefix(s, "7.5")
}

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

const (
	normal = iota
)

const suffix = ".sql"
const splitLine = "## -"
const tblName = "${TABLE_NAME}"

type script struct {
	name string
	tp   int
	sql  []string
}

func isSQL(a int) bool {
	return a == sql
}

func isOther(a int) bool {
	return a == other
}

func getScript(name string, a int) ([]string, error) {
	fName := fmt.Sprintf("%s%s", name, suffix)
	var fPath string
	var f fs.File
	var err error
	switch a {
	case sql:
		fPath = filepath.Join("script", fName)
		f, err = srt.Open(fPath)
	case other:
		fPath = filepath.Join(others, fName)
		f, err = os.Open(fPath)
	}
	if err != nil {
		log.Logger.Warnf("%s is not exists, skip", fName)
		return nil, nil
	}
	defer f.Close()
	data, _ := ioutil.ReadAll(f)
	text := string(data)
	if a == sql {
		tName := strings.Split(name, " ")[1]
		text = strings.ReplaceAll(text, tblName, tName)
	}
	return strings.Split(text, splitLine), nil
}

func isKill(s string) bool {
	return strings.HasPrefix(s, "7.1")
}

func isCrash(s string) bool {
	return strings.HasPrefix(s, "7.3")
}

func isRecoverSystemd(s string) bool {
	return strings.HasPrefix(s, "7.4")
}

func isDisaster(s string) bool {
	return strings.HasPrefix(s, "7.5")
}

func isDataCorrupted(s string) bool {
	return strings.HasPrefix(s, "7.2")
}

package server

import (
	"embed"
	"fmt"
	"io/fs"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
)

//go:embed "script/*.sql"
var srt embed.FS

const suffix = ".sql"
const splitLine = "## -"
const dbName = "poc"
const tablePlaceholder = "${TABLE_NAME}"

type option struct {
	value     string
	isCatalog bool
	operatorTp
	componentTp
}

func (o option) String() string {
	return o.value
}

type operatorTp int

const (
	sql operatorTp = iota
	otherSql
	safetySql
	scaleIn
	kill
	dataCorrupted
	crash
	recoverSystemd
	disaster
	reboot
)

func isNormal(name string) bool {
	return strings.HasPrefix(name, "1") ||
		strings.HasPrefix(name, "2") ||
		strings.HasPrefix(name, "3") ||
		strings.HasPrefix(name, "4")
}

func isScalability(name string) bool {
	return strings.HasPrefix(name, "5")
}

func isHighAvailability(name string) bool {
	return strings.HasPrefix(name, "7")
}

var operatorMapping = map[string]operatorTp{
	"5.2": scaleIn,
	"7.1": kill,
	"7.2": dataCorrupted,
	"7.3": crash,
	"7.4": recoverSystemd,
	"7.5": disaster,
	"7.6": reboot,
}

func getOperatorTp(oTp operatorTp) string {
	switch oTp {
	case sql:
		return "sql"
	case kill:
		return "kill"
	case crash:
		return "crash"
	case dataCorrupted:
		return "data_corrupted"
	case recoverSystemd:
		return "recover_systemd"
	case disaster:
		return "disaster"
	case scaleIn:
		return "scale_in"
	case reboot:
		return "reboot"
	default:
		return ""
	}
}

type script struct {
	name string
	sql  []string
}

func getScript(op option) (*script, error) {
	var fp string
	var f fs.File
	var err error
	var fname string
	switch op.operatorTp {
	case sql, safetySql:
		fname = fmt.Sprintf("%s%s", op.value, suffix)
		fp = filepath.Join("script", fname)
		f, err = srt.Open(fp)
	case otherSql:
		fname = filepath.Join(others, op.value)
		f, err = os.Open(fname)
	}
	if err != nil {
		return nil, fmt.Errorf("%s isn't exists", fname)
	}
	defer f.Close()
	data, _ := ioutil.ReadAll(f)
	text := string(data)
	if op.operatorTp == sql || op.operatorTp == safetySql {
		tblName := strings.Split(op.String(), " ")[1]
		fullName := fmt.Sprintf("%s.%s", dbName, tblName)
		text = strings.ReplaceAll(text, tablePlaceholder, fullName)
	}
	return &script{
		name: fname,
		sql:  strings.Split(text, splitLine),
	}, nil
}

func isDisaster(s string) bool {
	return strings.HasPrefix(s, "7.5")
}

func isSafety(s string) bool {
	return strings.HasPrefix(s, "6")
}

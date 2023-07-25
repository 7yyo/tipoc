package widget

import (
	"embed"
	"fmt"
	"github.com/gizak/termui/v3/widgets"
	"io/ioutil"
	"path/filepath"
	"pictorial/comp"
	"pictorial/operator"
	"strings"
)

//go:embed "script/*.sql"
var scriptPath embed.FS

type Example struct {
	Value string
	CType comp.CType
	operator.OType
}

var OTypeCompMapping = map[string]operator.OType{
	"7.1": operator.RecoverSystemd,
	"7.2": operator.Kill,
	"7.3": operator.DataCorrupted,
	"7.4": operator.Crash,
	"7.5": operator.Disaster,
	"7.6": operator.Reboot,
	"7.7": operator.DiskFull,
	"9.2": operator.ScaleIn,
}

func IsCompCatalogMapping(idx string) bool {
	if _, ok := OTypeCompMapping[idx]; ok {
		return true
	}
	return false
}

func (e Example) String() string {
	return e.Value
}

func NewExample(v string, c comp.CType, o operator.OType) *Example {
	return &Example{
		Value: v,
		CType: c,
		OType: o,
	}
}

func getIdxByValue(v string) string {
	return strings.Split(v, " ")[0]
}

func getNameByValue(v string) string {
	return strings.Split(v, " ")[1]
}

func (e Example) isConflict(o operator.OType) bool {
	return e.OType != o
}

func ChangeToExample(node *widgets.TreeNode) *Example {
	return node.Value.(*Example)
}

const fragment = "## -\n"

func (e Example) getScriptValue() ([]string, error) {
	fName := e.scriptPath()
	v, err := e.scriptValue(fName)
	if err != nil {
		return nil, err
	}
	return strings.Split(v, fragment), nil
}

func (e Example) scriptPath() string {
	switch e.OType {
	case operator.Script, operator.SafetyScript:
		return fmt.Sprintf("script/%s%s", e.Value, ".sql")
	case operator.OtherScript:
		return filepath.Join(OtherConfig, e.Value)
	default:
		return ""
	}
}

func (e Example) scriptValue(fName string) (string, error) {
	var v string
	switch e.OType {
	case operator.Script, operator.SafetyScript:
		output, err := scriptPath.ReadFile(fName)
		if err != nil {
			return "", e.scriptIsNotExists()
		}
		v = e.replaceTableName(output)
	case operator.OtherScript:
		output, err := ioutil.ReadFile(fName)
		if err != nil {
			return "", e.scriptIsNotExists()
		}
		v = string(output)
	}
	return v, nil
}

func (e Example) scriptIsNotExists() error {
	return fmt.Errorf("[%s] is not exists, skip", e.Value)
}

const tableNameIdentification = "${TABLE_NAME}"

func (e Example) replaceTableName(o []byte) string {
	tableName := strings.Split(e.Value, " ")[1]
	tableName = fmt.Sprintf("%s.%s", "poc", tableName)
	return strings.ReplaceAll(string(o), tableNameIdentification, tableName)
}

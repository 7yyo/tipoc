package widget

import (
	"embed"
	"fmt"
	"github.com/gizak/termui/v3/widgets"
	"io/ioutil"
	"path/filepath"
	"pictorial/comp"
	"strings"
)

//go:embed "script/*.sql"
var scriptPath embed.FS

type Example struct {
	Value string
	CType comp.CType
	OType
}

type OType int

const (
	Script OType = iota
	SafetyScript
	OtherScript
	ScaleIn
	Kill
	DataCorrupted
	Crash
	RecoverSystemd
	Disaster
	Reboot
	LoadDataTPCC
)

func GetOTypeValue(o OType) string {
	switch o {
	case Script:
		return "script"
	case SafetyScript:
		return "safetyScript"
	case OtherScript:
		return "otherScript"
	case ScaleIn:
		return "scale_in"
	case Kill:
		return "kill"
	case DataCorrupted:
		return "data_corrupted"
	case Crash:
		return "crash"
	case RecoverSystemd:
		return "recover_systemd"
	case Disaster:
		return "disaster"
	case Reboot:
		return "reboot"
	case LoadDataTPCC:
		return "load_data_tpc-c"
	default:
		return ""
	}
}

var OTypeMapping = map[string]OType{
	"5.2": ScaleIn,
	"7.1": RecoverSystemd,
	"7.2": Kill,
	"7.3": DataCorrupted,
	"7.4": Crash,
	"7.5": Disaster,
	"7.6": Reboot,
}

func (e Example) String() string {
	return e.Value
}

func NewExample(v string, c comp.CType, o OType) *Example {
	return &Example{
		Value: v,
		CType: c,
		OType: o,
	}
}

func getIdxByValue(v string) string {
	return strings.Split(v, " ")[0]
}

func (e Example) isConflict(o OType) bool {
	return e.OType != o
}

func ChangeToExample(node *widgets.TreeNode) *Example {
	return node.Value.(*Example)
}

const fragment = "## -"

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
	case Script, SafetyScript:
		return fmt.Sprintf("script/%s%s", e.Value, ".sql")
	case OtherScript:
		return filepath.Join(OtherConfig, e.Value)
	default:
		return ""
	}
}

func (e Example) scriptValue(fName string) (string, error) {
	var v string
	switch e.OType {
	case Script, SafetyScript:
		output, err := scriptPath.ReadFile(fName)
		if err != nil {
			return "", e.scriptIsNotExists()
		}
		v = e.replaceTableName(output)
	case OtherScript:
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

func (e Example) replaceTableName(o []byte) string {
	tableName := strings.Split(e.Value, " ")[1]
	tableName = fmt.Sprintf("%s.%s", "poc", tableName)
	return strings.ReplaceAll(string(o), "${TABLE_NAME}", tableName)
}

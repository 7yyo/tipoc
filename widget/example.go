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

func getIdxByName(v string) string {
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
	var v string
	switch e.OType {
	case Script, SafetyScript:
		fname := fmt.Sprintf("script/%s%s", e.Value, ".sql")
		output, err := scriptPath.ReadFile(fname)
		if err != nil {
			return nil, err
		}
		tableName := strings.Split(e.Value, " ")[1]
		full := fmt.Sprintf("%s.%s", "poc", tableName)
		v = strings.ReplaceAll(string(output), "${TABLE_NAME}", full)
	case OtherScript:
		fname := filepath.Join(OtherConfig, e.Value)
		output, err := ioutil.ReadFile(fname)
		if err != nil {
			return nil, err
		}
		v = string(output)
	}
	return strings.Split(v, fragment), nil
}

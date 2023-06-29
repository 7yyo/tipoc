package comp

import (
	"fmt"
	"pictorial/ssh"
	"strings"
)

var PdAddr string

const topologyGrafana = "/topology/grafana"

const membersUrl = "http://%s/pd/api/v1/members"
const storeUrl = "http://%s/pd/api/v1/stores"
const tidbAllInfoUrl = "http://%s/info/all"

const Leader = "(L)"

type CType int

const (
	NoBody CType = iota
	TiDB
	PD
	TiKV
	TiFlash
	Grafana
)

type Mapping struct {
	Map map[CType][]Component
}

type Component struct {
	Host       string
	Port       string
	StatusPort string
	DeployPath string
	Labels     map[string]string
	Status     string
}

func New() (*Mapping, error) {
	m := Mapping{
		Map: make(map[CType][]Component),
	}
	if err := m.GetServer(); err != nil {
		return nil, err
	}
	if err := m.GetStore(); err != nil {
		return nil, err
	}
	if err := m.GetPD(); err != nil {
		return nil, err
	}
	if err := m.GetGrafana(); err != nil {
		return nil, err
	}
	return &m, nil
}

func GetCTypeValue(c CType) string {
	switch c {
	case TiDB:
		return "tidb"
	case TiKV:
		return "tikv"
	case PD:
		return "pd"
	case TiFlash:
		return "tiflash"
	case Grafana:
		return "grafana"
	default:
		return ""
	}
}

func (m *Mapping) GetComponent(c CType) []Component {
	return m.Map[c]
}

func GetDataPath(host, deployPath string, cType CType) (string, error) {
	deployPath = strings.TrimSuffix(deployPath, "/bin")
	var o []byte
	var err error
	switch cType {
	case TiKV:
		script := fmt.Sprintf("%s/scripts/run_tikv.sh", deployPath)
		o, err = ssh.S.RunSSH(host, fmt.Sprintf("grep -oP -- '--data-dir \\K[^\\n:]+' %s | tr -d ' '", script))
	case PD:
		script := fmt.Sprintf("%s/scripts/run_pd.sh", deployPath)
		o, err = ssh.S.RunSSH(host, fmt.Sprintf("grep -oP -- '--data-dir=\\K[^\\s]*' %s", script))
	default:
		return "", fmt.Errorf("only support tikv and pd")
	}
	if err != nil {
		return "", nil
	}
	dataPath := string(o)
	dataPath = processDataPath(dataPath)
	return dataPath, nil
}

func processDataPath(dp string) string {
	dp = strings.ReplaceAll(dp, "\"", "")
	dp = strings.ReplaceAll(dp, "\n", "")
	dp = strings.TrimSuffix(dp, "\\")
	return dp
}

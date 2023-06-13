package comp

var PdAddr string

const topologyTiDB = "/topology/tidb/"
const topologyGrafana = "/topology/grafana"

const membersUrl = "http://%s/pd/api/v1/members"
const storeUrl = "http://%s/pd/api/v1/stores"
const tidbAllInfoUrl = "http://%s/info/all"

type CType int

const (
	TiDB CType = iota
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

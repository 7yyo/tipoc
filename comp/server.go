package comp

import (
	"encoding/json"
	"fmt"
	"net"
	"pictorial/http"
	"pictorial/mysql"
	"strconv"
)

type ServerInfo struct {
	IP            string `json:"ip"`
	ListeningPort int    `json:"listening_port"`
	StatusPort    int    `json:"status_port"`
}

type ServersInfo struct {
	ServersNum     int                   `json:"servers_num"`
	AllServersInfo map[string]ServerInfo `json:"all_servers_info"`
}

func (m *Mapping) GetServer() error {
	host, statusPort, err := GetTiDBHostStatusPort()
	if err != nil {
		return err
	}
	addr := net.JoinHostPort(host, statusPort)
	r, err := http.Get(fmt.Sprintf(tidbAllInfoUrl, addr))
	if err != nil {
		return err
	}
	var ss ServersInfo
	if err := json.Unmarshal(r, &ss); err != nil {
		return err
	}
	cs := make([]Component, 0)
	for _, s := range ss.AllServersInfo {
		c := Component{
			Host:       s.IP,
			Port:       strconv.Itoa(s.ListeningPort),
			StatusPort: strconv.Itoa(s.StatusPort),
		}
		cs = append(cs, c)
	}
	m.Map[TiDB] = cs
	return nil
}

func GetTiDBHostStatusPort() (string, string, error) {
	rs, err := mysql.M.ExecuteSQL("SELECT * FROM information_schema.tidb_servers_info")
	if err != nil {
		return "", "", err
	}
	defer rs.Close()
	if rs == nil {
		return "", "", fmt.Errorf("please confirm that the [tidb] exists in the cluster")
	}
	host := string(rs.Values[0][1].AsString())
	statusPort := strconv.FormatInt(rs.Values[0][3].AsInt64(), 10)
	return host, statusPort, nil
}

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
	host, statusPort, err := mysql.M.GetTiDBHostStatusPort()
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

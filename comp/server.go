package comp

import (
	"encoding/json"
	"fmt"
	"net"
	"pictorial/etcd"
	"pictorial/mysql"
	"strconv"
	"strings"
)

type Server struct {
	DeployPath string `json:"deploy_path"`
}

func (m *Mapping) GetServer() error {
	pd := m.Map[PD][0]
	port := CleanLeaderFlag(pd.Port)
	pdAddr := net.JoinHostPort(pd.Host, port)
	rs, err := etcd.GetByPrefix(pdAddr, "/topology/tidb/")
	if err != nil {
		return err
	}
	var cs []Component
	for _, v := range rs.Kvs {
		switch {
		case strings.HasSuffix(string(v.Key), "info"):
			var c Component
			var s Server
			addr := strings.Split(string(v.Key), "/")[3]
			host, port, err := net.SplitHostPort(addr)
			if err != nil {
				return err
			}
			c.Host = host
			c.Port = port
			if err := json.Unmarshal(v.Value, &s); err != nil {
				return err
			}
			deployPath := strings.TrimSuffix(s.DeployPath, "/bin")
			c.DeployPath = deployPath
			cs = append(cs, c)
		}
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

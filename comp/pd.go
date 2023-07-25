package comp

import (
	"encoding/json"
	"fmt"
	"net"
	"pictorial/log"
	"pictorial/mysql"
	"pictorial/util/http"
	"strings"
)

const LeaderDistributionSQL = "use information_schema; " +
	"select trp.store_id, address, trs.region_id ,trs.db_name " +
	"from tikv_region_peers trp join tikv_store_status tss on tss.store_id = trp.store_id join tikv_region_status trs on trp.region_id = trs.region_id " +
	"where trs.table_name = '%s' and trp.is_leader = 1;"

type PlacementDriver struct {
	Members []struct {
		ClientURLs []string `json:"client_urls"`
		DeployPath string   `json:"deploy_path"`
	} `json:"members"`
	Leader struct {
		ClientURLs []string `json:"client_urls"`
	} `json:"leader"`
}

func (m *Mapping) GetPD() error {
	resp, err := http.Get(fmt.Sprintf(membersUrl, PdAddr))
	if err != nil {
		return err
	}
	var pd *PlacementDriver
	if err := json.Unmarshal(resp, &pd); err != nil {
		return err
	}
	for _, p := range pd.Members {
		addr := http.ClearHttpHeader(p.ClientURLs[0])
		host, port, err := net.SplitHostPort(addr)
		if err != nil {
			return err
		}
		if p.ClientURLs[0] == pd.Leader.ClientURLs[0] {
			port = fmt.Sprintf("%s%s", port, Leader)
		}
		c := Component{
			Host:       host,
			Port:       port,
			DeployPath: p.DeployPath,
		}
		m.Map[PD] = append(m.Map[PD], c)
	}
	return nil
}

func GetPdAddr() (string, error) {
	rs, err := mysql.M.ExecuteSQL("SELECT * FROM information_schema.cluster_info WHERE type = 'pd'")
	if err != nil {
		return "", err
	}
	defer rs.Close()
	if rs == nil {
		return "", fmt.Errorf("please confirm that pd node exists in the cluster")
	}
	pd := string(rs.Values[0][1].AsString())
	log.Logger.Debug("pd = %s", pd)
	return pd, nil
}

func CleanLeaderFlag(v string) string {
	return strings.Trim(v, Leader)
}

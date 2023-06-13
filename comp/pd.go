package comp

import (
	"encoding/json"
	"fmt"
	"net"
	"pictorial/http"
)

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
		c := Component{
			Host:       host,
			Port:       port,
			DeployPath: p.DeployPath,
		}
		m.Map[PD] = append(m.Map[PD], c)
	}
	return nil
}

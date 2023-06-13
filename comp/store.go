package comp

import (
	"encoding/json"
	"fmt"
	"net"
	"pictorial/http"
	"pictorial/ssh"
	"strings"
)

type Store struct {
	Count  int `json:"count"`
	Stores []struct {
		Store struct {
			Address    string `json:"address"`
			DeployPath string `json:"deploy_path"`
			Labels     []struct {
				Key   string `json:"key"`
				Value string `json:"value"`
			} `json:"labels"`
		} `json:"store"`
	} `json:"stores"`
}

func (m *Mapping) GetStore() error {
	resp, err := http.Get(fmt.Sprintf(storeUrl, PdAddr))
	if err != nil {
		return err
	}
	var store *Store
	if err := json.Unmarshal(resp, &store); err != nil {
		return err
	}
	for _, s := range store.Stores {
		host, port, err := net.SplitHostPort(s.Store.Address)
		if err != nil {
			return err
		}
		c := Component{
			Host:       host,
			Port:       port,
			DeployPath: s.Store.DeployPath,
			Labels:     map[string]string{},
		}
		isTiFlash := false
		for _, l := range s.Store.Labels {
			if l.Value == "tiflash" {
				isTiFlash = true
				break
			} else {
				c.Labels[l.Key] = l.Value
			}
		}
		if isTiFlash {
			port, err := getTiFlashPort(c.Host, c.DeployPath)
			if err != nil {
				return err
			}
			c.Port = port
			m.Map[TiFlash] = append(m.Map[TiFlash], c)
		} else {
			m.Map[TiKV] = append(m.Map[TiKV], c)
		}
	}
	return nil
}

func getTiFlashPort(host, deployPath string) (string, error) {
	tiflashConfigPath := strings.Replace(deployPath, "bin/tiflash", "conf/tiflash.toml", -1)
	port, err := ssh.S.RunSSH(host, fmt.Sprintf("grep tcp_port %s | awk -F '= ' '{print $2}'", tiflashConfigPath))
	if err != nil {
		return "", err
	}
	return strings.Replace(string(port), "\n", "", -1), nil
}

func GetLabelKey(store []Component) map[string]bool {
	label := make(map[string]bool)
	for _, s := range store {
		for k, _ := range s.Labels {
			label[k] = true
		}
	}
	return label
}

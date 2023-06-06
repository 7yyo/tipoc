package server

import (
	"context"
	"encoding/json"
	"fmt"
	"go.etcd.io/etcd/clientv3"
	"net"
	"path/filepath"
	"pictorial/http"
	"pictorial/log"
	"pictorial/mysql"
	"pictorial/panel"
	"pictorial/ssh"
	"regexp"
	"strconv"
	"strings"
	"time"
)

const (
	tidb    = "tidb"
	pd      = "pd"
	pdL     = "pd(L)"
	tikv    = "tikv"
	tiflash = "tiflash"
	grafana = "grafana"
)

type component struct {
	host       string
	port       string
	tp         string
	deployPath string
	labels     map[string]string
}

func getComponents() (map[string][]component, error) {
	pdAddr, err := mysql.M.GetPdAddr()
	if err != nil {
		return nil, err
	}
	components := make(map[string][]component)
	if err := getTiDB(pdAddr, components); err != nil {
		return nil, err
	}
	if err := getStore(pdAddr, components); err != nil {
		return nil, err
	}
	if err := getPd(pdAddr, components); err != nil {
		return nil, err
	}
	if err := getGrafana(pdAddr, components); err != nil {
		return nil, err
	}
	return components, nil
}

func getTiDB(pdAddr string, components map[string][]component) error {
	etcdCli, err := clientv3.New(clientv3.Config{
		Endpoints:   []string{pdAddr},
		DialTimeout: 5 * time.Second,
	})
	if err != nil {
		return err
	}
	defer etcdCli.Close()
	r, err := etcdCli.Get(context.TODO(), "/topology/tidb/", clientv3.WithPrefix())
	if err != nil {
		return err
	}
	var data struct {
		Host       string `json:"ip"`
		DeployPath string `json:"deploy_path"`
	}
	for _, v := range r.Kvs {
		if strings.HasSuffix(string(v.Key), "info") {
			if err := json.Unmarshal(v.Value, &data); err != nil {
				return err
			}
			c := component{
				host:       data.Host,
				port:       getTiDBPort(string(v.Key)),
				deployPath: data.DeployPath,
			}
			components[tidb] = append(components[tidb], c)
		}
	}
	return nil
}

func getTiDBPort(s string) string {
	re := regexp.MustCompile(`:(\d+)/`)
	matches := re.FindStringSubmatch(s)
	return matches[1]
}

func getStore(pdAddr string, components map[string][]component) error {
	uri := fmt.Sprintf("http://%s/pd/api/v1/stores", pdAddr)
	resp, err := http.Get(uri)
	if err != nil {
		return err
	}
	var data struct {
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
	if err := json.Unmarshal(resp, &data); err != nil {
		return err
	}
	for _, s := range data.Stores {
		host, port, err := net.SplitHostPort(s.Store.Address)
		if err != nil {
			return err
		}
		c := component{
			host:       host,
			port:       port,
			tp:         tikv,
			deployPath: s.Store.DeployPath,
		}
		isTiflash := false
		if len(s.Store.Labels) != 0 {
			lb := make(map[string]string)
			for _, v := range s.Store.Labels {
				if v.Value == tiflash {
					tiflashPort, err := getTiflashPort(c.host, c.deployPath)
					if err != nil {
						return err
					}
					c.tp = tiflash
					c.port = tiflashPort
					components[tiflash] = append(components[tiflash], c)
					isTiflash = true
				} else {
					lb[v.Key] = v.Value
				}
			}
			c.labels = lb
		}
		if !isTiflash {
			components[tikv] = append(components[tikv], c)
		}
	}
	return nil
}

func getTiflashPort(host, deployPath string) (string, error) {
	tiflashConfigPath := strings.Replace(deployPath, "bin/tiflash", "conf/tiflash.toml", -1)
	port, err := ssh.S.RunSSH(host, fmt.Sprintf("grep tcp_port %s | awk -F '= ' '{print $2}'", tiflashConfigPath))
	if err != nil {
		return "", err
	}
	return strings.Replace(string(port), "\n", "", -1), nil
}

func getPd(pdAddr string, components map[string][]component) error {
	uri := fmt.Sprintf("http://%s/pd/api/v1/members", pdAddr)
	resp, err := http.Get(uri)
	if err != nil {
		return err
	}
	var data struct {
		Members []struct {
			ClientURLs []string `json:"client_urls"`
			DeployPath string   `json:"deploy_path"`
		} `json:"members"`
		Leader struct {
			ClientURLs []string `json:"client_urls"`
		} `json:"leader"`
	}
	if err := json.Unmarshal(resp, &data); err != nil {
		return err
	}
	for _, member := range data.Members {
		for _, h := range member.ClientURLs {
			addr := strings.Split(h, "//")[1]
			host, port, err := net.SplitHostPort(addr)
			if err != nil {
				return err
			}
			c := component{
				host:       host,
				port:       port,
				tp:         pd,
				deployPath: member.DeployPath,
			}
			if h == data.Leader.ClientURLs[0] {
				c.tp = pdL
			}
			components[pd] = append(components[pd], c)
		}
	}
	return nil
}

const grafanaEtcd = "/topology/grafana"

func getGrafana(pdAddr string, components map[string][]component) error {
	etcdCli, err := clientv3.New(clientv3.Config{
		Endpoints: []string{
			pdAddr,
		},
		DialTimeout: 5 * time.Second,
	})
	if err != nil {
		return err
	}
	defer etcdCli.Close()
	rs, err := etcdCli.Get(context.Background(), grafanaEtcd)
	if err != nil {
		return err
	}
	var data struct {
		Host       string `json:"ip"`
		Port       int    `json:"port"`
		DeployPath string `json:"deploy_path"`
	}
	if err := json.Unmarshal(rs.Kvs[0].Value, &data); err != nil {
		return err
	}
	c := component{
		host:       data.Host,
		port:       strconv.Itoa(data.Port),
		deployPath: data.DeployPath,
		tp:         grafana,
	}
	components[grafana] = append(components[grafana], c)
	return nil
}

func (c *component) preCheck() error {

	var o []byte
	var err error
	s := ssh.S
	plugin := s.Carry.Plugin
	pluginName := filepath.Base(plugin)
	pluginPath := filepath.Join(c.deployPath, "plugins")
	pluginSelf := filepath.Join(pluginPath, pluginName)

	if o, err = s.DirWalk(c.host, pluginPath); err != nil {
		return err
	}
	if len(o) == 0 && plugin == "" {
		return fmt.Errorf("plugin & pluginPath is nil, skip render, you can quit now")
	} else if len(o) != 0 {
		printPlugins(string(o))
	} else {
		log.Logger.Infof("%s is nil, start to set %s.", pluginPath, plugin)
		target := fmt.Sprintf("%s@%s:%s", s.User, c.host, pluginPath)
		if o, err = s.Transfer(plugin, target); err != nil {
			return err
		}
		if _, err = s.UnZip(c.host, pluginSelf, pluginPath); err != nil {
			return err
		}
		if _, err = s.Restart("grafana"); err != nil {
			return err
		}
		log.Logger.Infof("%s -> %s complete.", plugin, pluginPath)
	}
	return nil
}

func (c *component) render(to string) error {
	if err := c.preCheck(); err != nil {
		return err
	}
	tok, err := c.newToken()
	if err != nil {
		return err
	}
	now, from := unixDuration()
	pls, err := panel.GetPanels()
	if err != nil {
		return err
	}
	s := ssh.S
	uri := "http://%s:%s/render/d-solo/%s/%s-%s?orgId=1&from=%s&to=%s&panelId=%s&width=1000&height=500&scale=3"
	source := filepath.Join(c.deployPath, "data", "png", "*")
	for _, p := range pls {
		cmd := fmt.Sprintf(uri, c.host, c.port, p.Org, s.ClusterName, p.Tab, from, now, p.ID) + "&tz=Asia%2FShanghai"
		kv := make(map[string]string)
		kv["Authorization"] = fmt.Sprintf("Bearer %s", tok.Key)
		out, err := http.NewRequestDo(cmd, http.MethodGet, nil, kv, "")
		if err != nil {
			return fmt.Errorf("%s failed: %v: %s", c, err, string(out))
		}
		sourcePath := fmt.Sprintf("%s@%s:%s", s.User, c.host, source)
		if _, err = s.Transfer(sourcePath, to); err != nil {
			return err
		}
		if _, err = s.Remove(c.host, source); err != nil {
			return err
		}
		log.Logger.Infof("[RENDER] %s", strings.ToUpper(p.Name))
	}
	return c.cleanToken(tok.Key)
}

type token struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
	Role string `json:"role"`
	Key  string `json:"key"`
}

func (c *component) newToken() (*token, error) {
	url := fmt.Sprintf("http://%s:%s/api/auth/keys", c.host, c.port)
	payload := fmt.Sprintf(`{"name":"%s", "role":"Admin"}`, dateFormat())
	auth := http.Auth{
		Username: "admin",
		Password: "admin",
	}
	kv := make(map[string]string)
	kv["Content-Type"] = "application/json"
	out, err := http.NewRequestDo(url, http.MethodPost, &auth, kv, payload)
	if err != nil {
		return nil, err
	}
	t := token{
		Role: "Admin",
	}
	if err := json.Unmarshal(out, &t); err != nil {
		return nil, err
	}
	log.Logger.Infof("new token: %s", string(out))
	return &t, nil
}

func (c *component) cleanToken(key string) error {
	url := fmt.Sprintf("http://%s:%s/api/auth/keys", c.host, c.port)
	kv := make(map[string]string)
	kv["Authorization"] = fmt.Sprintf("Bearer %s", key)
	out, err := http.NewRequestDo(url, http.MethodGet, nil, kv, "")
	if err != nil {
		return err
	}
	var tks []token
	if err := json.Unmarshal(out, &tks); err != nil {
		return err
	}
	for _, t := range tks {
		if err := c.dropToken(strconv.Itoa(t.ID)); err != nil {
			return err
		}
	}
	return nil
}

func (c *component) dropToken(id string) error {
	url := fmt.Sprintf("http://%s:%s/api/auth/keys/%s", c.host, c.port, id)
	auth := http.Auth{
		Username: "admin",
		Password: "admin",
	}
	kv := make(map[string]string)
	kv["Content-Type"] = "application/json"
	out, err := http.NewRequestDo(url, http.MethodDelete, &auth, kv, "")
	if err != nil {
		return err
	}
	log.Logger.Infof("drop token: %s: %s", id, string(out))
	return nil
}

func unixDuration() (string, string) {
	t := time.Now()
	now := t.UnixNano() / 1000000
	ot := t.Add(-30 * time.Minute)
	from := ot.UnixNano() / 1000000
	log.Logger.Infof("[TIME_HORIZON] %s ~ %s", timeFormat(t), timeFormat(ot))
	return strconv.Itoa(int(now)), strconv.Itoa(int(from))
}

func timeFormat(t time.Time) string {
	return t.Format("2006-01-02 15:04:05.000")
}

func printPlugins(o string) {
	log.Logger.Info("maybe plugins installed:")
	log.Logger.Info(o)
	log.Logger.Info("have a try now, good luck!")
}

func getLabelKey(components map[string][]component) map[string]bool {
	labelKey := make(map[string]bool)
	for _, s := range components[tikv] {
		for k, _ := range s.labels {
			labelKey[k] = true
		}
	}
	return labelKey
}

func (c *component) isPdLeader(addr string) string {
	if c.tp == pdL {
		addr += "(L)"
	}
	return addr
}

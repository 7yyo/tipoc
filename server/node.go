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
	tidb = iota
	pd
	pdL
	tikv
	tiflash
	grafana
)

func getNodeTp(tp int) string {
	switch tp {
	case tidb:
		return "tidb"
	case pd:
		return "pd"
	case pdL:
		return "pd(L)"
	case tikv:
		return "tikv"
	case tiflash:
		return "tiflash"
	case grafana:
		return "grafana"
	default:
		return ""
	}
}

type node struct {
	host       string
	port       string
	tp         int
	deployPath string
	labels     map[string]string
}

func getNodes() (map[string][]node, error) {
	rs, err := mysql.M.ExecuteSQL("SELECT * FROM information_schema.cluster_info WHERE type = 'pd'")
	if err != nil {
		return nil, err
	}
	if rs == nil {
		return nil, fmt.Errorf("please confirm that the pd exists in the cluster")
	}
	pdAddr := string(rs.Values[0][1].AsString())
	defer rs.Close()
	nodes := make(map[string][]node)
	if err := getTidb(pdAddr, nodes); err != nil {
		return nil, err
	}
	if err := getStore(pdAddr, nodes); err != nil {
		return nil, err
	}
	if err := getPd(pdAddr, nodes); err != nil {
		return nil, err
	}
	if err := getGrafana(pdAddr, nodes); err != nil {
		return nil, err
	}
	return nodes, nil
}

func getTidb(pdAddr string, nodes map[string][]node) error {
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
	db := getNodeTp(tidb)
	for _, v := range r.Kvs {
		if strings.HasSuffix(string(v.Key), "info") {
			if err := json.Unmarshal(v.Value, &data); err != nil {
				return err
			}
			node := node{
				host:       data.Host,
				port:       getTiDBPort(string(v.Key)),
				deployPath: data.DeployPath,
			}
			nodes[db] = append(nodes[db], node)
		}
	}
	return nil
}

func getTiDBPort(s string) string {
	re := regexp.MustCompile(`:(\d+)/`)
	matches := re.FindStringSubmatch(s)
	return matches[1]
}

func getStore(pdAddr string, nodes map[string][]node) error {
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
	kv := getNodeTp(tikv)
	flash := getNodeTp(tiflash)
	for _, s := range data.Stores {
		host, port, err := net.SplitHostPort(s.Store.Address)
		if err != nil {
			return err
		}
		node := node{
			host:       host,
			port:       port,
			tp:         tikv,
			deployPath: s.Store.DeployPath,
		}
		isTiflash := false
		if len(s.Store.Labels) != 0 {
			lb := make(map[string]string)
			for _, v := range s.Store.Labels {
				if v.Value == "tiflash" {
					node.tp = tiflash
					nodes[flash] = append(nodes[flash], node)
					isTiflash = true
				} else {
					lb[v.Key] = v.Value
				}
			}
			node.labels = lb
		}
		if !isTiflash {
			nodes[kv] = append(nodes[kv], node)
		}
	}
	return nil
}

func getPd(pdAddr string, nodes map[string][]node) error {
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
	p := getNodeTp(pd)
	for _, member := range data.Members {
		for _, h := range member.ClientURLs {
			addr := strings.Split(h, "//")[1]
			host, port, err := net.SplitHostPort(addr)

			if err != nil {
				return err
			}
			node := node{
				host:       host,
				port:       port,
				tp:         pd,
				deployPath: member.DeployPath,
			}
			if h == data.Leader.ClientURLs[0] {
				node.tp = pdL
			}
			nodes[p] = append(nodes[p], node)
		}
	}
	return nil
}

func getGrafana(pdAddr string, nodes map[string][]node) error {
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
	rs, err := etcdCli.Get(context.Background(), "/topology/grafana")
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
	node := node{
		host:       data.Host,
		port:       strconv.Itoa(data.Port),
		deployPath: data.DeployPath,
		tp:         grafana,
	}
	g := getNodeTp(grafana)
	nodes[g] = append(nodes[g], node)
	return nil
}

const killCmd = "kill -9 %s"
const tidbProcess = "bin/tidb-server -P %s"
const pdProcess = "advertise-client-urls=http://%s:%s"
const tikvProcess = "bin/tikv-server --addr 0.0.0.0:%s"
const tiflashProcess = "bin/tiflash/tiflash"
const grafanaProcess = "grafana-%s"

func (n *node) kill() error {
	addr := net.JoinHostPort(n.host, n.port)
	tp := getNodeTp(n.tp)
	var psCmd string
	switch n.tp {
	case tidb:
		psCmd = fmt.Sprintf(tidbProcess, n.port)
	case pd, pdL:
		psCmd = fmt.Sprintf(pdProcess, n.host, n.port)
	case tikv:
		psCmd = fmt.Sprintf(tikvProcess, n.port)
	case tiflash:
		psCmd = tiflashProcess
	case grafana:
		psCmd = fmt.Sprintf(grafanaProcess, n.port)
	default:
		return fmt.Errorf("unsupported type: %s", tp)
	}
	processIDs, err := ssh.S.GetProcessID(n.host, psCmd)
	if err != nil {
		return err
	}
	if len(processIDs) == 0 {
		log.Logger.Warnf("[KILL] [%s] %s is offline, skip.", tp, addr)
		return nil
	}
	log.Logger.Infof("[KILL] [%s] [%s] - %v", tp, addr, processIDs)
	for _, pID := range processIDs {
		o, err := ssh.S.RunSSH(n.host, fmt.Sprintf(killCmd, pID))
		if err != nil {
			log.Logger.Warnf("[KILL] [%s] %s {%s} failed: %v: %s", tp, addr, pID, err, string(o))
		}
	}
	return nil
}

func (n *node) dataCorrupted() error {
	return fmt.Errorf("no GA")
}

const systemdPath = "/etc/systemd/system/"
const alwaysToNo = "sudo sed -i 's/always/no/g' %s"
const reloadSystemd = "sudo systemctl daemon-reload"
const noToAlways = "sudo sed -i 's/no/always/g' %s"
const service = "%s-%s.service"

func (n *node) systemd(a int) error {
	var systemd string
	tp := getNodeTp(n.tp)
	tp = strings.Replace(tp, "(L)", "", -1)
	systemd = fmt.Sprintf(service, tp, n.port)
	service := filepath.Join(systemdPath, systemd)
	addr := net.JoinHostPort(n.host, n.port)
	var c string
	switch a {
	case crash:
		c = fmt.Sprintf(alwaysToNo, service)
	case recoverSystemd:
		c = fmt.Sprintf(noToAlways, service)
		log.Logger.Infof("[RECOVER_SYSTEMD] [%s] %s", tp, addr)
	}
	if _, err := ssh.S.RunSSH(n.host, c); err != nil {
		return err
	}
	if _, err := ssh.S.RunSSH(n.host, reloadSystemd); err != nil {
		return err
	}
	if a == crash {
		return n.kill()
	}
	return nil
}

func (n *node) preCheck() error {

	var o []byte
	var err error
	s := ssh.S
	plugin := s.Carry.Plugin
	pluginName := filepath.Base(plugin)
	pluginPath := filepath.Join(n.deployPath, "plugins")
	pluginSelf := filepath.Join(pluginPath, pluginName)

	if o, err = s.DirWalk(n.host, pluginPath); err != nil {
		return err
	}
	if len(o) == 0 && plugin == "" {
		return fmt.Errorf("plugin & pluginPath is nil, skip render, you can quit now")
	} else if len(o) != 0 {
		printPlugins(string(o))
	} else {
		log.Logger.Infof("%s is nil, start to set %s.", pluginPath, plugin)
		target := fmt.Sprintf("%s@%s:%s", s.User, n.host, pluginPath)
		if o, err = s.Transfer(plugin, target); err != nil {
			return err
		}
		if _, err = s.UnZip(n.host, pluginSelf, pluginPath); err != nil {
			return err
		}
		if _, err = s.Restart("grafana"); err != nil {
			return err
		}
		log.Logger.Infof("%s -> %s complete.", plugin, pluginPath)
	}
	return nil
}

func (n *node) render(to string) error {
	if err := n.preCheck(); err != nil {
		return err
	}
	tok, err := n.newToken()
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
	source := filepath.Join(n.deployPath, "data", "png", "*")
	for _, p := range pls {
		c := fmt.Sprintf(uri, n.host, n.port, p.Org, s.ClusterName, p.Tab, from, now, p.ID) + "&tz=Asia%2FShanghai"
		kv := make(map[string]string)
		kv["Authorization"] = fmt.Sprintf("Bearer %s", tok.Key)
		out, err := http.NewRequestDo(c, http.MethodGet, nil, kv, "")
		if err != nil {
			return fmt.Errorf("%s failed: %v: %s", c, err, string(out))
		}
		sourcePath := fmt.Sprintf("%s@%s:%s", s.User, n.host, source)
		if _, err = s.Transfer(sourcePath, to); err != nil {
			return err
		}
		if _, err = s.Remove(n.host, source); err != nil {
			return err
		}
		log.Logger.Infof("[RENDER] %s", strings.ToUpper(p.Name))
	}
	return n.cleanToken(tok.Key)
}

type token struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
	Role string `json:"role"`
	Key  string `json:"key"`
}

func (n *node) newToken() (*token, error) {
	url := fmt.Sprintf("http://%s:%s/api/auth/keys", n.host, n.port)
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

func (n *node) cleanToken(key string) error {
	url := fmt.Sprintf("http://%s:%s/api/auth/keys", n.host, n.port)
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
		if err := n.dropToken(strconv.Itoa(t.ID)); err != nil {
			return err
		}
	}
	return nil
}

func (n *node) dropToken(id string) error {
	url := fmt.Sprintf("http://%s:%s/api/auth/keys/%s", n.host, n.port, id)
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

func getLabelKey(nodes map[string][]node) map[string]bool {
	labelKey := make(map[string]bool)
	for _, s := range nodes[getNodeTp(tikv)] {
		for k, _ := range s.labels {
			labelKey[k] = true
		}
	}
	return labelKey
}

func (n *node) isPdLeader(addr string) string {
	if n.tp == pdL {
		addr += "(L)"
	}
	return addr
}

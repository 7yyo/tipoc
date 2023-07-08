package comp

import (
	"embed"
	"encoding/json"
	"fmt"
	"gopkg.in/yaml.v2"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"pictorial/etcd"
	"pictorial/http"
	"pictorial/log"
	"pictorial/ssh"
	"pictorial/util/file"
	"strconv"
	"strings"
	"time"
)

type G struct {
	Host       string `json:"ip"`
	Port       int    `json:"port"`
	DeployPath string `json:"deploy_path"`
}

//go:embed "resource/*"
var RenderPlugin embed.FS

func (m *Mapping) GetGrafana() error {
	rs, err := etcd.GetByPrefix(PdAddr, topologyGrafana)
	if err != nil {
		return err
	}
	var g *G
	if err := json.Unmarshal(rs.Kvs[0].Value, &g); err != nil {
		return err
	}
	c := Component{
		Host:       g.Host,
		Port:       strconv.Itoa(g.Port),
		DeployPath: g.DeployPath,
	}
	m.Map[Grafana] = append(m.Map[Grafana], c)
	return nil
}

const pluginName = "plugin-linux-x64-glibc"

var zipIdx = [6]string{"a", "b", "c", "d", "e", "f"}

func (c *Component) installPlugin() error {
	s := ssh.S
	zipPackage := fmt.Sprintf("%s.zip", pluginName)
	pluginPath := filepath.Join(c.DeployPath, "plugins")
	defer func() {
		if _, err := s.Restart("grafana"); err != nil {
			log.Logger.Warnf("restart grafana failed: %s", err.Error())
		}
	}()
	ls, err := s.DirWalk(c.Host, pluginPath)
	if err != nil {
		return err
	}
	if strings.Contains(string(ls), pluginName) {
		log.Logger.Infof("grafana image render installed.")
		return nil
	}
	mergeZip, err := os.Create(zipPackage)
	if err != nil {
		return err
	}
	var f fs.File
	defer func() {
		if err := os.RemoveAll(pluginName); err != nil {
			log.Logger.Error(err)
		}
		if err := os.Remove(zipPackage); err != nil {
			log.Logger.Error(err)
		}
		mergeZip.Close()
		f.Close()
	}()
	for _, idx := range zipIdx {
		split := fmt.Sprintf("resource/%s.zip.a%s", pluginName, idx)
		f, err = RenderPlugin.Open(split)
		log.Logger.Infof("copy %s -> %s", split, mergeZip.Name())
		if _, err = io.Copy(mergeZip, f); err != nil {
			return err
		}
	}
	if err := file.UnzipPackage(mergeZip.Name(), "./"); err != nil {
		return err
	}
	target := fmt.Sprintf("%s@%s:%s", s.User, c.Host, pluginPath)
	log.Logger.Infof("%s -> %s", pluginName, target)
	if _, err := s.TransferR(pluginName, target); err != nil {
		return err
	}
	if err := c.dependencies(); err != nil {
		return err
	}
	return nil
}

func (c *Component) Render(to string, oType string) error {
	log.Logger.Info("start grafana image render...")
	if err := c.installPlugin(); err != nil {
		return err
	}
	tok, err := c.newToken()
	if err != nil {
		return err
	}
	now, from := unixDuration()
	pls, err := getPanels(oType)
	if err != nil {
		return err
	}
	s := ssh.S
	uri := "http://%s:%s/render/d-solo/%s/%s-%s?orgId=1&from=%s&to=%s&panelId=%s&width=1000&height=500&scale=3"
	source := filepath.Join(c.DeployPath, "data", "png", "*")
	dataPath := filepath.Join(c.DeployPath, "data", "png")
	if _, err := s.Remove(c.Host, source); err != nil {
		return err
	}
	time.Sleep(3 * time.Second)
	for _, p := range pls {
		cmd := fmt.Sprintf(uri, c.Host, c.Port, p.Org, s.Cluster.Name, p.Tab, from, now, p.ID) + "&tz=Asia%2FShanghai"
		kv := map[string]string{
			"Authorization": fmt.Sprintf("Bearer %s", tok.Key),
		}
		out, err := http.NewRequestDo(cmd, http.MethodGet, nil, kv, "")
		if err != nil {
			return fmt.Errorf("%s failed: %v: %s", c, err, string(out))
		}
		out, err = ssh.S.DirWalk(c.Host, dataPath)
		if err != nil {
			return err
		}
		if len(out) == 0 {
			return fmt.Errorf("render failed, 'grep 'eror' %s/log/grafana.log', skip: %s", p.Name, c.DeployPath)
		}
		sourcePath := fmt.Sprintf("%s@%s:%s", s.User, c.Host, source)
		if _, err = s.Mv(c.Host, source+".png", filepath.Join(dataPath, fmt.Sprintf("%s.png", p.Name))); err != nil {
			return err
		}
		if _, err = s.Transfer(sourcePath, to); err != nil {
			return err
		}
		if _, err = s.Remove(c.Host, source); err != nil {
			return err
		}
		log.Logger.Infof("[render] %s", p.Name)
	}
	return c.cleanToken(tok.Key)
}

type token struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
	Role string `json:"role"`
	Key  string `json:"key"`
}

func (c *Component) newToken() (*token, error) {
	url := fmt.Sprintf("http://%s:%s/api/auth/keys", c.Host, c.Port)
	payload := fmt.Sprintf(`{"name":"%s", "role":"Admin"}`, log.DateFormat())
	auth := http.Auth{
		Username: "admin",
		Password: "admin",
	}
	kv := map[string]string{
		"Content-Type": "application/json",
	}
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

func (c *Component) cleanToken(key string) error {
	url := fmt.Sprintf("http://%s:%s/api/auth/keys", c.Host, c.Port)
	kv := map[string]string{
		"Authorization": fmt.Sprintf("Bearer %s", key),
	}
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

func (c *Component) dropToken(id string) error {
	url := fmt.Sprintf("http://%s:%s/api/auth/keys/%s", c.Host, c.Port, id)
	auth := http.Auth{
		Username: "admin",
		Password: "admin",
	}
	kv := map[string]string{
		"Content-Type": "application/json",
	}
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
	log.Logger.Infof("time_horizon: %s ~ %s", timeFormat(ot), timeFormat(t))
	return strconv.Itoa(int(now)), strconv.Itoa(int(from))
}

func timeFormat(t time.Time) string {
	return t.Format("2006-01-02 15:04:05.000")
}

var dependencies = [2]string{
	"libatk-bridge*",
	"libxkbcommon*",
}

func (c *Component) dependencies() error {
	log.Logger.Infof("check for dependencies %v, maybe take a while...", dependencies)
	for _, d := range dependencies {
		if _, err := ssh.S.YumInstall(c.Host, d); err != nil {
			log.Logger.Warn(err)
		}
	}
	return nil
}

//go:embed "panel.yaml"
var panelPath embed.FS

const panelYaml = "panel.yaml"

type panel struct {
	ID   string
	Tab  string
	Name string
	Org  string
}

func getPanels(oType string) (map[string]panel, error) {
	p, err := panelPath.ReadFile(panelYaml)
	if err != nil {
		return nil, err
	}
	pls := make(map[string]panel)
	if err = yaml.Unmarshal(p, &pls); err != nil {
		return nil, err
	}
	newPls := make(map[string]panel)
	switch oType {
	case "data_distribution":
		newPls["region"] = getTargetPanel(pls, "region")
		newPls["store_size"] = getTargetPanel(pls, "store_size")
		newPls["leader"] = getTargetPanel(pls, "leader")
	case "disk_full":
		newPls["io_util"] = getTargetPanel(pls, "io_util")
		newPls["duration"] = getTargetPanel(pls, "duration")
		newPls["qps"] = getTargetPanel(pls, "qps")
		newPls["pd_uptime"] = getTargetPanel(pls, "pd_uptime")
		newPls["tikv_uptime"] = getTargetPanel(pls, "tikv_uptime")
	case "online_ddl_add_index":
		newPls["duration"] = getTargetPanel(pls, "duration")
		newPls["qps"] = getTargetPanel(pls, "qps")
	default:
		newPls["duration"] = getTargetPanel(pls, "duration")
		newPls["qps"] = getTargetPanel(pls, "qps")
		newPls["tidb_uptime"] = getTargetPanel(pls, "tidb_uptime")
		newPls["pd_uptime"] = getTargetPanel(pls, "pd_uptime")
		newPls["tikv_uptime"] = getTargetPanel(pls, "tikv_uptime")
	}
	return newPls, nil
}

func getTargetPanel(m map[string]panel, k string) panel {
	return m[k]
}

package comp

import (
	"embed"
	"encoding/json"
	"fmt"
	"gopkg.in/yaml.v2"
	"path/filepath"
	"pictorial/etcd"
	"pictorial/http"
	"pictorial/log"
	"pictorial/ssh"
	"strconv"
	"time"
)

type G struct {
	Host       string `json:"ip"`
	Port       int    `json:"port"`
	DeployPath string `json:"deploy_path"`
}

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

func (c *Component) preCheck() error {

	var o []byte
	var err error
	s := ssh.S
	plugin := s.Cluster.Plugin
	pluginName := filepath.Base(plugin)
	pluginPath := filepath.Join(c.DeployPath, "plugins")
	pluginSelf := filepath.Join(pluginPath, pluginName)

	if o, err = s.DirWalk(c.Host, pluginPath); err != nil {
		return err
	}
	if len(o) == 0 && plugin == "" {
		return fmt.Errorf("plugin & pluginPath is nil, skip render, you can quit now")
	} else if len(o) != 0 {
		printPlugins(string(o))
	} else {
		log.Logger.Infof("%s is nil, start to set %s.", pluginPath, plugin)
		target := fmt.Sprintf("%s@%s:%s", s.User, c.Host, pluginPath)
		if o, err = s.Transfer(plugin, target); err != nil {
			return err
		}
		if _, err = s.UnZip(c.Host, pluginSelf, pluginPath); err != nil {
			return err
		}
		log.Logger.Infof("%s -> %s complete.", plugin, pluginPath)
	}
	if _, err = s.Restart("grafana"); err != nil {
		return err
	}
	if err := c.dependencies(); err != nil {
		return err
	}
	return nil
}

func (c *Component) Render(to string) error {
	if err := c.preCheck(); err != nil {
		return err
	}
	tok, err := c.newToken()
	if err != nil {
		return err
	}
	now, from := unixDuration()
	pls, err := GetPanels()
	log.Logger.Infof("[render] count: %d", len(pls))
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
	payload := fmt.Sprintf(`{"name":"%s", "role":"Admin"}`, dateFormat())
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

func printPlugins(o string) {
	log.Logger.Info("maybe plugins installed:")
	log.Logger.Info(o)
}

func dateFormat() string {
	now := time.Now()
	year, month, day := now.Date()
	hour, min, sec := now.Clock()
	return fmt.Sprintf("%d-%02d-%02d_%02d:%02d:%02d", year, int(month), day, hour, min, sec)
}

var dependencies = [2]string{
	"libatk-bridge*",
	"libxkbcommon*",
}

func (c *Component) dependencies() error {
	log.Logger.Infof("check for dependencies %v, maybe take a while...", dependencies)
	for _, d := range dependencies {
		if _, err := ssh.S.RunSSH(c.Host, fmt.Sprintf("sudo yum install -y %s", d)); err != nil {
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

func GetPanels() (map[string]panel, error) {
	p, err := panelPath.ReadFile(panelYaml)
	if err != nil {
		return nil, err
	}
	pls := make(map[string]panel)
	if err = yaml.Unmarshal(p, &pls); err != nil {
		return nil, err
	}
	return pls, nil
}

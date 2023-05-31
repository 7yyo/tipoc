package panel

import (
	"embed"
	"gopkg.in/yaml.v2"
)

//go:embed "panel.yaml"
var P embed.FS

const panelYaml = "panel.yaml"

type panel struct {
	ID   string
	Tab  string
	Name string
	Org  string
}

func GetPanels() (map[string]panel, error) {
	p, err := P.ReadFile(panelYaml)
	if err != nil {
		return nil, err
	}
	pls := make(map[string]panel)
	if err = yaml.Unmarshal(p, &pls); err != nil {
		return nil, err
	}
	return pls, nil
}

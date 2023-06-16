package operator

import (
	"fmt"
	"pictorial/log"
	"pictorial/ssh"
	"strings"
)

type dataCorruptedOperator struct {
	host       string
	port       string
	cType      string
	deployPath string
}

func (d *dataCorruptedOperator) Execute() error {
	var dataPath string
	var err error
	switch d.cType {
	case "tikv":
		dataPath, err = d.getTiKVDataPath()
	case "pd":
		dataPath, err = d.getPdDataPath()
	default:
		err = fmt.Errorf("only support: tikv, pd")
	}
	if err != nil {
		return err
	}
	filename := fmt.Sprintf("%s_bak", dataPath)
	cmd := fmt.Sprintf("mv %s %s", dataPath, filename)
	if _, err = ssh.S.RunSSH(d.host, cmd); err != nil {
		return err
	}
	log.Logger.Infof("rename [%s] to [%s].", dataPath, filename)
	return nil
}

func (d *dataCorruptedOperator) getTiKVDataPath() (string, error) {
	deployPath := strings.Replace(d.deployPath, "/bin", "", -1)
	script := fmt.Sprintf("%s/scripts/run_tikv.sh", deployPath)
	o, err := ssh.S.RunSSH(d.host, fmt.Sprintf("grep -oP -- '--data-dir \\K[^\\n:]+' %s | tr -d ' '", script))
	if err != nil {
		return "", err
	}
	dataPath := string(o)
	dataPath = processDataPath(dataPath)
	return dataPath, nil
}

func (d *dataCorruptedOperator) getPdDataPath() (string, error) {
	deployPath := strings.TrimSuffix(d.deployPath, "/bin")
	script := fmt.Sprintf("%s/scripts/run_pd.sh", deployPath)
	o, err := ssh.S.RunSSH(d.host, fmt.Sprintf("grep -oP -- '--data-dir=\\K[^\\s]*' %s", script))
	if err != nil {
		return "", err
	}
	dataPath := string(o)
	dataPath = processDataPath(dataPath)
	return dataPath, nil
}

func processDataPath(dp string) string {
	dp = strings.ReplaceAll(dp, "\"", "")
	dp = strings.ReplaceAll(dp, "\n", "")
	dp = strings.TrimSuffix(dp, "\\")
	return dp
}

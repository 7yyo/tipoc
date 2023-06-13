package operator

import (
	"fmt"
	"net"
	"path/filepath"
	"pictorial/log"
	"pictorial/ssh"
	"strings"
)

type crashOperator struct {
	host        string
	port        string
	componentTp string
}

const systemdPath = "/etc/systemd/system/"
const alwaysToNo = "sudo sed -i 's/always/no/g' %s"
const reloadSystemd = "sudo systemctl daemon-reload"
const service = "%s-%s.service"

func (c *crashOperator) Execute() error {
	nodeTp := strings.Replace(c.componentTp, "(L)", "", -1)
	systemd := fmt.Sprintf(service, nodeTp, c.port)
	service := filepath.Join(systemdPath, systemd)
	cmd := fmt.Sprintf(alwaysToNo, service)
	if _, err := ssh.S.RunSSH(c.host, cmd); err != nil {
		return err
	}
	if _, err := ssh.S.RunSSH(c.host, reloadSystemd); err != nil {
		return err
	}

	addr := net.JoinHostPort(c.host, c.port)
	out, _ := ssh.S.GetProcessIDByPort(c.host, c.port)
	processID := string(out)
	if len(processID) == 0 {
		log.Logger.Warnf("[crash] [%s] %s is offline, skip.", nodeTp, addr)
		return nil
	}
	log.Logger.Infof("[crash] [%s] [%s] - %v", nodeTp, addr, processID)
	o, err := ssh.S.RunSSH(c.host, fmt.Sprintf("kill -9 %s", processID))
	if err != nil {
		log.Logger.Warnf("[crash] [%s] %s {%s} failed: %v: %s", nodeTp, addr, processID, err, string(o))
	}
	return nil
}

package operator

import (
	"fmt"
	"net"
	"path/filepath"
	"pictorial/comp"
	"pictorial/log"
	"pictorial/ssh"
)

type crashOperator struct {
	host       string
	port       string
	cType      string
	deployPath string
}

const systemdPath = "/etc/systemd/system/"
const service = "%s-%s.service"

func (c *crashOperator) Execute() error {
	nodeTp := comp.CleanLeaderFlag(c.cType)
	if c.cType == "tiflash" {
		port, err := comp.GetTiFlashPort(c.host, c.deployPath)
		if err != nil {
			return err
		}
		c.port = port
	}
	systemd := fmt.Sprintf(service, nodeTp, c.port)
	service := filepath.Join(systemdPath, systemd)
	if _, err := ssh.S.Systemd(c.host, ssh.No, service); err != nil {
		return err
	}
	addr := net.JoinHostPort(c.host, c.port)
	processID, err := ssh.S.GetProcessIDByPort(c.host, c.port)
	if err != nil {
		return err
	}
	if processID == "" {
		log.Logger.Warnf("[crash] [%s] %s is offline, skip.", nodeTp, addr)
		return nil
	}
	log.Logger.Infof("[crash] [%s] [%s] - %v", nodeTp, addr, processID)
	o, err := ssh.S.Kill9(c.host, processID)
	if err != nil {
		log.Logger.Warnf("[crash] [%s] %s {%s} failed: %v: %s", nodeTp, addr, processID, err, string(o))
	}
	return nil
}

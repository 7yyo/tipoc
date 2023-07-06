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
	cType      comp.CType
	deployPath string
}

const systemdPath = "/etc/systemd/system/"
const serviceFile = "%s-%s.service"
const crash = "crash"

func (c *crashOperator) Execute() error {
	cType := comp.GetCTypeValue(c.cType)
	cType = comp.CleanLeaderFlag(cType)
	if cType == "tiflash" {
		port, err := comp.GetTiFlashPort(c.host, c.deployPath)
		if err != nil {
			return err
		}
		c.port = port
	}
	systemd := fmt.Sprintf(serviceFile, cType, c.port)
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
		log.Logger.Warnf("[%s] [%s] %s is offline, skip.", crash, cType, addr)
		return nil
	}
	log.Logger.Infof("[%s] [%s] [%s] - %v", crash, cType, addr, processID)
	if _, err = ssh.S.Kill9(c.host, processID); err != nil {
		log.Logger.Error(err)
	}
	return nil
}

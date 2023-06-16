package operator

import (
	"fmt"
	"net"
	"path/filepath"
	"pictorial/comp"
	"pictorial/log"
	"pictorial/ssh"
)

type recoverSystemdOperator struct {
	host       string
	port       string
	cType      string
	deployPath string
}

const noToAlways = "sudo sed -i 's/no/always/g' %s"

func (r *recoverSystemdOperator) Execute() error {
	var systemd string
	nodeTp := comp.CleanLeaderFlag(r.cType)
	if r.cType == "tiflash" {
		port, err := comp.GetTiFlashPort(r.host, r.deployPath)
		if err != nil {
			return err
		}
		r.port = port
	}
	systemd = fmt.Sprintf(service, nodeTp, r.port)
	service := filepath.Join(systemdPath, systemd)
	if _, err := ssh.S.Systemd(r.host, ssh.Always, service); err != nil {
		return err
	}
	addr := net.JoinHostPort(r.host, r.port)
	log.Logger.Infof("[recover_systemd] [%s] %s", nodeTp, addr)
	return nil
}

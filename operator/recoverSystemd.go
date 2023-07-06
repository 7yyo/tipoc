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
	cType      comp.CType
	deployPath string
}

func (r *recoverSystemdOperator) Execute() error {
	var systemd string
	co := comp.GetCTypeValue(r.cType)
	nodeTp := comp.CleanLeaderFlag(co)
	if co == "tiflash" {
		port, err := comp.GetTiFlashPort(r.host, r.deployPath)
		if err != nil {
			return err
		}
		r.port = port
	}
	systemd = fmt.Sprintf(serviceFile, nodeTp, r.port)
	service := filepath.Join(systemdPath, systemd)
	if _, err := ssh.S.Systemd(r.host, ssh.Always, service); err != nil {
		return err
	}
	addr := net.JoinHostPort(r.host, r.port)
	log.Logger.Infof("[recover_systemd] [%s] %s", nodeTp, addr)
	return nil
}

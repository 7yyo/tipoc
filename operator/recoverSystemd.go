package operator

import (
	"fmt"
	"net"
	"path/filepath"
	"pictorial/log"
	"pictorial/ssh"
	"strings"
)

type recoverSystemdOperator struct {
	host        string
	port        string
	componentTp string
}

const noToAlways = "sudo sed -i 's/no/always/g' %s"

func (r *recoverSystemdOperator) Execute() error {
	var systemd string
	nodeTp := strings.Replace(r.componentTp, "(L)", "", -1)
	systemd = fmt.Sprintf(service, nodeTp, r.port)
	service := filepath.Join(systemdPath, systemd)
	cmd := fmt.Sprintf(noToAlways, service)
	if _, err := ssh.S.RunSSH(r.host, cmd); err != nil {
		return err
	}
	if _, err := ssh.S.RunSSH(r.host, reloadSystemd); err != nil {
		return err
	}
	addr := net.JoinHostPort(r.host, r.port)
	log.Logger.Infof("[RECOVER_SYSTEMD] [%s] %s", nodeTp, addr)
	return nil
}

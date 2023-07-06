package operator

import (
	"pictorial/log"
	"pictorial/ssh"
)

type rebootOperator struct {
	host string
}

const reboot = "reboot"
const rebootCmd = "sudo reboot"

func (r *rebootOperator) Execute() error {
	if _, err := ssh.S.RunSSH(r.host, rebootCmd); err != nil {
		return err
	}
	log.Logger.Infof("[%s] %s", reboot, r.host)
	return nil
}

package operator

import (
	"pictorial/log"
	"pictorial/ssh"
)

type rebootOperator struct {
	host string
}

func (r *rebootOperator) Execute() error {
	if _, err := ssh.S.RunSSH(r.host, "sudo reboot"); err != nil {
		return err
	}
	log.Logger.Infof("[reboot] %s", r.host)
	return nil
}

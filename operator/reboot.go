package operator

import (
	"pictorial/ssh"
)

type rebootOperator struct {
	host string
}

func (r *rebootOperator) Execute() error {
	if _, err := ssh.S.RunSSH(r.host, "sudo reboot"); err != nil {
		return err
	}
	return nil
}

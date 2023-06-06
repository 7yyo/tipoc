package operator

import (
	"pictorial/ssh"
)

type rebootOperator struct {
	host string
}

func (r *rebootOperator) Execute() error {
	_, err := ssh.S.RunSSH(r.host, "sudo reboot")
	if err != nil {
		return err
	}
	return nil
}

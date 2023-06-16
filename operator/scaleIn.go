package operator

import (
	"net"
	"pictorial/log"
	"pictorial/ssh"
)

type scaleInOperator struct {
	host        string
	port        string
	clusterName string
}

func (s *scaleInOperator) Execute() error {
	addr := net.JoinHostPort(s.host, s.port)
	if _, err := ssh.S.ScaleIn(addr); err != nil {
		return err
	}
	log.Logger.Infof("[scale_in] %s", addr)
	return nil
}

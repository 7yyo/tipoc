package operator

import (
	"fmt"
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
	c := fmt.Sprintf("tiup cluster scale-in %s -N %s --yes", s.clusterName, addr)
	if _, err := ssh.RunLocal(c); err != nil {
		return err
	}
	log.Logger.Infof("[SCALE] %s", addr)
	return nil
}

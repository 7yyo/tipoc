package operator

import (
	"net"
	"pictorial/comp"
	"pictorial/log"
	"pictorial/ssh"
)

type scaleInOperator struct {
	host        string
	port        string
	clusterName string
	cType       comp.CType
	deployPath  string
}

func (s *scaleInOperator) Execute() error {
	addr := net.JoinHostPort(s.host, s.port)
	co := comp.GetCTypeValue(s.cType)
	if co == "tiflash" {
		port, err := comp.GetTiFlashPort(s.host, s.deployPath)
		if err != nil {
			return err
		}
		addr = net.JoinHostPort(s.host, port)
	}
	log.Logger.Infof("[scale-in] [%s] %s ...", s.cType, addr)
	if _, err := ssh.S.ScaleIn(addr); err != nil {
		return err
	}
	log.Logger.Infof("[scale-in] [%s] %s complete", s.cType, addr)
	return nil
}

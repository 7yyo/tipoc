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

const scaleIn = "scale_in"

func (s *scaleInOperator) Execute() error {
	cType := comp.GetCTypeValue(s.cType)
	addr := net.JoinHostPort(s.host, s.port)
	if s.cType == comp.TiFlash {
		port, err := comp.GetTiFlashPort(s.host, s.deployPath)
		if err != nil {
			return err
		}
		addr = net.JoinHostPort(s.host, port)
	}
	log.Logger.Infof("[%s] [%s] %s ...", scaleIn, cType, addr)
	if _, err := ssh.S.ScaleIn(addr); err != nil {
		return err
	}
	log.Logger.Infof("[%s] [%s] %s complete", scaleIn, cType, addr)
	return nil
}

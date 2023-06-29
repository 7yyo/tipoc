package operator

import (
	"net"
	"pictorial/comp"
	"pictorial/log"
	"pictorial/ssh"
)

type killOperator struct {
	host  string
	port  string
	cType comp.CType
}

func (k *killOperator) Execute() error {
	addr := net.JoinHostPort(k.host, k.port)
	c := comp.GetCTypeValue(k.cType)
	processID, _ := ssh.S.GetProcessIDByPort(k.host, k.port)
	if processID == "" {
		log.Logger.Warnf("[kill] [%s] %s is offline, skip.", c, addr)
		return nil
	}
	log.Logger.Infof("[kill] [%s] [%s] - %s", c, addr, processID)
	o, err := ssh.S.Kill9(k.host, processID)
	if err != nil {
		log.Logger.Warnf("[kill] [%s] %s {%s} failed: %v: %s", c, addr, processID, err, string(o))
	}
	return nil
}

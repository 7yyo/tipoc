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

const kill = "kill"

func (k *killOperator) Execute() error {
	addr := net.JoinHostPort(k.host, k.port)
	cType := comp.GetCTypeValue(k.cType)
	processID, _ := ssh.S.GetProcessIDByPort(k.host, k.port)
	if processID == "" {
		log.Logger.Warnf("[%s] [%s] %s maybe offline, skip.", kill, cType, addr)
		return nil
	}
	log.Logger.Infof("[%s] [%s] [%s] - %s", kill, cType, addr, processID)
	o, err := ssh.S.Kill9(k.host, processID)
	if err != nil {
		log.Logger.Warnf("[%s] [%s] %s {%s} failed: %v: %s", kill, cType, addr, processID, err, string(o))
	}
	return nil
}

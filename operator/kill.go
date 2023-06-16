package operator

import (
	"net"
	"pictorial/log"
	"pictorial/ssh"
)

type killOperator struct {
	host        string
	port        string
	componentTp string
}

func (k *killOperator) Execute() error {
	addr := net.JoinHostPort(k.host, k.port)
	processID, _ := ssh.S.GetProcessIDByPort(k.host, k.port)
	if processID == "" {
		log.Logger.Warnf("[kill] [%s] %s is offline, skip.", k.componentTp, addr)
		return nil
	}
	log.Logger.Infof("[kill] [%s] [%s] - %s", k.componentTp, addr, processID)
	o, err := ssh.S.Kill9(k.host, processID)
	if err != nil {
		log.Logger.Warnf("[kill] [%s] %s {%s} failed: %v: %s", k.componentTp, addr, processID, err, string(o))
	}
	return nil
}

package operator

import (
	"fmt"
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
	out, _ := ssh.S.GetProcessIDByPort(k.host, k.port)
	processID := string(out)
	if len(processID) == 0 {
		log.Logger.Warnf("[kill] [%s] %s is offline, skip.", k.componentTp, addr)
		return nil
	}
	log.Logger.Infof("[kill] [%s] [%s] -%s", k.componentTp, addr, processID)
	o, err := ssh.S.RunSSH(k.host, fmt.Sprintf("kill -9 %s", processID))
	if err != nil {
		log.Logger.Warnf("[kill] [%s] %s {%s} failed: %v: %s", k.componentTp, addr, processID, err, string(o))
	}
	return nil
}

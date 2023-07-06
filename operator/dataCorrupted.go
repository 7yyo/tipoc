package operator

import (
	"fmt"
	"net"
	"pictorial/comp"
	"pictorial/log"
	"pictorial/ssh"
)

const dataCorrupted = "data_corrupted"

type dataCorruptedOperator struct {
	host       string
	port       string
	cType      comp.CType
	deployPath string
}

func (d *dataCorruptedOperator) Execute() error {
	dataPath, err := comp.GetDataPath(d.host, d.deployPath, d.cType)
	cType := comp.GetCTypeValue(d.cType)
	if err != nil {
		return err
	}
	bakName := fmt.Sprintf("%s_bak", dataPath)
	if _, err := ssh.S.Mv(d.host, dataPath, bakName); err != nil {
		return err
	}
	addr := net.JoinHostPort(d.host, d.port)
	log.Logger.Infof("[%s] [%s] [%s] [%s] to [%s].", dataCorrupted, cType, addr, dataPath, bakName)
	return nil
}

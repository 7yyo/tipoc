package operator

import (
	"fmt"
	"net"
	"pictorial/comp"
	"pictorial/log"
	"pictorial/ssh"
)

type dataCorruptedOperator struct {
	host       string
	port       string
	cType      comp.CType
	deployPath string
}

func (d *dataCorruptedOperator) Execute() error {
	var dataPath string
	var err error
	switch d.cType {
	case comp.TiKV:
		dataPath, err = comp.GetDataPath(d.host, d.deployPath, comp.TiKV)
	case comp.PD:
		dataPath, err = comp.GetDataPath(d.host, d.deployPath, comp.PD)
	default:
		err = fmt.Errorf("only support: tikv, pd")
	}
	if err != nil {
		return err
	}
	sprintf := fmt.Sprintf("%s_bak", dataPath)
	cmd := fmt.Sprintf("mv %s %s", dataPath, sprintf)
	if _, err = ssh.S.RunSSH(d.host, cmd); err != nil {
		return err
	}
	addr := net.JoinHostPort(d.host, d.port)
	log.Logger.Infof("[%s] [%s] [%s] [%s] to [%s].", "data_corrupted", comp.GetCTypeValue(d.cType), addr, dataPath, sprintf)
	return nil
}

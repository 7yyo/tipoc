package operator

import (
	"fmt"
	"net"
	"path/filepath"
	"pictorial/comp"
	"pictorial/log"
	"pictorial/ssh"
)

type diskFullOperator struct {
	host       string
	port       string
	cType      comp.CType
	deployPath string
	stopC      chan bool
}

const fio = "fio"

const (
	threads = "8"
	size    = "10000G"
)

const c = "fio -threads=%s -size=%s -bs=1m -direct=1 -rw=write -name=tipp -filename=%s -continue_on_error=1"

func (d *diskFullOperator) Execute() error {
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
	dataPath = filepath.Join(dataPath, "disk_full")
	log.Logger.Infof("[disk_full] [%s] [%s] [%s]", comp.GetCTypeValue(d.cType), net.JoinHostPort(d.host, d.port), dataPath)
	if err != nil {
		return err
	}
	if _, err := ssh.S.YumInstall(d.host, fio); err != nil {
		return err
	}
	cmd := fmt.Sprintf(c, threads, size, dataPath)
	go func() {
		if _, err := ssh.S.RunSSH(d.host, cmd); err != nil {
			log.Logger.Error(err)
		}
	}()
	go func() {
		<-d.stopC
		if err := d.aftercare(dataPath); err != nil {
			log.Logger.Errorf("[disk_full] aftercare failed: %s", err.Error())
		}
	}()
	return nil
}

func (d *diskFullOperator) aftercare(dataPath string) error {
	pids, err := ssh.S.GetProcessIDByPs(d.host, fio)
	if err != nil {
		return err
	}
	for _, pid := range pids {
		if _, err := ssh.S.Kill9(d.host, pid); err != nil {
			return err
		}
	}
	log.Logger.Infof("[disk_full] killed fio %v, removed [%s]", pids, dataPath)
	return nil
}

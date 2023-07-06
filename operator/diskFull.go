package operator

import (
	"context"
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
	ctx        context.Context
}

const diskFull = "disk_full"
const fio = "fio"
const fioCmd = "fio -threads=%s -size=%s -bs=1m -direct=1 -rw=write -name=tipp -filename=%s -continue_on_error=1"

const (
	threads = "8"
	size    = "10000G"
)

func (d *diskFullOperator) Execute() error {
	cType := comp.GetCTypeValue(d.cType)
	dataPath, err := comp.GetDataPath(d.host, d.deployPath, d.cType)
	if err != nil {
		return err
	}
	dataPath = filepath.Join(dataPath, "disk_full")
	log.Logger.Infof("[%s] [%s] [%s] [%s]", diskFull, cType, net.JoinHostPort(d.host, d.port), dataPath)
	if err != nil {
		return err
	}
	if _, err := ssh.S.YumInstall(d.host, fio); err != nil {
		return err
	}
	cmd := fmt.Sprintf(fioCmd, threads, size, dataPath)
	go func() {
		if _, err := ssh.S.RunSSH(d.host, cmd); err != nil {
			log.Logger.Error(err)
		}
	}()
	go func() {
		select {
		case <-d.ctx.Done():
			if err := d.aftercare(dataPath); err != nil {
				log.Logger.Errorf("[disk_full] aftercare failed: %s", err.Error())
			}
		}
	}()
	return nil
}

func (d *diskFullOperator) aftercare(dataPath string) error {
	ids, err := ssh.S.GetProcessIDByPs(d.host, fio)
	if err != nil {
		return err
	}
	for _, pid := range ids {
		log.Logger.Debugf("killed %s", pid)
		_, _ = ssh.S.Kill9(d.host, pid)
	}
	if _, err := ssh.S.Remove(d.host, dataPath); err != nil {
		return err
	}
	log.Logger.Infof("[disk_full] killed fio %v, removed [%s]", ids, dataPath)
	return nil
}

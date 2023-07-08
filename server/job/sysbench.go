package job

import (
	_ "embed"
	"fmt"
	"os"
	"pictorial/log"
	"pictorial/operator"
	"pictorial/ssh"
	"pictorial/util/file"
)

//go:embed "resource/1.0.15.tar.gz"
var sysBenchGz []byte

const gzName = "1.0.15.tar.gz"
const folderName = "sysbench-1.0.15"

const oltpInsert = "sysbench oltp_insert --db-driver=mysql --mysql-host=%s --mysql-port=%s --mysql-user=%s --mysql-password=%s --mysql-db=%s --table-size=%s --tables=%s --threads=%s %s"
const oltpWriteRead = "sysbench oltp_read_write --db-driver=mysql --mysql-host=%s --mysql-port=%s --mysql-user=%s --mysql-password=%s --mysql-db=%s --table-size=%s --tables=%s --report-interval=10 --threads=%s %s"

func (j *Job) runInstallSysBench() {
	ov := operator.GetOTypeValue(operator.InstallSysBench)
	defer func() {
		if err := os.RemoveAll(folderName); err != nil {
			log.Logger.Warn(err)
		}
	}()
	if _, err := ssh.S.RunLocal("sysbench"); err != nil {
		log.Logger.Infof("[%s] %s", ov, err.Error())
		log.Logger.Infof("[%s] unzip sysbench: %s", ov, gzName)
		if err := file.UnTar(sysBenchGz, "./"); err != nil {
			j.ErrC <- err
			return
		}
		log.Logger.Infof("[%s] unzip sysbench: %s complete", ov, gzName)
		dep := "sudo yum -y install make automake libtool pkgconfig libaio-devel mysql-devel openssl-devel"
		if _, err := ssh.S.RunLocal(dep); err != nil {
			j.ErrC <- err
			return
		}
		if _, err := ssh.S.RunLocal(fmt.Sprintf("cd %s/; ./autogen.sh; ./configure; make -j; sudo make install;", folderName)); err != nil {
			j.ErrC <- err
			return
		}
		if _, err := ssh.S.RunLocal("sysbench"); err != nil {
			log.Logger.Info("[%s] install sysbench failed.")
		} else {
			log.Logger.Infof("[%s] install sysbench complete.", ov)
		}
	} else {
		log.Logger.Infof("[%s] sysbench installed.", ov)
	}
}

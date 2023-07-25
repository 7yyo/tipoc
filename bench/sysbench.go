package bench

import (
	_ "embed"
	"fmt"
	"os"
	"pictorial/log"
	"pictorial/mysql"
	"pictorial/operator"
	"pictorial/ssh"
	"pictorial/util/file"
	"strings"
)

//go:embed "resource/sysbench-1.0.20.tar.gz"
var sysBenchGz []byte

const folderName = "sysbench-1.0.20"

const cmd = "sysbench ${test_type} --db-driver=mysql --mysql-host=%s --mysql-port=%s --mysql-user=%s --mysql-password=%s --mysql-db=%s --table-size=%d --tables=%d --threads=%d --report-interval=10 --time=1440 %s"

type TestTp int

const (
	OltpInsert TestTp = iota
	OltpReadWrite
)

func GetSysbenchTpValue(tp TestTp) string {
	switch tp {
	case OltpInsert:
		return "oltp_insert"
	case OltpReadWrite:
		return "oltp_read_write"
	}
	return ""
}

type Sysbench struct {
	Test      TestTp
	Db        string
	TableSize int
	Tables    int
	Threads   int
	Cmd       string
	Mysql     mysql.MySQL
}

var dependencies = []string{
	"make",
	"automake",
	"libtool",
	"pkgconfig",
	"libaio-devel",
	"mysql-devel",
	"openssl-devel",
}

func InstallSysBench() error {
	ov := operator.GetOTypeValue(operator.InstallSysBench)
	defer func() {
		if err := os.RemoveAll(folderName); err != nil {
			panic(err)
		}
	}()
	if _, err := TestSysbench(); err != nil {
		log.Logger.Infof("[%s] run sysbench failed: %s", ov, err.Error())
		log.Logger.Infof("[%s] unzip sysbench...", ov)

		if err := file.UnTar(sysBenchGz, "./"); err != nil {
			return err
		}
		log.Logger.Infof("[%s] unzip sysbench complete.", ov)

		for _, d := range dependencies {
			yumInstall := fmt.Sprintf("sudo yum -y install %s", d)
			if _, err := ssh.S.RunLocal(yumInstall); err != nil {
				return err
			}
		}

		installStep := fmt.Sprintf("cd %s/; ./autogen.sh; ./configure; make -j; sudo make install;", folderName)
		if _, err := ssh.S.RunLocal(installStep); err != nil {
			return err
		}

		if _, err := TestSysbench(); err != nil {
			log.Logger.Info("[%s] install sysbench failed.")
		} else {
			log.Logger.Infof("[%s] install sysbench complete.", ov)
		}
	} else {
		log.Logger.Infof("[%s] sysbench installed.", ov)
	}
	return nil
}

func TestSysbench() ([]byte, error) {
	return ssh.S.RunLocal("sysbench")
}

func (s *Sysbench) Run() ([]byte, error) {
	cmd := s.String()
	log.Logger.Infof("[sysbench] %s", cmd)
	return ssh.S.RunLocal(cmd)
}

func (s *Sysbench) String() string {
	cmd := fmt.Sprintf(cmd,
		s.Mysql.Host,
		s.Mysql.Port,
		s.Mysql.User,
		s.Mysql.Password,
		s.Db,
		s.TableSize,
		s.Tables,
		s.Threads,
		s.Cmd)
	switch s.Test {
	case OltpInsert:
		cmd = strings.Replace(cmd, "${test_type}", GetSysbenchTpValue(OltpInsert), -1)
	case OltpReadWrite:
		cmd = strings.Replace(cmd, "${test_type}", GetSysbenchTpValue(OltpReadWrite), -1)
	}
	return cmd
}

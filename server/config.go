package server

import (
	"flag"
	"fmt"
	"github.com/pelletier/go-toml"
	"github.com/sirupsen/logrus"
	"pictorial/comp"
	"pictorial/log"
	"pictorial/mysql"
	"pictorial/server/job"
	"pictorial/ssh"
	"pictorial/widget"
	"time"
)

const defaultCfg = "config.toml"

const (
	mysqlHost     = "mysql.host"
	mysqlPort     = "mysql.port"
	mysqlUser     = "mysql.user"
	mysqlPassword = "mysql.password"

	sshUser     = "ssh.user"
	sshPassword = "ssh.password"
	sshPort     = "ssh.sshPort"

	clusterName  = "cluster.name"
	loadCmd      = "load.cmd"
	loadInterval = "load.interval"
	loadSleep    = "load.sleep"
	logLevel     = "log.level"
	otherDir     = "other.dir"
)

var notNil = []string{
	mysqlHost, mysqlPort, mysqlUser, mysqlPassword,
	sshUser, sshPort,
	clusterName,
}

func parseFlag() (*toml.Tree, error) {
	var cfg string
	flag.StringVar(&cfg, "c", defaultCfg, "")
	flag.Parse()
	return toml.LoadFile(cfg)
}

func initConfig(cfg *toml.Tree) error {
	for _, c := range notNil {
		if cfg.Get(c) == nil {
			return fmt.Errorf("config [%s] must not be empty", c)
		}
	}
	for k, v := range cfg.Values() {
		log.Logger.Debugf("%s: %s", k, v)
	}

	mysql.M.Host = cfg.Get(mysqlHost).(string)
	mysql.M.Port = cfg.Get(mysqlPort).(string)
	mysql.M.User = cfg.Get(mysqlUser).(string)
	mysql.M.Password = cfg.Get(mysqlPassword).(string)

	var err error
	comp.PdAddr, err = comp.GetPdAddr()
	if err != nil {
		return err
	}

	ssh.S.User = cfg.Get(sshUser).(string)
	if cfg.Get(sshPassword) != nil {
		ssh.S.Password = cfg.Get(sshPassword).(string)
	}
	ssh.S.SshPort = cfg.Get(sshPort).(string)
	ssh.S.Cluster.Name = cfg.Get(clusterName).(string)
	ssh.S.LogC = make(chan string)
	if err := ssh.S.CheckClusterName(); err != nil {
		return err
	}
	if err := ssh.S.AddSSHKey(); err != nil {
		return err
	}
	if cfg.Get(loadCmd) != nil {
		job.Ld.Cmd = cfg.Get(loadCmd).(string)
	}
	if cfg.Get(loadInterval) != nil {
		job.Ld.Interval = cfg.Get(loadInterval).(int64)
	}
	if cfg.Get(loadSleep) != nil {
		job.Ld.Sleep = time.Duration(cfg.Get(loadSleep).(int64))
	}

	if cfg.Get(logLevel) != nil {
		logLevel := cfg.Get(logLevel).(string)
		switch logLevel {
		case "debug":
			log.Logger.SetLevel(logrus.DebugLevel)
		}
	}
	if cfg.Get(widget.OtherConfig) != nil {
		widget.OtherConfig = cfg.Get(otherDir).(string)
	}

	return nil
}

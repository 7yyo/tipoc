package server

import (
	"fmt"
	"github.com/pelletier/go-toml"
)

const (
	mysqlHost     = "mysql.host"
	mysqlPort     = "mysql.port"
	mysqlUser     = "mysql.user"
	mysqlPassword = "mysql.password"

	sshUser      = "ssh.user"
	sshPassword  = "ssh.password"
	sshPort      = "ssh.sshPort"
	clusterName  = "cluster.name"
	plugin       = "cluster.plugin"
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

func checkConfig(config *toml.Tree) error {
	for _, c := range notNil {
		if config.Get(c) == nil {
			return fmt.Errorf("config: %s must be not nil", c)
		}
	}
	return nil
}

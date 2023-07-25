package job

import (
	"path/filepath"
	"pictorial/comp"
	"pictorial/log"
	"pictorial/mysql"
	"pictorial/operator"
	"pictorial/ssh"
)

func (j *Job) runGeneralLogJob() error {
	ov := operator.GetOTypeValue(operator.GeneralLog)

	log.Logger.Infof("[%s] execute sql with general log.", ov)
	output, err := mysql.M.ExecuteForceWithOutput(""+
		"SET GLOBAL tidb_general_log = ON;"+
		"CREATE TABLE poc.test_general_log (id int PRIMARY KEY);"+
		"INSERT INTO poc.test_general_log VALUES (1), (2), (3);"+
		"SET GLOBAL tidb_general_log = OFF;",
		mysql.M.User, mysql.M.Password)
	if err != nil {
		return err
	}
	j.writeResultFile(ov, 1, 0, output)

	var deployPath string
	for _, tidb := range j.components[comp.TiDB] {
		if tidb.Host == mysql.M.Host && tidb.Port == mysql.M.Port {
			deployPath = tidb.DeployPath
		}
	}
	logPath := filepath.Join(deployPath, "log", "tidb.log")
	if _, err := ssh.S.GrepTailN(mysql.M.Host, logPath, 2); err != nil {
		return err
	}
	return nil
}

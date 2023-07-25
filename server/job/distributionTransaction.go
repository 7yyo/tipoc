package job

import (
	"fmt"
	"pictorial/log"
	"pictorial/mysql"
	"pictorial/operator"
	"time"
)

func (j *Job) runFlashbackCluster() error {
	ov := operator.GetOTypeValue(operator.FlashBackCluster)
	ts := log.Timestamp()
	log.Logger.Infof("[%s] timestamp: %s", ov, ts)
	time.Sleep(1 * time.Second)
	sql := fmt.Sprintf(
		"SELECT NOW();"+
			"CREATE TABLE poc.%s (id INT PRIMARY KEY, c1 INT);"+
			"INSERT INTO poc.%s VALUES (1, 100), (2, 200);"+
			"SELECT * FROM poc.%s;"+
			"FLASHBACK CLUSTER TO TIMESTAMP '%s';"+
			"SELECT * FROM poc.%s;",
		ov, ov, ov, ts, ov)
	time.Sleep(3 * time.Second)
	output, err := mysql.M.ExecuteForceWithOutput(sql, mysql.M.User, mysql.M.Password)
	j.writeResultFile(ov, 1, 0, output)
	if err != nil {
		return err
	}
	return nil
}

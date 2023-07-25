package job

import (
	"fmt"
	ms "github.com/go-mysql-org/go-mysql/mysql"
	"pictorial/bench"
	"pictorial/comp"
	"pictorial/log"
	"pictorial/mysql"
	"pictorial/operator"
	"strings"
	"time"
)

func (j *Job) runDataSeparation() error {
	ov := operator.GetOTypeValue(operator.DataSeparation)
	rs, err := mysql.M.ExecuteSQL(mysql.ShowPlacementLabels)
	if err != nil {
		return err
	}
	key, value := confirmLabels(rs.Values)
	if key == "" {
		return fmt.Errorf(fmt.Sprintf("[%s] no label instance more than 1, cancel.", ov))
	}
	values := processLabel(value)

	policyP1 := "p1"
	policyP2 := "p2"
	db := "poc"
	table1 := "sbtest1"
	table2 := "sbtest2"

	createPolicySQL := mysql.DropPlacementPolicy(policyP1) +
		mysql.DropPlacementPolicy(policyP2) +
		mysql.CreatePlacementPolicy(fmt.Sprintf("%s constraints='[+%s=%s]';", policyP1, key, values[0])) +
		mysql.CreatePlacementPolicy(fmt.Sprintf("%s constraints='[+%s=%s]';", policyP2, key, values[1]))

	out, err := mysql.M.ExecuteForceWithOutput(createPolicySQL, mysql.M.User, mysql.M.Password)
	if err != nil {
		return err
	}

	output := make([]string, 0)
	output = append(output, out...)
	log.Logger.Infof("[%s] create placement policy [%s,%s] for label %s [%s,%s]", ov, policyP1, policyP2, key, values[0], values[1])
	log.Logger.Infof("[%s] create table %s.%s for %s, %s.%s for %s and init data...", ov, db, table1, policyP1, db, table2, policyP2)
	sb := bench.Sysbench{
		Test:      bench.OltpReadWrite,
		Mysql:     mysql.M,
		Db:        "poc",
		TableSize: 100000,
		Tables:    2,
		Threads:   5,
		Cmd:       "prepare",
	}
	if _, err := sb.Run(); err != nil {
		return err
	}
	alterSQL := mysql.AlterPlacementPolicy(fmt.Sprintf("%s.%s", db, table1), policyP1) +
		mysql.AlterPlacementPolicy(fmt.Sprintf("%s.%s", db, table2), policyP2)
	out, err = mysql.M.ExecuteForceWithOutput(alterSQL, mysql.M.User, mysql.M.Password)
	if err != nil {
		return err
	}
	output = append(output, out...)

	time.Sleep(5 * time.Second)
	selectSQL := fmt.Sprintf(comp.LeaderDistributionSQL, table1) +
		fmt.Sprintf(comp.LeaderDistributionSQL, table2)
	out, err = mysql.M.ExecuteForceWithOutput(selectSQL, mysql.M.User, mysql.M.Password)
	if err != nil {
		return err
	}
	output = append(output, out...)
	defer func() {
		j.writeResultFile(ov, 1, 0, output)
	}()
	return nil
}

func confirmLabels(values [][]ms.FieldValue) (string, []string) {
	var key string
	var value []string
	for i := len(values) - 1; i >= 0; i-- {
		if string(values[i][0].AsString()) == "engine" {
			continue
		}
		value = strings.Split(string(values[i][1].AsString()), ",")
		if len(value) > 1 {
			key = string(values[i][0].AsString())
			break
		}
	}
	return key, value
}

func processLabel(value []string) []string {
	var newValue []string
	for _, v := range value {
		v = strings.Trim(v, "[")
		v = strings.Trim(v, "]")
		v = strings.Trim(v, "\"")
		v = strings.Trim(v, " \"")
		newValue = append(newValue, v)
	}
	return newValue
}

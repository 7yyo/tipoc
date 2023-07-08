package job

import (
	"fmt"
	"pictorial/comp"
	"pictorial/log"
	"pictorial/mysql"
	"pictorial/operator"
	"pictorial/ssh"
	"strings"
	"time"
)

func (j *Job) runDataSeparation() {
	ov := operator.GetOTypeValue(operator.DataSeparation)
	rs, err := mysql.M.ExecuteSQL("SHOW PLACEMENT LABELS;")
	if err != nil {
		j.ErrC <- err
		return
	}

	output := make([]string, 0)
	defer func() {
		j.writeResultFile(ov, 1, 0, output)
	}()

	var key string
	var value []string
	var isContinue bool
	for _, r := range rs.Values {
		key = string(r[0].AsString())
		value = strings.Split(string(r[1].AsString()), ",")
		if len(value) > 1 {
			isContinue = true
			break
		}
	}
	var values []string
	for _, v := range value {
		v = strings.Trim(v, "[")
		v = strings.Trim(v, "]")
		v = strings.Trim(v, "\"")
		v = strings.Trim(v, " \"")
		values = append(values, v)
	}
	if !isContinue {
		j.ErrC <- fmt.Errorf(fmt.Sprintf("[%s] no label instance more than 1, stop.", ov))
		return
	}
	log.Logger.Infof("[%s] choose label for test: %s %s", ov, key, values)

	createPolicySQL := fmt.Sprintf(
		"drop placement policy if exists p1; create placement policy p1 leader_constraints='[+%s=%s]';"+
			"drop placement policy if exists p2; create placement policy p2 leader_constraints='[+%s=%s]';",
		key, values[0], key, values[1],
	)
	out, err := mysql.M.ExecuteForceWithOutput(createPolicySQL, mysql.M.User, mysql.M.Password)
	if err != nil {
		j.ErrC <- err
		return
	}
	output = append(output, out...)
	log.Logger.Infof("[%s] create placement policy [p1, p2] for label: %s[%s, %s]", ov, key, values[0], values[1])

	log.Logger.Infof("create table sbtest1, sbtest2 for p1, p2.")
	prepareCmd := fmt.Sprintf(oltpWriteRead, mysql.M.Host, mysql.M.Port, mysql.M.User, mysql.M.Password, "poc", "1000000", "2", "10", "prepare")
	if _, err = ssh.S.RunLocal(prepareCmd); err != nil {
		j.ErrC <- err
		return
	}
	alterSQL := "use poc; " +
		"alter table sbtest1 placement policy = p1; " +
		"alter table sbtest2 placement policy = p2;"
	out, err = mysql.M.ExecuteForceWithOutput(alterSQL, mysql.M.User, mysql.M.Password)
	if err != nil {
		j.ErrC <- err
		return
	}
	output = append(output, out...)
	selectSQL := fmt.Sprintf("%s;%s",
		fmt.Sprintf(comp.LeaderDistributionSQL, "sbtest1"),
		fmt.Sprintf(comp.LeaderDistributionSQL, "sbtest2"))
	time.Sleep(5 * time.Second)
	out, err = mysql.M.ExecuteForceWithOutput(selectSQL, mysql.M.User, mysql.M.Password)
	if err != nil {
		j.ErrC <- err
		return
	}
	output = append(output, out...)
}

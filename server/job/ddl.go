package job

import (
	"fmt"
	"github.com/gizak/termui/v3/widgets"
	"path/filepath"
	"pictorial/log"
	"pictorial/mysql"
	"pictorial/operator"
	"pictorial/ssh"
	"pictorial/widget"
	"time"
)

func (j *Job) runOnlineDDL() {
	j.selected.Walk(func(node *widgets.TreeNode) bool {
		e := widget.ChangeToExample(node)
		switch e.OType {
		case operator.OnlineDDLAddIndex:
			j.runOnlineDDLAddIndex()
		}
		return true
	})
}

func (j *Job) runOnlineDDLAddIndex() {
	ov := operator.GetOTypeValue(operator.OnlineDDLAddIndex)
	j.runInstallSysBench()
	prepareCmd := fmt.Sprintf(oltpWriteRead, mysql.M.Host, mysql.M.Port, mysql.M.User, mysql.M.Password, "1000000", "1", "10", "prepare")
	log.Logger.Info(fmt.Sprintf("[%s] init data: %s", ov, prepareCmd))
	if _, err := ssh.S.RunLocal(prepareCmd); err != nil {
		j.ErrC <- err
		return
	}
	go func() {
		runCmd := fmt.Sprintf(oltpWriteRead, mysql.M.Host, mysql.M.Port, mysql.M.User, mysql.M.Password, "1000000", "1", "10", "--time=3600 run")
		log.Logger.Infof("[%s] run sysbench oltp_write_read.", "online_ddl_add_index")
		Ld.Cmd = runCmd
		logName := filepath.Join(j.resultPath, "load.log")
		go Ld.captureLoadLog(logName, j.ErrC, j.LdC)
		go Ld.run(logName, j.ErrC, j.StopC)
	}()
	time.Sleep(1 * time.Second)
	cntDown("add index k1(c)", Ld.Interval)
	log.Logger.Infof("[%s] %s", ov, "add index k1(c)")
	sql := fmt.Sprintf("" +
		"show create table poc.sbtest1;" +
		"alter table poc.sbtest1 add index k1(c);" +
		"show create table poc.sbtest1;")
	output, err := mysql.M.ExecuteForceWithOutput(sql, mysql.M.User, mysql.M.Password)
	if err != nil {
		j.ErrC <- err
		return
	}
	log.Logger.Infof("[%s] %s", ov, "add index k1(c) complete")
	j.writeResultFile(ov, 1, 0, output)
}

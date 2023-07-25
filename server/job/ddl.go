package job

import (
	"fmt"
	"github.com/gizak/termui/v3/widgets"
	"path/filepath"
	"pictorial/bench"
	"pictorial/log"
	"pictorial/mysql"
	"pictorial/operator"
	"pictorial/widget"
	"time"
)

func (j *Job) runOnlineDDL() error {
	var err error
	j.selected.Walk(func(node *widgets.TreeNode) bool {
		e := widget.ChangeToExample(node)
		switch e.OType {
		case operator.OnlineDDLAddIndex, operator.OnlineDDLModifyColumn:
			err = j.runOnlineDDLAlter(e.OType)
		case operator.AddIndexPerformance:
			err = j.runAddIndexPerformance()
		}
		return true
	})
	return err
}

func (j *Job) runOnlineDDLAlter(oType operator.OType) error {
	ov := operator.GetOTypeValue(oType)
	if err := bench.InstallSysBench(); err != nil {
		return err
	}
	sb := bench.Sysbench{
		Test:      bench.OltpReadWrite,
		Mysql:     mysql.M,
		Db:        "poc",
		TableSize: 1000000,
		Tables:    1,
		Threads:   5,
		Cmd:       "prepare",
	}
	log.Logger.Info(fmt.Sprintf("[%s] init data: %s", ov, sb.String()))
	if _, err := sb.Run(); err != nil {
		return err
	}
	go func() {
		sb.Cmd = "run"
		log.Logger.Infof("[%s] run sysbench %s.", ov, bench.GetSysbenchTpValue(bench.OltpReadWrite))
		Ld.Cmd = sb.String()
		logName := filepath.Join(j.resultPath, "load.log")
		go Ld.captureLoadLog(logName, j.ErrC, j.LdC)
		go Ld.run(logName, j.ErrC, j.StopC)
	}()
	time.Sleep(1 * time.Second)

	table := "poc.sbtest1"
	index := "k_2(c)"
	col := "k bigint"

	var ddl string
	switch oType {
	case operator.OnlineDDLModifyColumn:
		ddl = mysql.ModifyColumn(table, col)
	case operator.OnlineDDLAddIndex:
		ddl = mysql.AddIndex(table, index)
	}
	cntDown(ddl, Ld.Interval)
	sql := mysql.ShowCreateTable(table) + ddl + mysql.ShowCreateTable(table)
	output, err := mysql.M.ExecuteForceWithOutput(sql, mysql.M.User, mysql.M.Password)
	if err != nil {
		return err
	}
	defer j.writeResultFile(ov, 1, 0, output)
	log.Logger.Infof("[%s] %s complete", ov, ddl)
	return nil
}

func (j *Job) runAddIndexPerformance() error {
	table := "poc.sbtest1"
	index := "k_2(c)"
	addIndexSQL := mysql.AddIndex(table, index)
	ov := operator.GetOTypeValue(operator.AddIndexPerformance)
	if err := bench.InstallSysBench(); err != nil {
		return err
	}
	sb := bench.Sysbench{
		Test:      bench.OltpReadWrite,
		Mysql:     mysql.M,
		Db:        "poc",
		TableSize: 5000000,
		Tables:    1,
		Threads:   5,
		Cmd:       "prepare",
	}
	log.Logger.Info(fmt.Sprintf("[%s] init data: %s", ov, sb.String()))
	if _, err := sb.Run(); err != nil {
		return err
	}
	cntDown(addIndexSQL, Ld.Interval)
	script := mysql.ShowCreateTable(table) +
		mysql.Count(table) +
		addIndexSQL +
		mysql.ShowCreateTable(table)
	output, err := mysql.M.ExecuteForceWithOutput(script, mysql.M.User, mysql.M.Password)
	if err != nil {
		return err
	}
	defer j.writeResultFile(ov, 1, 0, output)
	log.Logger.Infof("[%s] %s complete", addIndexSQL, ov)
	return nil
}

package job

import (
	"fmt"
	"github.com/gizak/termui/v3/widgets"
	"path/filepath"
	"pictorial/bench"
	"pictorial/log"
	"pictorial/mysql"
	"pictorial/operator"
	"pictorial/util/http"
	"pictorial/widget"
)

func (j *Job) runLoadData() error {
	var err error
	j.selected.Walk(func(node *widgets.TreeNode) bool {
		e := widget.ChangeToExample(node)
		switch e.OType {
		case operator.LoadDataTPCC:
			err = j.runTPCCLoadData()
		case operator.LoadDataImportInto:
			err = j.runImportInto()
		case operator.LoadData:
			err = j.runLoadDataJob()
		case operator.LoadDataSelectIntoOutFile:
			err = j.runSelectIntoOutFile()
		}
		return true
	})
	return err
}

func (j *Job) runTPCCLoadData() error {
	ov := operator.GetOTypeValue(operator.LoadDataTPCC)
	clean := bench.Tpcc{
		Mysql:      mysql.M,
		DB:         "poc",
		Warehouses: 10,
		Threads:    5,
		Cmd:        "clean",
	}
	prepare := clean
	prepare.Cmd = "prepare"
	logPath := fmt.Sprintf("%s/%s.log", j.resultPath, ov)
	log.Logger.Infof("[%s] %s", ov, prepare.String())
	go Ld.captureLoadLog(logPath, j.Channel.ErrC, j.Channel.LdC)
	originalCmd := Ld.Cmd
	defer func() {
		Ld.Cmd = originalCmd
	}()
	Ld.Cmd = clean.String() + prepare.String()
	Ld.run(logPath, j.Channel.ErrC, nil)
	defer func() {
		j.Channel.BarC <- 1
	}()
	return nil
}

func (j *Job) runImportInto() error {
	rowCnt := 1000000
	colCnt := 10
	ov := operator.GetOTypeValue(operator.LoadDataImportInto)
	table := fmt.Sprintf("poc.%s", ov)
	csvPath := filepath.Join(j.resultPath, fmt.Sprintf("%s.csv", ov))
	if err := mysql.InitCSV(csvPath, rowCnt, colCnt); err != nil {
		return err
	}
	ddl := mysql.CreateTableSQL(ov, 10)
	if _, err := mysql.M.ExecuteSQL(ddl.String()); err != nil {
		return err
	}
	script := mysql.Count(table) +
		mysql.ImportInto(table, csvPath) +
		mysql.Count(table) +
		fmt.Sprintf("select * from %s limit 50", table)
	log.Logger.Infof("[%s] start import by csv: %s", ov, csvPath)
	output, err := mysql.M.ExecuteForceWithOutput(script, mysql.M.User, mysql.M.Password)
	if err != nil {
		return err
	}
	defer j.writeResultFile(ov, 1, 0, output)
	j.Channel.BarC <- 1
	return nil
}

func (j *Job) runLoadDataJob() error {
	ov := operator.GetOTypeValue(operator.LoadData)
	csvPath := filepath.Join(j.resultPath, fmt.Sprintf("%s.csv", ov))
	rowCnt := 50000
	colCnt := 10
	table := fmt.Sprintf("poc.%s", ov)
	if err := mysql.InitCSV(csvPath, rowCnt, colCnt); err != nil {
		return err
	}
	ddl := mysql.CreateTableSQL(ov, 10)
	if _, err := mysql.M.ExecuteSQL(ddl.String()); err != nil {
		return err
	}
	sql := mysql.Count(table) + mysql.LoadData(table, csvPath) + mysql.Count(table)
	log.Logger.Infof("[%s] %s", ov, mysql.LoadData(table, csvPath))
	output, err := mysql.M.ExecuteForceWithOutput(sql, mysql.M.User, mysql.M.Password)
	if err != nil {
		return err
	}
	defer func() {
		j.writeResultFile(ov, 1, 0, output)
		j.Channel.BarC <- 1
	}()
	return nil
}

func (j *Job) runSelectIntoOutFile() error {
	ov := operator.GetOTypeValue(operator.LoadDataSelectIntoOutFile)
	hit, err := http.MatchIp()
	if !hit {
		return fmt.Errorf("[%s] please connect the tidb-server on this server, If not, deploy one.", ov)
	}
	if err := bench.InstallSysBench(); err != nil {
		return err
	}
	sb := bench.Sysbench{
		Test:      bench.OltpInsert,
		Mysql:     mysql.M,
		Db:        "poc",
		TableSize: 1000000,
		Tables:    1,
		Threads:   5,
		Cmd:       "prepare",
	}
	log.Logger.Infof("[%s] %s", ov, sb.String())
	if _, err := sb.Run(); err != nil {
		return err
	}
	table := "poc.sbtest1"
	csvPath := filepath.Join(j.resultPath, fmt.Sprintf("%s.csv", ov))
	log.Logger.Infof("[%s] start dump data from table: %s", ov, table)
	sql := mysql.Count(table) + mysql.SelectInfoFile(table, csvPath)
	output, err := mysql.M.ExecuteForceWithOutput(sql, mysql.M.User, mysql.M.Password)
	if err != nil {
		return err
	}
	defer func() {
		j.writeResultFile(ov, 1, 0, output)
		j.Channel.BarC <- 1
	}()
	return nil
}

func (j *Job) runDataDistribution() error {
	ov := operator.GetOTypeValue(operator.DataDistribution)
	lName := fmt.Sprintf("%s/%s.log", j.resultPath, ov)
	sb := bench.Sysbench{
		Test:      bench.OltpInsert,
		Mysql:     mysql.M,
		Db:        "poc",
		TableSize: 10000000,
		Tables:    1,
		Threads:   5,
		Cmd:       "prepare",
	}
	log.Logger.Infof("[%s] %s", ov, sb.String())
	if err := bench.InstallSysBench(); err != nil {
		return err
	}
	originalCmd := Ld.Cmd
	defer func() {
		Ld.Cmd = originalCmd
	}()
	Ld.Cmd = sb.String()
	go Ld.captureLoadLog(lName, j.Channel.ErrC, j.Channel.LdC)
	Ld.run(lName, j.Channel.ErrC, nil)
	j.BarC <- 1
	return nil
}

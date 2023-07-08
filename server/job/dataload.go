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
)

func (j *Job) runLoadData() {
	j.selected.Walk(func(node *widgets.TreeNode) bool {
		e := widget.ChangeToExample(node)
		switch e.OType {
		case operator.LoadDataTPCC:
			j.runTPCCLoadData()
		case operator.LoadDataImportInto:
			j.runImportInto()
		case operator.LoadData:
			j.runLoadDataJob()
		case operator.LoadDataSelectIntoOutFile:
			j.runSelectIntoOutFile()
		}
		return true
	})
	j.Channel.BarC <- 1
}

func (j *Job) runTPCCLoadData() {
	tiupRoot, err := ssh.S.WhereTiup()
	if err != nil {
		j.Channel.ErrC <- err
		return
	}
	warehouses := 10
	threads := 10
	lName := fmt.Sprintf("%s/%s.log", j.resultPath, operator.GetOTypeValue(operator.LoadDataTPCC))
	log.Logger.Infof("[%s] warehouses: %d, threads: %d, database: %s", operator.GetOTypeValue(operator.LoadDataTPCC), warehouses, threads, "tpcc_pp")
	go Ld.captureLoadLog(lName, j.Channel.ErrC, j.Channel.LdC)
	tpccCmd := fmt.Sprintf("%s/bin/tiup bench tpcc -H %s -P %s -D tpcc_pp", tiupRoot, mysql.M.Host, mysql.M.Port)
	cleanCmd := fmt.Sprintf("%s clean", tpccCmd)
	prepareCmd := fmt.Sprintf("%s --warehouses %d --threads %d prepare", tpccCmd, warehouses, threads)
	originalCmd := Ld.Cmd
	defer func() {
		Ld.Cmd = originalCmd
	}()
	Ld.Cmd = fmt.Sprintf("%s;%s", cleanCmd, prepareCmd)
	Ld.run(lName, j.Channel.ErrC, nil)
}

func (j *Job) runImportInto() {
	rowCnt := 1000000
	colCnt := 10
	ov := operator.GetOTypeValue(operator.LoadDataImportInto)
	csvPath := filepath.Join(j.resultPath, fmt.Sprintf("%s.csv", ov))
	if err := mysql.InitCSV(csvPath, rowCnt, colCnt); err != nil {
		j.Channel.ErrC <- err
		return
	}
	ddl := mysql.CreateTableSQL(ov, 10)
	if _, err := mysql.M.ExecuteSQL(ddl.String()); err != nil {
		j.Channel.ErrC <- err
		return
	}
	sql := fmt.Sprintf(""+
		"select count(*) from poc.%s; "+
		"import into poc.import_into from '%s'; "+
		"select count(*) from poc.%s; "+
		"select * from poc.%s limit 50;",
		ov, csvPath, ov, ov)
	log.Logger.Infof("[%s] start import by csv: %s", ov, csvPath)
	output, err := mysql.M.ExecuteForceWithOutput(sql, "root", "")
	if err != nil {
		j.Channel.ErrC <- err
		return
	}
	j.writeResultFile(ov, 1, 0, output)
	j.Channel.BarC <- 1
}

func (j *Job) runLoadDataJob() {
	ov := operator.GetOTypeValue(operator.LoadData)
	csvPath := filepath.Join(j.resultPath, fmt.Sprintf("%s.csv", ov))
	rowCnt := 50000
	colCnt := 10
	if err := mysql.InitCSV(csvPath, rowCnt, colCnt); err != nil {
		j.Channel.ErrC <- err
		return
	}
	ddl := mysql.CreateTableSQL(ov, 10)
	if _, err := mysql.M.ExecuteSQL(ddl.String()); err != nil {
		j.Channel.ErrC <- err
		return
	}
	sql := fmt.Sprintf(""+
		"select count(*) from poc.%s; "+
		"load data local infile '%s' into table poc.load_data fields terminated by ','; "+
		"select count(*) from poc.%s; "+
		"select * from poc.%s limit 50;",
		ov, csvPath, ov, ov)
	log.Logger.Infof("start load_data")
	output, err := mysql.M.ExecuteForceWithOutput(sql, "root", "")
	if err != nil {
		j.Channel.ErrC <- err
		return
	}
	j.writeResultFile(ov, 1, 0, output)
	j.Channel.BarC <- 1
}

func (j *Job) runSelectIntoOutFile() {
	ov := operator.GetOTypeValue(operator.LoadDataSelectIntoOutFile)
	j.runInstallSysBench()
	load := fmt.Sprintf(oltpInsert, mysql.M.Host, mysql.M.Port, mysql.M.User, mysql.M.Password, "poc", "1000000", "1", "5", "prepare")
	log.Logger.Infof("[%s] %s", ov, load)
	if _, err := ssh.S.RunLocal(load); err != nil {
		j.ErrC <- err
		return
	}
	csvPath := filepath.Join(j.resultPath, fmt.Sprintf("%s.csv", ov))
	log.Logger.Infof("[%s] start dump data from table: %s", ov, ov)
	sql := fmt.Sprintf(""+
		"select count(*) from poc.sbtest1;"+
		"select * from poc.sbtest1 into outfile '%s' fields terminated by ',';",
		csvPath)
	output, err := mysql.M.ExecuteForceWithOutput(sql, "root", "")
	if err != nil {
		j.ErrC <- err
		return
	}
	j.writeResultFile(ov, 1, 0, output)
	j.Channel.BarC <- 1
}

func (j *Job) runDataDistribution() {
	ov := operator.GetOTypeValue(operator.DataDistribution)
	lName := fmt.Sprintf("%s/%s.log", j.resultPath, ov)
	sysbench := fmt.Sprintf(oltpInsert, mysql.M.Host, mysql.M.Port, mysql.M.User, mysql.M.Password, "poc", "10000000", "1", "5", "prepare")
	log.Logger.Infof("[%s] %s", ov, sysbench)
	go Ld.captureLoadLog(lName, j.Channel.ErrC, j.Channel.LdC)
	j.runInstallSysBench()
	originalCmd := Ld.Cmd
	defer func() {
		Ld.Cmd = originalCmd
	}()
	Ld.Cmd = sysbench
	Ld.run(lName, j.Channel.ErrC, nil)
	j.BarC <- 1
}

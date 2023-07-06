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
	tiupRoot, err := ssh.S.WhichTiup()
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
	Ld.Cmd = fmt.Sprintf("%s;%s", cleanCmd, prepareCmd)
	// todo if use Ld, will instead of original config cmd
	Ld.run(lName, j.Channel.ErrC, nil)
}

func (j *Job) runImportInto() {
	rowCnt := 100000
	colCnt := 10
	tp := operator.GetOTypeValue(operator.LoadDataImportInto)
	csvPath := filepath.Join(j.resultPath, fmt.Sprintf("%s.csv", tp))
	if err := mysql.InitCSV(csvPath, rowCnt, colCnt); err != nil {
		j.Channel.ErrC <- err
		return
	}
	if _, err := mysql.M.ExecuteSQL(fmt.Sprintf("drop table if exists poc.%s", tp)); err != nil {
		j.Channel.ErrC <- err
		return
	}
	ddl := mysql.CreateTableSQL(tp, 10)
	if _, err := mysql.M.ExecuteSQL(ddl.String()); err != nil {
		j.Channel.ErrC <- err
		return
	}
	sql := fmt.Sprintf(""+
		"select count(*) from poc.%s; "+
		"import into poc.import_into from '%s'; "+
		"select count(*) from poc.%s; "+
		"select * from poc.%s limit 50;",
		tp, csvPath, tp, tp)
	log.Logger.Debug(sql)
	log.Logger.Infof("start %s", tp)
	output, err := mysql.M.ExecuteForceWithOutput(sql, "root", "")
	if err != nil {
		j.Channel.ErrC <- err
		return
	}
	j.writeResultFile(tp, 1, 0, output)
	j.Channel.BarC <- 1
}

func (j *Job) runLoadDataJob() {
	tp := operator.GetOTypeValue(operator.LoadData)
	csvPath := filepath.Join(j.resultPath, fmt.Sprintf("%s.csv", tp))
	rowCnt := 50000
	colCnt := 10
	if err := mysql.InitCSV(csvPath, rowCnt, colCnt); err != nil {
		j.Channel.ErrC <- err
		return
	}
	if _, err := mysql.M.ExecuteSQL(fmt.Sprintf("drop table if exists poc.%s", tp)); err != nil {
		j.Channel.ErrC <- err
		return
	}
	ddl := mysql.CreateTableSQL(tp, 10)
	if _, err := mysql.M.ExecuteSQL(ddl.String()); err != nil {
		j.Channel.ErrC <- err
		return
	}
	sql := fmt.Sprintf(""+
		"select count(*) from poc.%s; "+
		"load data local infile '%s' into table poc.load_data fields terminated by ','; "+
		"select count(*) from poc.%s; "+
		"select * from poc.%s limit 50;",
		tp, csvPath, tp, tp)
	log.Logger.Debug(sql)
	log.Logger.Infof("start load_data")
	output, err := mysql.M.ExecuteForceWithOutput(sql, "root", "")
	if err != nil {
		j.Channel.ErrC <- err
		return
	}
	j.writeResultFile(tp, 1, 0, output)
	j.Channel.BarC <- 1
}

func (j *Job) runSelectIntoOutFile() {
	tp := operator.GetOTypeValue(operator.LoadDataSelectIntoOutFile)
	if _, err := mysql.M.ExecuteSQL(fmt.Sprintf("drop table if exists poc.%s", tp)); err != nil {
		j.ErrC <- err
		return
	}
	ddl := mysql.CreateTableSQL(operator.GetOTypeValue(operator.LoadDataSelectIntoOutFile), 3)
	if _, err := mysql.M.ExecuteSQL(ddl.String()); err != nil {
		j.Channel.ErrC <- err
		return
	}
	rowCnt := 1000
	log.Logger.Infof("[%s] start init data: %d", tp, rowCnt)
	for i := 1; i <= rowCnt; i++ {
		insert := fmt.Sprintf(
			"insert into poc.%s values("+
				"%d,'%s','%s','%s');",
			tp, i,
			mysql.RandomStr(11),
			mysql.RandomStr(11),
			mysql.RandomStr(11))
		log.Logger.Debug(insert)
		if _, err := mysql.M.ExecuteSQL(insert); err != nil {
			j.ErrC <- err
			return
		}
	}
	csvPath := filepath.Join(j.resultPath, fmt.Sprintf("%s.csv", tp))
	log.Logger.Infof("[%s] start dump data from table: %s", tp, tp)
	sql := fmt.Sprintf(""+
		"select count(*) from poc.%s;"+
		"select * from poc.%s into outfile '%s' fields terminated by ',';",
		tp, tp, csvPath)
	output, err := mysql.M.ExecuteForceWithOutput(sql, "root", "")
	if err != nil {
		j.ErrC <- err
		return
	}
	j.writeResultFile(tp, 1, 0, output)
	j.Channel.BarC <- 1
}

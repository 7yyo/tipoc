package server

import (
	"fmt"
	"github.com/gizak/termui/v3/widgets"
	"io"
	"net"
	"os"
	"path/filepath"
	"pictorial/comp"
	"pictorial/log"
	"pictorial/mysql"
	"pictorial/operator"
	"pictorial/ssh"
	"pictorial/widget"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

type job struct {
	selected   *widgets.Tree
	examples   map[string][]string
	components map[comp.CType][]comp.Component
	channel
	resultPath string
}

type channel struct {
	barC      chan int
	ldC       chan string
	stopC     chan bool
	errC      chan error
	completeC chan bool
}

var loadJob = []widget.OType{
	widget.ScaleIn,
	widget.Kill,
	widget.DataCorrupted,
	widget.Crash,
	widget.Disaster,
	widget.Reboot,
	widget.DiskFull,
}

func isLoadJob(o widget.OType) bool {
	for _, oType := range loadJob {
		if o == oType {
			return true
		}
	}
	return false
}

const resultPath = "./result"

func newJob(e map[string][]string, s *widgets.Tree) job {
	mkdirResultPath := func() string {
		if err := os.MkdirAll(resultPath, os.ModePerm); err != nil {
			panic(err)
		}
		fp, err := os.Getwd()
		if err != nil {
			panic(err)
		}
		return filepath.Join(fp, resultPath)
	}
	c, err := comp.New()
	if err != nil {
		panic(err)
	}
	return job{
		examples:   e,
		selected:   s,
		components: c.Map,
		channel: channel{
			barC:      make(chan int),
			ldC:       make(chan string),
			stopC:     make(chan bool),
			errC:      make(chan error),
			completeC: make(chan bool),
		},
		resultPath: mkdirResultPath(),
	}
}

const completeSignal = "complete_signal"

func (j *job) run() {

	tp := j.tp()
	j.printSelected(tp)

	switch tp {
	case widget.Script, widget.OtherScript:
		j.runScript()
	case widget.SafetyScript:
		j.runSafety()
	default:
		if err := j.createOTypeResult(); err != nil {
			j.errC <- err
			return
		}
		if isLoadJob(tp) {
			if ld.cmd != "" {
				ldName := filepath.Join(j.resultPath, "load.log")
				go ld.run(ldName, j.channel.errC, j.channel.stopC)
				go ld.captureLoadLog(ldName, j.errC, j.ldC)
				time.Sleep(time.Second * 1)
				cntDown("run items", ld.interval)
			}
		}
		switch tp {
		case widget.Disaster:
			j.runLabel()
		case widget.LoadDataTPCC:
			j.runLoadData()
		default:
			j.runComponent()
		}
		if isLoadJob(tp) {
			cntDown("render", ld.interval)
			j.channel.stopC <- true
			if err := j.components[comp.Grafana][0].Render(j.resultPath); err != nil {
				j.errC <- err
				return
			}
		}
		if _, err := ssh.S.Transfer(ssh.ShellLog, j.resultPath); err != nil {
			j.errC <- err
			return
		}
	}
	j.channel.completeC <- true
	log.Logger.Infof("complete at %s.", j.resultPath)
}

func isCompleteSignal(err error) bool {
	return err.Error() == completeSignal
}

func (j *job) runScript() {
	if err := resetDB(); err != nil {
		j.errC <- err
		return
	}
	log.Logger.Info("reset DB complete, start job")
	var cnt int
	j.selected.Walk(func(i *widgets.TreeNode) bool {
		cnt++
		var idx int32
		var wg sync.WaitGroup
		var errOut string
		name := i.Value.String()
		scripts := j.examples[i.Value.String()]
		for _, s := range scripts {
			wg.Add(1)
			go func(sql string) {
				defer wg.Done()
				n := atomic.AddInt32(&idx, 1)
				output, err := mysql.M.ExecuteForceWithOutput(sql, "root", "")
				if err != nil {
					errOut = err.Error()
				}
				j.writeResultFile(name, len(scripts), int(n), output)
			}(s)
		}
		wg.Wait()
		if errOut != "" {
			log.Logger.Infof("[warn] %s: %s", name, errOut)
		} else {
			log.Logger.Infof("[pass] %s", name)
		}
		j.channel.barC <- cnt
		return true
	})
}

func (j *job) runComponent() {
	var cnt int
	j.selected.Walk(func(node *widgets.TreeNode) bool {
		cnt++
		j.barC <- cnt
		e := widget.ChangeToExample(node)
		addr := strings.Trim(e.String(), comp.Leader)
		var failMsg = "[%s] %s failed: %s"
		for _, c := range j.components[e.CType] {
			c.Port = strings.Trim(c.Port, comp.Leader)
			if addr == net.JoinHostPort(c.Host, c.Port) {
				b := operator.Builder{
					OType:      e.OType,
					CType:      e.CType,
					Host:       c.Host,
					Port:       c.Port,
					DeployPath: c.DeployPath,
					StopC:      j.stopC,
				}
				r, err := b.Build()
				if err != nil {
					log.Logger.Error(err)
					return true
				}
				if err = r.Execute(); err != nil {
					log.Logger.Errorf(failMsg, widget.GetOTypeValue(e.OType), addr, err.Error())
					return true
				}
			}
		}
		time.Sleep(time.Second * ld.sleep)
		return true
	})
}

func (j *job) runLabel() {
	kvs := j.components[comp.TiKV]
	j.selected.Walk(func(i *widgets.TreeNode) bool {
		targetLabel := i.Value.String()
		for _, kv := range kvs {
			for _, v := range kv.Labels {
				if targetLabel == v {
					b := operator.Builder{
						Host:  kv.Host,
						Port:  kv.Port,
						OType: widget.Crash,
						CType: comp.TiKV,
					}
					r, _ := b.Build()
					if err := r.Execute(); err != nil {
						log.Logger.Errorf("[disaster] %s failed: %v", net.JoinHostPort(b.Host, b.Port), err)
					}
					break
				}
			}
		}
		log.Logger.Infof("[disaster] %s", targetLabel)
		return true
	})
	cntDown("render", ld.interval)
}

func (j *job) runLoadData() {
	tiupRoot, err := ssh.S.WhichTiup()
	if err != nil {
		j.errC <- err
		return
	}
	j.selected.Walk(func(node *widgets.TreeNode) bool {
		e := widget.ChangeToExample(node)
		var l load
		var lName string
		switch e.OType {
		case widget.LoadDataTPCC:
			warehouses := 10
			threads := 10
			lName = fmt.Sprintf("%s/%s.log", j.resultPath, widget.GetOTypeValue(e.OType))
			log.Logger.Infof("[%s] warehouses: 10, threads: 10, database: %s", widget.GetOTypeValue(e.OType), "tpcc_pp")
			go ld.captureLoadLog(lName, j.errC, j.ldC)
			tpccCmd := fmt.Sprintf("%s/bin/tiup bench tpcc -H %s -P %s -D tpcc_pp", tiupRoot, mysql.M.Host, mysql.M.Port)
			cleanCmd := fmt.Sprintf("%s clean", tpccCmd)
			prepareCmd := fmt.Sprintf("%s --warehouses %d --threads %d prepare", tpccCmd, warehouses, threads)
			l.cmd = fmt.Sprintf("%s;%s", cleanCmd, prepareCmd)
			l.run(lName, j.errC, nil)
		}
		return true
	})
}

func (j *job) writeResultFile(name string, len, n int, output []string) {
	var fName string
	if len == 1 {
		fName = filepath.Join(j.resultPath, name)
	} else {
		fName = filepath.Join(j.resultPath, fmt.Sprintf("%s_%d", name, n))
	}
	f, err := os.Create(fName)
	if err != nil {
		log.Logger.Warnf("write %s failed: %s", name, err.Error())
	}
	defer f.Close()
	for _, o := range output {
		_, err = io.WriteString(f, fmt.Sprintf("%s\n", o))
		if err != nil {
			log.Logger.Warnf("write %s failed: %s", name, err.Error())
		}
	}
}

const rootUser = "## root"
const tidbUser = "## tidb_user"
const tidbUserName = "tidb_user"
const tidbUserPassword = "tidb_password"

func (j *job) runSafety() {
	if err := resetDB(); err != nil {
		j.errC <- err
		return
	}
	var cnt int
	j.selected.Walk(func(i *widgets.TreeNode) bool {
		name := i.Value.String()
		scripts := j.examples[i.Value.String()]
		var err error
		var output []string
		var errOutput string
		for i, sql := range scripts {
			user := strings.Split(sql, "\n")[0]
			sql = strings.Trim(sql, "\n")
			switch {
			case strings.Contains(user, rootUser):
				sql = strings.Trim(sql, fmt.Sprintf("%s\n", rootUser))
				output, err = mysql.M.ExecuteForceWithOutput(sql, "root", "")
			case strings.Contains(user, tidbUser):
				sql = strings.Trim(sql, fmt.Sprintf("%s\n", tidbUser))
				output, err = mysql.M.ExecuteForceWithOutput(sql, tidbUserName, tidbUserPassword)
			default:
				err = fmt.Errorf("invalid username, please use 'root' and 'tidb_user'")
			}
			if err != nil {
				errOutput = err.Error()
			}
			j.writeResultFile(name, len(scripts), i, output)
		}
		if errOutput != "" {
			log.Logger.Infof("[warn] %s: %s", name, errOutput)
		} else {
			log.Logger.Infof("[pass] %s", name)
		}
		cnt++
		j.channel.barC <- cnt
		return true
	})

}

func (j *job) createOTypeResult() error {
	result := fmt.Sprintf("%s/%s_%s", resultPath, j.selected.Title, log.DateFormat())
	if err := os.MkdirAll(result, os.ModePerm); err != nil {
		return err
	}
	f, err := filepath.Abs(result)
	if err != nil {
		return err
	}
	j.resultPath = f
	return nil
}

func (j *job) tp() widget.OType {
	return j.selected.SelectedNode().Value.(*widget.Example).OType
}

func resetDB() error {
	if _, err := mysql.M.ExecuteSQL("DROP DATABASE IF EXISTS poc"); err != nil {
		return err
	}
	if _, err := mysql.M.ExecuteSQL("CREATE DATABASE poc"); err != nil {
		return err
	}
	if _, err := mysql.M.ExecuteSQL("SET GLOBAL validate_password.enable = OFF;"); err != nil {
		return err
	}
	return nil
}

func (j *job) printSelected(tp widget.OType) {
	log.Logger.Info("you selected:")
	cnt := 0
	j.selected.Walk(func(node *widgets.TreeNode) bool {
		cnt++
		switch tp {
		case widget.Script, widget.OtherScript, widget.SafetyScript:
			log.Logger.Infof("[%d] %s", cnt, node.Value.String())
		default:
			log.Logger.Infof("[%d] %s_%s", cnt, widget.GetOTypeValue(tp), node.Value.String())
		}
		return true
	})
}

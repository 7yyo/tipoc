package job

import (
	"context"
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

type Job struct {
	selected   *widgets.Tree
	examples   map[string][]string
	components map[comp.CType][]comp.Component
	Channel
	resultPath string
}

type Channel struct {
	BarC      chan int
	LdC       chan string
	StopC     chan bool
	ErrC      chan error
	CompleteC chan bool
}

var loadJob = []operator.OType{
	operator.ScaleIn,
	operator.Kill,
	operator.DataCorrupted,
	operator.Crash,
	operator.Disaster,
	operator.Reboot,
	operator.DiskFull,
}

func isLoadJob(o operator.OType) bool {
	for _, oType := range loadJob {
		if o == oType {
			return true
		}
	}
	return false
}

var renderJob = []operator.OType{
	operator.ScaleIn,
	operator.Kill,
	operator.DataCorrupted,
	operator.Crash,
	operator.Disaster,
	operator.Reboot,
	operator.DiskFull,
	operator.DataDistribution,
	operator.OnlineDDLAddIndex,
}

func isRenderJob(o operator.OType) bool {
	for _, oType := range renderJob {
		if o == oType {
			return true
		}
	}
	return false
}

const resultPath = "./result"

func New(e map[string][]string, s *widgets.Tree) Job {
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
	Ld.IsOver = false
	c, err := comp.New()
	if err != nil {
		panic(err)
	}
	return Job{
		examples:   e,
		selected:   s,
		components: c.Map,
		Channel: Channel{
			BarC:      make(chan int),
			LdC:       make(chan string),
			StopC:     make(chan bool),
			ErrC:      make(chan error),
			CompleteC: make(chan bool),
		},
		resultPath: mkdirResultPath(),
	}
}

const CompleteSignal = "complete_signal"

func (j *Job) Run() {

	oType := j.tp()
	ov := operator.GetOTypeValue(oType)
	j.printSelected(oType)
	if err := resetDB(); err != nil {
		j.ErrC <- err
		return
	}

	// for job internal load, e.g disk_full
	ctx, cancel := context.WithCancel(context.Background())
	// for shell listener goroutine
	shellCtx, shellCancel := context.WithCancel(context.Background())
	go ssh.S.ShellListener(shellCtx)

	defer func() {
		cancel()
		shellCancel()
		time.Sleep(1 * time.Second)
		j.Channel.CompleteC <- true
		log.Logger.Infof("complete at %s.", j.resultPath)
	}()

	switch oType {
	case operator.Script, operator.OtherScript:
		j.runScript()
	case operator.SafetyScript:
		j.runSafety()
	default:
		if err := j.createOTypeResult(); err != nil {
			j.ErrC <- err
			return
		}
		if isLoadJob(oType) {
			if Ld.Cmd != "" {
				ldName := filepath.Join(j.resultPath, "load.log")
				go Ld.run(ldName, j.Channel.ErrC, j.Channel.StopC)
				go Ld.captureLoadLog(ldName, j.ErrC, j.LdC)
				time.Sleep(time.Second * 1)
				cntDown("start executing the test case", Ld.Interval)
			}
		}
		switch oType {
		case operator.DataSeparation:
			j.runDataSeparation()
		case operator.Disaster:
			j.runLabel()
		case operator.LoadDataTPCC, operator.LoadDataImportInto, operator.LoadData, operator.LoadDataSelectIntoOutFile:
			j.runLoadData()
		case operator.DataDistribution:
			j.runDataDistribution()
		case operator.OnlineDDLAddIndex:
			j.runOnlineDDL()
		case operator.InstallSysBench:
			j.runInstallSysBench()
		default:
			j.runComponent(ctx)
		}
	}
	if err := ssh.S.AfterCareShellLog(j.resultPath); err != nil {
		j.ErrC <- err
		return
	}
	if isRenderJob(oType) {
		cntDown("grafana image render", Ld.Interval)
		log.Logger.Debug(fmt.Sprintf("load over status: %v", Ld.IsOver))
		if !Ld.IsOver {
			j.Channel.StopC <- true
		}
		time.Sleep(1 * time.Second)
		if err := j.components[comp.Grafana][0].Render(j.resultPath, ov); err != nil {
			j.ErrC <- err
			return
		}
	}
}

func IsCompleteSignal(err error) bool {
	return err.Error() == CompleteSignal
}

func (j *Job) runScript() {
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
		j.Channel.BarC <- cnt
		return true
	})
}

func (j *Job) runComponent(ctx context.Context) {
	var cnt int
	j.selected.Walk(func(node *widgets.TreeNode) bool {
		cnt++
		e := widget.ChangeToExample(node)
		ov := operator.GetOTypeValue(e.OType)
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
					Ctx:        ctx,
				}
				r, err := b.Build()
				if err != nil {
					log.Logger.Error(err)
					return true
				}
				if err = r.Execute(); err != nil {
					log.Logger.Errorf(failMsg, ov, addr, err.Error())
					return true
				}
			}
		}
		time.Sleep(time.Second * Ld.Sleep)
		j.Channel.BarC <- cnt
		return true
	})
}

func (j *Job) runLabel() {
	kvs := j.components[comp.TiKV]
	j.selected.Walk(func(i *widgets.TreeNode) bool {
		targetLabel := i.Value.String()
		for _, kv := range kvs {
			for _, v := range kv.Labels {
				if targetLabel == v {
					b := operator.Builder{
						Host:  kv.Host,
						Port:  kv.Port,
						OType: operator.Crash,
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
		j.Channel.BarC <- 1
		return true
	})
	cntDown("grafana image render", Ld.Interval)
}

func (j *Job) writeResultFile(name string, len, n int, output []string) {
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

func (j *Job) runSafety() {
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
		j.Channel.BarC <- cnt
		return true
	})

}

func (j *Job) createOTypeResult() error {
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

func (j *Job) tp() operator.OType {
	return j.selected.SelectedNode().Value.(*widget.Example).OType
}

func resetDB() error {
	if _, err := mysql.M.ExecuteSQL("DROP DATABASE IF EXISTS poc"); err != nil {
		return err
	}
	if _, err := mysql.M.ExecuteSQL("CREATE DATABASE poc"); err != nil {
		return err
	}
	return nil
}

func (j *Job) printSelected(oType operator.OType) {
	log.Logger.Info("you selected:")
	ov := operator.GetOTypeValue(oType)
	cnt := 0
	j.selected.Walk(func(node *widgets.TreeNode) bool {
		cnt++
		switch oType {
		case operator.Script, operator.OtherScript, operator.SafetyScript:
			log.Logger.Infof("[%d] %s", cnt, node.Value.String())
		default:
			log.Logger.Infof("[%d] %s_%s", cnt, ov, node.Value.String())
		}
		return true
	})
}

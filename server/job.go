package server

import (
	"fmt"
	"github.com/gizak/termui/v3/widgets"
	"net"
	"os"
	"path/filepath"
	"pictorial/log"
	"pictorial/mysql"
	"pictorial/operator"
	"pictorial/ssh"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

type job struct {
	selected *widgets.Tree
	scripts  map[string]script
	rd       string
	channel
}

type channel struct {
	barC    chan int
	ldC     chan string
	errC    chan error
	finishC chan bool
}

func newJob(ss map[string]script, selected *widgets.Tree) job {
	return job{
		scripts:  ss,
		selected: selected,
		channel: channel{
			barC:    make(chan int),
			ldC:     make(chan string),
			errC:    make(chan error),
			finishC: make(chan bool),
		},
	}
}

const completeSignal = "complete_signal"

func (j *job) run() {
	if err := j.preRun(); err != nil {
		j.errC <- err
		return
	}
	value := j.selected.SelectedNode().Value
	if isDisaster(value.String()) {
		log.Logger.Warn("cluster: %s has no labels, stop", ssh.S.ClusterName)
		return
	}
	switch value.(option).operatorTp {
	case sql, otherSql:
		j.runSQL()
	case disaster:
		j.runLabel()
	case safetySql:
		j.runSafety()
	default:
		j.runNodes()
	}
	log.Logger.Infof("complete at %s", j.rd)
	j.finishC <- true
}

func isCompleteSignal(err error) bool {
	return err.Error() == completeSignal
}

func (j *job) preRun() error {
	j.printSelected()
	if err := j.mkdirRd(); err != nil {
		return err
	}
	return nil
}

func (j *job) runSQL() {
	if err := mysql.M.ResetDB(); err != nil {
		j.errC <- err
	}
	log.Logger.Info("start job, reset DB complete.")
	var cnt int
	j.selected.Walk(func(i *widgets.TreeNode) bool {
		cnt++
		var idx int32
		var wg sync.WaitGroup
		var out string
		name := i.Value.String()
		scripts := j.scripts[i.Value.String()].sql
		for _, sql := range scripts {
			wg.Add(1)
			go func(sql string) {
				defer wg.Done()
				atomic.AddInt32(&idx, 1)
				if err := mysql.M.ExecuteAndWrite(sql, j.rd, name, idx); err != nil {
					out = err.Error()
				}
			}(sql)
		}
		wg.Wait()
		if out != "" {
			log.Logger.Infof("[WARN] %s: %s", name, out)
		} else {
			log.Logger.Infof("[PASS] %s", name)
		}
		j.channel.barC <- cnt
		return true
	})
}

func (j *job) runNodes() {
	result, err := j.mkdirOperatorResult()
	if err != nil {
		j.errC <- err
		return
	}
	if ld.path != "" {
		ldName := filepath.Join(result, "loader.log")
		go ld.run(ldName, j.channel.errC)
		go ld.captureLoaderLog(ldName, j.errC, j.ldC)
		time.Sleep(time.Second * 1)
		cntDown("run items", ld.interval)
	}
	components, err := getComponents()
	if err != nil {
		j.errC <- err
		return
	}

	var cnt int
	j.selected.Walk(func(i *widgets.TreeNode) bool {
		addr := strings.Replace(i.Value.(option).value, "(L)", "", -1)
		componentTp := i.Value.(option).componentTp
		o := i.Value.(option).operatorTp
		operatorTp := getOperatorTp(o)
		var failMsg = "[%s] %s failed: %v"
		for _, c := range components[componentTp] {
			if addr == net.JoinHostPort(c.host, c.port) {
				b := operator.Builder{
					OType:      operatorTp,
					CType:      getComponentTp(componentTp),
					Host:       c.host,
					Port:       c.port,
					DeployPath: c.deployPath,
				}
				r, err := b.Build()
				if err != nil {
					log.Logger.Errorf("unknown operator: %s", b.OType)
				}
				if err = r.Execute(); err != nil {
					log.Logger.Errorf(failMsg, operatorTp, addr, err)
					return false
				}
			}
		}
		cnt++
		j.barC <- cnt
		return true
	})
	if err != nil {
		j.errC <- err
		return
	}
	if ld.path != "" {
		cntDown("render", ld.interval)
		if err := components[grafana][0].render(result); err != nil {
			j.errC <- err
			return
		}
	}
}

func (j *job) runLabel() {
	components, err := getComponents()
	if err != nil {
		j.errC <- err
	}
	kvs := components[tikv]
	j.selected.Walk(func(i *widgets.TreeNode) bool {
		targetLabel := i.Value.String()
		for _, kv := range kvs {
			for _, v := range kv.labels {
				if targetLabel == v {
					b := operator.Builder{
						Host:  kv.host,
						Port:  kv.port,
						OType: getOperatorTp(crash),
						CType: getComponentTp(tikv),
					}
					r, _ := b.Build()
					if err := r.Execute(); err != nil {
						log.Logger.Errorf("[DISASTER] %s failed: %v", net.JoinHostPort(b.Host, b.Port), err)
					}
					break
				}
			}
		}
		log.Logger.Infof("[DISASTER] [%s] complete", targetLabel)
		return true
	})
}

const rootUser = "## root"
const tidbUser = "## tidb_user"
const tidbUserName = "tidb_user"
const tidbUserPassword = "tidb_password"

func (j *job) runSafety() {
	if err := mysql.M.ResetDB(); err != nil {
		j.errC <- err
		return
	}
	var cnt int
	j.selected.Walk(func(i *widgets.TreeNode) bool {
		name := i.Value.String()
		scripts := j.scripts[i.Value.String()].sql
		var err error
		var out string
		for i, sql := range scripts {
			var user string
			if i == 0 {
				user = strings.Split(sql, "\n")[0]
			} else {
				user = strings.Split(sql, "\n")[1]
			}
			sql = strings.Trim(sql, "\n")
			fname := filepath.Join("./result", fmt.Sprintf("%s_%d", name, i))
			switch {
			case strings.Contains(user, rootUser):
				sql = strings.ReplaceAll(sql, rootUser, "")
				err = mysql.M.ExecuteUserAndWrite(sql, fname, "root", "")
			case strings.Contains(user, tidbUser):
				sql = strings.ReplaceAll(sql, tidbUser, "")
				err = mysql.M.ExecuteUserAndWrite(sql, fname, tidbUserName, tidbUserPassword)
			default:
				err = fmt.Errorf("invalid username, please use 'root' and 'tidb_user'")
			}
			if err != nil {
				out = err.Error()
			}
		}
		if out != "" {
			log.Logger.Infof("[WARN] %s: %s", name, out)
		} else {
			log.Logger.Infof("[PASS] %s", name)
		}
		cnt++
		j.channel.barC <- cnt
		return true
	})
	c := "SET GLOBAL validate_password.enable = OFF;"
	if _, err := mysql.M.ExecuteSQL(c); err != nil {
		log.Logger.Warnf("recover database: %s failed: %s", c, err.Error())
	}
}

func (j *job) printSelected() {
	log.Logger.Info("you select: ")
	j.selected.Walk(func(i *widgets.TreeNode) bool {
		log.Logger.Infof("	[%s]", i.Value.String())
		return true
	})
}

const rd = "result"

func (j *job) mkdirRd() error {
	var err error
	if err = os.MkdirAll(rd, os.ModePerm); err != nil {
		return err
	}
	if j.rd, err = filepath.Abs(rd); err != nil {
		return err
	}
	return nil
}

func dateFormat() string {
	now := time.Now()
	year, month, day := now.Date()
	hour, min, sec := now.Clock()
	return fmt.Sprintf("%d-%02d-%02d_%02d:%02d:%02d", year, int(month), day, hour, min, sec)
}

func (j *job) mkdirOperatorResult() (string, error) {
	result := fmt.Sprintf("./result/%s_%s", j.selected.Title, dateFormat())
	if err := os.MkdirAll(result, os.ModePerm); err != nil {
		return "", err
	}
	return filepath.Abs(result)
}

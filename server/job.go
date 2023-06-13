package server

import (
	"fmt"
	"github.com/gizak/termui/v3/widgets"
	"net"
	"os"
	"path/filepath"
	"pictorial/comp"
	"pictorial/log"
	"pictorial/mysql"
	"pictorial/operator"
	"pictorial/widget"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

type job struct {
	list       *widgets.Tree
	examples   map[string][]string
	components map[comp.CType][]comp.Component
	channel
}

type channel struct {
	barC      chan int
	ldC       chan string
	errC      chan error
	completeC chan bool
}

const rd = "result"

func newJob(e map[string][]string, s *widgets.Tree, cs map[comp.CType][]comp.Component) job {
	return job{
		examples:   e,
		list:       s,
		components: cs,
		channel: channel{
			barC:      make(chan int),
			ldC:       make(chan string),
			errC:      make(chan error),
			completeC: make(chan bool),
		},
	}
}

const completeSignal = "complete_signal"

func (j *job) run() {
	job := j.list.SelectedNode().Value.(*widget.Example)
	switch job.OType {
	case widget.Script, widget.OtherScript:
		j.runScript()
	case widget.Disaster:
		j.runLabel()
	case widget.SafetyScript:
		j.runSafety()
	default:
		j.runComponent()
	}
	log.Logger.Infof("complete at %s", rd)
	j.completeC <- true
}

func isCompleteSignal(err error) bool {
	return err.Error() == completeSignal
}

func (j *job) runScript() {
	if err := mysql.M.ResetDB(); err != nil {
		j.errC <- err
	}
	log.Logger.Info("start job, reset DB complete.")
	var cnt int
	j.list.Walk(func(i *widgets.TreeNode) bool {
		cnt++
		var idx int32
		var wg sync.WaitGroup
		var out string
		name := i.Value.String()
		scripts := j.examples[i.Value.String()]
		for _, s := range scripts {
			wg.Add(1)
			go func(sql string) {
				defer wg.Done()
				atomic.AddInt32(&idx, 1)
				if err := mysql.M.ExecuteAndWrite(sql, rd, name, idx); err != nil {
					out = err.Error()
				}
			}(s)
		}
		wg.Wait()
		if out != "" {
			log.Logger.Infof("[warn] %s: %s", name, out)
		} else {
			log.Logger.Infof("[pass] %s", name)
		}
		j.channel.barC <- cnt
		return true
	})
}

func (j *job) runComponent() {
	result, err := j.createOTypeResult()
	if err != nil {
		j.errC <- err
		return
	}
	if ld.cmd != "" {
		ldName := filepath.Join(result, "loader.log")
		go ld.run(ldName, j.channel.errC)
		go ld.captureLoaderLog(ldName, j.errC, j.ldC)
		time.Sleep(time.Second * 1)
		cntDown("run items", ld.interval)
	}
	var cnt int
	j.list.Walk(func(i *widgets.TreeNode) bool {
		cnt++
		j.barC <- cnt
		e := widget.ChangeToExample(i)
		addr := strings.Replace(e.String(), "(L)", "", -1)
		o := i.Value.(*widget.Example).OType
		oType := widget.GetOTypeValue(o)
		var failMsg = "[%s] %s failed: %v"
		for _, c := range j.components[e.CType] {
			if addr == net.JoinHostPort(c.Host, c.Port) {
				b := operator.Builder{
					OType:      oType,
					CType:      comp.GetCTypeValue(e.CType),
					Host:       c.Host,
					Port:       c.Port,
					DeployPath: c.DeployPath,
				}
				r, err := b.Build()
				if err != nil {
					log.Logger.Error(err)
					return true
				}
				if err = r.Execute(); err != nil {
					log.Logger.Errorf(failMsg, oType, addr, err)
					return true
				}
			}
		}
		time.Sleep(time.Second * time.Duration(ld.sleep))
		return true
	})
	if ld.cmd != "" {
		cntDown("render", ld.interval)
		if err := j.components[comp.Grafana][0].Render(result); err != nil {
			j.errC <- err
			return
		}
	}
}

func (j *job) runLabel() {
	result, err := j.createOTypeResult()
	if err != nil {
		j.errC <- err
		return
	}
	if ld.cmd != "" {
		ldName := filepath.Join(result, "loader.log")
		go ld.run(ldName, j.channel.errC)
		go ld.captureLoaderLog(ldName, j.errC, j.ldC)
		time.Sleep(time.Second * 1)
		cntDown("run items", ld.interval)
	}
	kvs := j.components[comp.TiKV]
	j.list.Walk(func(i *widgets.TreeNode) bool {
		targetLabel := i.Value.String()
		for _, kv := range kvs {
			for _, v := range kv.Labels {
				if targetLabel == v {
					b := operator.Builder{
						Host:  kv.Host,
						Port:  kv.Port,
						OType: widget.GetOTypeValue(widget.Crash),
						CType: comp.GetCTypeValue(comp.TiKV),
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
	if ld.cmd != "" {
		cntDown("render", ld.interval)
		if err := j.components[comp.Grafana][0].Render(result); err != nil {
			j.errC <- err
			return
		}
	}
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
	j.list.Walk(func(i *widgets.TreeNode) bool {
		name := i.Value.String()
		scripts := j.examples[i.Value.String()]
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
			fname := filepath.Join(rd, fmt.Sprintf("%s_%d", name, i))
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
			log.Logger.Infof("[warn] %s: %s", name, out)
		} else {
			log.Logger.Infof("[pass] %s", name)
		}
		cnt++
		j.channel.barC <- cnt
		return true
	})

}

func dateFormat() string {
	now := time.Now()
	year, month, day := now.Date()
	hour, min, sec := now.Clock()
	return fmt.Sprintf("%d-%02d-%02d_%02d:%02d:%02d", year, int(month), day, hour, min, sec)
}

func (j *job) createOTypeResult() (string, error) {
	result := fmt.Sprintf("%s/%s_%s", rd, j.list.Title, dateFormat())
	if err := os.MkdirAll(result, os.ModePerm); err != nil {
		return "", err
	}
	return filepath.Abs(result)
}

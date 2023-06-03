package server

import (
	"fmt"
	"github.com/gizak/termui/v3/widgets"
	"net"
	"os"
	"path/filepath"
	"pictorial/log"
	"pictorial/mysql"
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
	barC chan int
	ldC  chan string
	errC chan error
}

func newJob(ss map[string]script, selected *widgets.Tree) job {
	return job{
		scripts:  ss,
		selected: selected,
		channel: channel{
			barC: make(chan int),
			ldC:  make(chan string),
			errC: make(chan error),
		},
	}
}

func (j *job) run() {
	if err := j.preRun(); err != nil {
		j.errC <- err
		return
	}
	value := j.selected.SelectedNode().Value
	if isDisaster(value.String()) {
		log.Logger.Warn("this cluster has no label, stop")
		return
	}
	switch value.(item).action {
	case sql, other:
		j.runSQL()
	case disaster:
		j.runLabel()
	default:
		j.runNodes()
	}
	log.Logger.Info("complete, you can quit now.")
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
	if ld.path != "" {
		ldName := filepath.Join(j.rd, "loader.log")
		go ld.run(ldName, j.channel.errC)
		go ld.captureLoaderLog(ldName, j.errC, j.ldC)
		time.Sleep(time.Second * 1)
		cntDown("run items", ld.interval)
	}
	nodes, err := getNodes()
	if err != nil {
		j.errC <- err
		return
	}
	var cnt int
	j.selected.Walk(func(i *widgets.TreeNode) bool {
		addr := strings.Replace(i.Value.(item).value, "(L)", "", -1)
		tp := i.Value.(item).tp
		ac := i.Value.(item).action
		var failMsg = "[%s] %s failed: %v"
		for _, node := range nodes[tp] {
			if addr == net.JoinHostPort(node.host, node.port) {
				switch ac {
				case kill:
					err = node.kill()
				case dataCorrupted:
					err = node.dataCorrupted()
				case crash:
					err = node.systemd(crash)
				case recoverSystemd:
					err = node.systemd(recoverSystemd)
				}
				if err != nil {
					log.Logger.Errorf(failMsg, getAction(ac), addr, err)
				}
				break
			}
		}
		cnt++
		j.barC <- cnt
		return true
	})
	if ld.path != "" {
		cntDown("render", ld.interval)
		g := nodes["grafana"][0]
		if err := g.render(j.rd); err != nil {
			j.errC <- err
			return
		}
	}
}

func (j *job) runLabel() {
	nodes, err := getNodes()
	if err != nil {
		j.errC <- err
	}
	kvs := nodes["tikv"]
	j.selected.Walk(func(i *widgets.TreeNode) bool {
		targetLabel := i.Value.String()
		for _, kv := range kvs {
			for _, v := range kv.labels {
				if targetLabel == v {
					n := node{
						host:       kv.host,
						port:       kv.port,
						tp:         tikv,
						deployPath: kv.deployPath,
					}
					switch i.Value.(item).action {
					case disaster:
						if err := n.systemd(crash); err != nil {
							log.Logger.Errorf("[DISASTER] %s failed: %v", net.JoinHostPort(n.host, n.port), err)
						}
					}
					break
				}
			}
		}
		return true
	})
}

func (j *job) printSelected() {
	log.Logger.Info("start job, you selected: ")
	j.selected.Walk(func(i *widgets.TreeNode) bool {
		log.Logger.Infof("	[%s]", i.Value.String())
		return true
	})
}

func (j *job) mkdirRd() error {
	timeStr := dateFormat()
	var err error
	if err = os.MkdirAll(timeStr, os.ModePerm); err != nil {
		return err
	}
	if j.rd, err = filepath.Abs(timeStr); err != nil {
		return err
	}
	log.Logger.Info(j.rd)
	return nil
}

func dateFormat() string {
	now := time.Now()
	year, month, day := now.Date()
	hour, min, sec := now.Clock()
	return fmt.Sprintf("%d-%02d-%02d_%02d:%02d:%02d", year, int(month), day, hour, min, sec)
}

package server

import (
	"flag"
	"fmt"
	ui "github.com/gizak/termui/v3"
	"github.com/pelletier/go-toml"
	"github.com/sirupsen/logrus"
	"pictorial/log"
	"pictorial/mysql"
	"pictorial/ssh"
)

type Server struct {
	w *widget
}

func New() error {

	if err := parse(); err != nil {
		return err
	}
	if err := ui.Init(); err != nil {
		return err
	}
	defer ui.Close()
	x, y := ui.TerminalDimensions()
	svr := Server{
		w: &widget{
			tree:       newTree("tree", 0, 0, x/3, y/2),
			selected:   newTree("", x/3, 0, 2*x/3, y/2),
			lg:         newList(nil, "log", 0, int(float64(y)*0.93), 2*x/3, int((float64(y)/10)*5)),
			processBar: newGauge("processBar", 0, int(float64(y)*0.94), 2*x/3, y-1),
			loader:     newList(nil, "loader", 2*x/3, 0, x, y-1),
		},
	}

	go svr.captureLog()

	if err := svr.w.setTreeNodes(); err != nil {
		return err
	}
	ui.Render(
		svr.w.tree,
		svr.w.selected,
		svr.w.lg,
		svr.w.processBar,
		svr.w.loader,
	)

	ue := ui.PollEvents()
	for {
		e := <-ue
		switch e.ID {
		case up:
			svr.w.tree.ScrollUp()
		case down:
			svr.w.tree.ScrollDown()
		case left:
			svr.w.tree.Collapse()
		case right:
			svr.w.right()
		case sUp:
			svr.w.selected.ScrollUp()
		case sDown:
			svr.w.selected.ScrollDown()
		case sDelete:
			svr.w.removeSelected()
		case enter:
			if svr.w.selected == nil || svr.w.treeLength(selected) == 0 {
				log.Logger.Warnf("selected is 0")
			} else {
				if err := svr.run(); err != nil {
					if !isCompleteSignal(err) {
						return err
					}
					continue
				}
			}
		case ctrlC:
			return nil
		}
		ui.Render(
			svr.w.tree,
			svr.w.selected,
			svr.w.processBar,
		)
	}
}

func (svr *Server) run() error {

	ss, err := svr.w.walkTreeScript()
	if err != nil {
		return err
	}
	j := newJob(ss, svr.w.selected)
	svr.w.selected.ScrollTop()
	ui.Render(
		svr.w.selected,
	)
	go j.run()
	ue := ui.PollEvents()
	for {
		select {
		case e := <-ue:
			if e.ID == ctrlC {
				return nil
			}
		case err := <-j.errC:
			log.Logger.Error(err)
		case idx := <-j.barC:
			svr.w.refresh(idx)
		case ldText := <-j.ldC:
			svr.w.loader.Rows = append(svr.w.loader.Rows, ldText)
			svr.w.loader.ScrollDown()
			ui.Render(svr.w.loader)
		case <-j.finishC:
			return fmt.Errorf(completeSignal)
		}
	}
}

func parse() error {
	var c string
	flag.StringVar(&c, "c", "./config.toml", "")
	flag.Parse()
	config, err := toml.LoadFile(c)
	if err != nil {
		return err
	}

	for k, v := range config.Values() {
		log.Logger.Infof(" %s == %s", k, v)
	}

	mysql.M.Host = config.Get("mysql.host").(string)
	mysql.M.Port = config.Get("mysql.port").(string)
	mysql.M.User = config.Get("mysql.user").(string)
	mysql.M.Password = config.Get("mysql.password").(string)

	ssh.S.User = config.Get("ssh.user").(string)
	ssh.S.Password = config.Get("ssh.password").(string)
	ssh.S.SshPort = config.Get("ssh.sshPort").(string)

	ssh.S.ClusterName = config.Get("cluster.name").(string)
	ssh.S.Carry.Plugin = config.Get("grafana.plugin").(string)
	if err := ssh.S.ApplySSHKey(); err != nil {
		return err
	}
	ld.path = config.Get("loader.path").(string)
	ld.interval = config.Get("loader.interval").(int64)
	others = config.Get("other.dir").(string)

	logLevel := config.Get("log.level").(string)
	switch logLevel {
	case "debug":
		log.Logger.SetLevel(logrus.DebugLevel)
	}
	return nil
}

func (svr *Server) captureLog() {
	t, err := log.Tail(log.Name)
	if err != nil {
		panic(err)
	}
	for l := range t.Lines {
		svr.w.lg.Rows = append(svr.w.lg.Rows, l.Text)
		svr.w.lg.ScrollBottom()
		ui.Render(
			svr.w.lg,
		)
	}
}

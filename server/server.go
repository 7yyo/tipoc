package server

import (
	"flag"
	"fmt"
	ui "github.com/gizak/termui/v3"
	"github.com/pelletier/go-toml"
	"github.com/sirupsen/logrus"
	"pictorial/comp"
	"pictorial/log"
	"pictorial/mysql"
	"pictorial/ssh"
	"pictorial/widget"
	"time"
)

const logName = "output.log"

type Server struct {
	w *widget.Widget
}

func New() {

	log.InitLogger(logName)

	if err := ui.Init(); err != nil {
		panic(err)
	}
	defer ui.Close()

	if err := initConfig(); err != nil {
		panic(err)
	}

	s := Server{
		w: &widget.Widget{
			T: widget.CreateTree(),
			C: widget.NewChosen(),
			O: widget.NewOutput(),
			P: widget.NewProcessBar(),
			L: widget.NewLoader(),
		},
	}

	go s.captureLog()

	ui.Render(s.w.T, s.w.C, s.w.O, s.w.P, s.w.L)

	ue := ui.PollEvents()
	for {
		e := <-ue
		switch e.ID {
		case "<Up>":
			s.w.T.ScrollUp()
		case "<Down>":
			s.w.T.ScrollDown()
		case "<Left>":
			s.w.T.Collapse()
		case "<Right>":
			s.w.ScrollRight()
		case ",":
			s.w.C.ScrollUp()
		case ".":
			s.w.C.ScrollDown()
		case "<Backspace>":
			s.w.ScrollBackSpace()
		case "<Enter>":
			if widget.TreeLength(s.w.C) == 0 {
				log.Logger.Warnf("selected is empty.")
			} else {
				widget.ScrollTopTree(s.w.C)
				if err := s.run(); err != nil {
					if !isCompleteSignal(err) {
						return
					}
					continue
				}
			}
		case "<C-c>":
			return
		}
		ui.Render(s.w.T, s.w.C, s.w.P)
	}
}

func (s *Server) run() error {

	examples, err := s.w.WalkTreeScript()
	if err != nil {
		return err
	}

	cs, err := comp.New()
	if err != nil {
		return err
	}

	j := newJob(examples, s.w.C, cs.Map)
	go j.run()

	ue := ui.PollEvents()
	for {
		select {
		case e := <-ue:
			if e.ID == "<C-c>" {
				return nil
			}
		case err := <-j.errC:
			log.Logger.Error(err)
		case idx := <-j.barC:
			s.w.RefreshProcessBar(idx)
		case ldText := <-j.ldC:
			s.w.L.Rows = append(s.w.L.Rows, ldText)
			s.w.L.ScrollDown()
			ui.Render(s.w.L)
		case <-j.completeC:
			widget.CleanTree(s.w.C)
			return fmt.Errorf(completeSignal)
		}
	}
}

const defaultCfg = "config.toml"

func initConfig() error {

	var cfg string
	flag.StringVar(&cfg, "c", defaultCfg, "")
	flag.Parse()

	config, err := toml.LoadFile(cfg)
	if err != nil {
		return err
	}

	for k, v := range config.Values() {
		log.Logger.Infof("%s == %s", k, v)
	}

	mysql.M.Host = config.Get("mysql.host").(string)
	mysql.M.Port = config.Get("mysql.port").(string)
	mysql.M.User = config.Get("mysql.user").(string)
	mysql.M.Password = config.Get("mysql.password").(string)

	comp.PdAddr, err = mysql.M.GetPdAddr()
	if err != nil {
		return err
	}

	ssh.S.User = config.Get("ssh.user").(string)
	ssh.S.SshPort = config.Get("ssh.sshPort").(string)
	ssh.S.Cluster.Name = config.Get("cluster.name").(string)
	if err := ssh.S.CheckClusterName(); err != nil {
		return err
	}
	ssh.S.Cluster.Plugin = config.Get("cluster.plugin").(string)
	if err := ssh.S.GetSSHKey(); err != nil {
		return err
	}
	ld.cmd = config.Get("load.cmd").(string)
	ld.interval = config.Get("load.interval").(int64)
	ld.sleep = time.Duration(config.Get("load.sleep").(int64))

	logLevel := config.Get("log.level").(string)
	switch logLevel {
	case "debug":
		log.Logger.SetLevel(logrus.DebugLevel)
	}

	widget.OtherConfig = config.Get("other.dir").(string)
	return nil
}

func (s *Server) captureLog() {
	t, err := log.Track(logName)
	if err != nil {
		panic(err)
	}
	for l := range t.Lines {
		s.w.O.Rows = append(s.w.O.Rows, l.Text)
		s.w.O.ScrollBottom()
		ui.Render(s.w.O)
	}
}

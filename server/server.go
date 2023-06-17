package server

import (
	"fmt"
	ui "github.com/gizak/termui/v3"
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

	if err := prepare(); err != nil {
		panic(err)
	}

	if err := ui.Init(); err != nil {
		panic(err)
	}
	defer ui.Close()

	s := Server{
		w: &widget.Widget{
			T: widget.CreateTree(),
			C: widget.NewChosen(),
			O: widget.NewOutput(),
			P: widget.NewProcessBar(),
			L: widget.NewLoader(),
		},
	}
	ui.Render(s.w.T, s.w.C, s.w.O, s.w.P, s.w.L)

	go s.captureLog()

	previousKey := ""
	ue := ui.PollEvents()
	for {
		e := <-ue
		switch e.ID {
		case "a":
			if previousKey == "a" {
				s.w.AppendAllScripts()
			}
		case "g":
			if previousKey == "g" {
				s.w.T.ScrollTop()
			}
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
		if previousKey == "g" || previousKey == "a" {
			previousKey = ""
		} else {
			previousKey = e.ID
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
			s.w.AutoScrollDownLoad(ldText)
		case <-j.completeC:
			widget.CleanTree(s.w.C)
			return fmt.Errorf(completeSignal)
		}
	}
}

func prepare() error {

	log.InitLogger(logName)

	config, err := parseC()
	if err != nil {
		return err
	}
	if err := checkConfig(config); err != nil {
		return err
	}

	for k, v := range config.Values() {
		log.Logger.Infof("%s == %s", k, v)
	}

	mysql.M.Host = config.Get(mysqlHost).(string)
	mysql.M.Port = config.Get(mysqlPort).(string)
	mysql.M.User = config.Get(mysqlUser).(string)
	mysql.M.Password = config.Get(mysqlPassword).(string)

	comp.PdAddr, err = comp.GetPdAddr()
	if err != nil {
		return err
	}

	ssh.S.User = config.Get(sshUser).(string)
	if config.Get(sshPassword) != nil {
		ssh.S.Password = config.Get(sshPassword).(string)
	}
	ssh.S.SshPort = config.Get(sshPort).(string)
	ssh.S.Cluster.Name = config.Get(clusterName).(string)
	ssh.S.LogC = make(chan string)
	if err := ssh.S.CheckClusterName(); err != nil {
		return err
	}
	if config.Get(plugin) != nil {
		ssh.S.Cluster.Plugin = config.Get(plugin).(string)
	}
	if err := ssh.S.AddSSHKey(); err != nil {
		return err
	}
	go ssh.S.CommandListener()

	if config.Get(loadCmd) != nil {
		ld.cmd = config.Get(loadCmd).(string)
	}
	if config.Get(loadInterval) != nil {
		ld.interval = config.Get(loadInterval).(int64)
	}
	if config.Get(loadSleep) != nil {
		ld.sleep = time.Duration(config.Get(loadSleep).(int64))
	}

	if config.Get(logLevel) != nil {
		logLevel := config.Get(logLevel).(string)
		switch logLevel {
		case "debug":
			log.Logger.SetLevel(logrus.DebugLevel)
		}
	}
	if config.Get(widget.OtherConfig) != nil {
		widget.OtherConfig = config.Get(otherDir).(string)
	}
	return nil
}

func (s *Server) captureLog() {
	t, err := log.Track(logName)
	if err != nil {
		panic(err)
	}
	for l := range t.Lines {
		s.w.AutoScrollDownOutput(l.Text)
	}
}

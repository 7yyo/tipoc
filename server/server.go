package server

import (
	"fmt"
	ui "github.com/gizak/termui/v3"
	"pictorial/log"
	"pictorial/server/job"
	"pictorial/widget"
)

const outputLog = "output.log"

type Server struct {
	w *widget.Widget
}

func New() {

	log.New(outputLog)

	if err := ui.Init(); err != nil {
		panic(err)
	}
	defer ui.Close()

	s := Server{
		w: &widget.Widget{
			O: widget.NewOutput(),
		},
	}
	go s.captureLog()
	ui.Render(s.w.O)

	ue := ui.PollEvents()
	jump := func() error {
		for {
			e := <-ue
			switch e.ID {
			case widget.KeyCtrlC:
				return nil
			}
		}
	}

	if err := prepare(); err != nil {
		log.Logger.Error(err)
		if jump() == nil {
			return
		}
	}
	tree, err := widget.BuildTree()
	if err != nil {
		log.Logger.Error(err)
		if jump() == nil {
			return
		}
	}

	s.w.T = tree
	s.w.S = widget.NewSelected()
	s.w.L = widget.NewLoad()
	s.w.P = widget.NewProcessBar()
	ui.Render(s.w.T, s.w.S, s.w.O, s.w.L, s.w.P)

	previousKey := ""
	for {
		e := <-ue
		switch e.ID {
		case widget.KeySelectAll:
			if previousKey == widget.KeySelectAll {
				s.w.AppendAllScripts()
			}
		case widget.KeyScrollTop:
			if previousKey == widget.KeyScrollTop {
				s.w.T.ScrollTop()
			}
		case widget.KeyCollapseAll:
			if previousKey == widget.KeyCollapseAll {
				s.w.T.CollapseAll()
				s.w.T.ScrollTop()
			}
		case widget.KeyRemoveAll:
			if previousKey == widget.KeyRemoveAll {
				widget.CleanTree(s.w.S)
				s.w.S.ScrollTop()
			}
		case widget.KeyArrowUp:
			s.w.T.ScrollUp()
		case widget.KeyArrowDown:
			s.w.T.ScrollDown()
		case widget.KeyArrowLeft:
			s.w.T.Collapse()
		case widget.KeyArrowRight:
			s.w.ScrollRight()
		case widget.KeyComma:
			s.w.S.ScrollUp()
		case widget.KeyPeriod:
			s.w.S.ScrollDown()
		case widget.KeyBackSpace:
			s.w.ScrollBackSpace()
		case widget.KeyEnter:
			if widget.TreeLength(s.w.S) == 0 {
				log.Logger.Warnf("selected is empty.")
			} else {
				widget.ScrollTopTree(s.w.S)
				if err := s.run(); err != nil {
					if !job.IsCompleteSignal(err) {
						return
					}
					continue
				}
			}
		case widget.KeyCtrlC:
			return
		}
		if previousKey == widget.KeyScrollTop ||
			previousKey == widget.KeySelectAll ||
			previousKey == widget.KeyCollapseAll ||
			previousKey == widget.KeyRemoveAll {
			previousKey = ""
		} else {
			previousKey = e.ID
		}

		ui.Render(s.w.T, s.w.S, s.w.P)
	}
}

func (s *Server) run() error {

	examples, err := s.w.WalkTreeScript()
	if err != nil {
		return err
	}

	j := job.New(examples, s.w.S)
	go j.Run()

	ue := ui.PollEvents()
	for {
		select {
		case e := <-ue:
			if e.ID == "<C-c>" {
				return nil
			}
		case err := <-j.Channel.ErrC:
			log.Logger.Error(err)
		case idx := <-j.Channel.BarC:
			s.w.RefreshProcessBar(idx)
		case ldText := <-j.Channel.LdC:
			s.w.PrintLoad(ldText)
		case <-j.Channel.CompleteC:
			widget.CleanTree(s.w.S)
			return fmt.Errorf(job.CompleteSignal)
		}
	}
}

func prepare() error {
	cfg, err := parseFlag()
	if err != nil {
		return err
	}
	if err := initConfig(cfg); err != nil {
		return err
	}
	return nil
}

func (s *Server) captureLog() {
	t, err := log.Track(outputLog)
	if err != nil {
		panic(err)
	}
	for l := range t.Lines {
		s.w.PrintOutput(l.Text)
	}
}

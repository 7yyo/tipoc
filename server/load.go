package server

import (
	"pictorial/log"
	"pictorial/ssh"
	"time"
)

type load struct {
	cmd      string
	interval int64
	sleep    time.Duration
}

var ld load

func (l *load) run(lgName string, errC chan error) {
	args := []string{"-c", l.cmd}
	if _, err := ssh.S.RunLocalWithWrite("sh", args, lgName); err != nil {
		errC <- err
	}
}

func (l *load) captureLoadLog(name string, errC chan error, ldC chan string) {
	time.Sleep(1 * time.Second)
	t, err := log.Track(name)
	if err != nil {
		errC <- err
	}
	for l := range t.Lines {
		ldC <- l.Text
	}
}

func cntDown(msg string, cnt int64) {
	if cnt == 0 {
		return
	}
	log.Logger.Infof("%s in %d minutes.", msg, cnt)
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			cnt--
			if cnt == 0 {
				return
			}
			log.Logger.Infof("%s in %d minutes.", msg, cnt)
		}
	}
}

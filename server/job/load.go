package job

import (
	"context"
	"fmt"
	"pictorial/log"
	"pictorial/ssh"
	"time"
)

type Load struct {
	Cmd      string
	Interval int64
	Sleep    time.Duration
	IsOver   bool
}

var Ld Load

func (l *Load) run(lgName string, errC chan error, stopLdC chan bool) {
	log.Logger.Infof("start load: %s", l.Cmd)
	action := "load ends and exits normally"
	args := []string{"-c", l.Cmd}
	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		<-stopLdC
		action = fmt.Sprintf("receive kill signal, cancel normally: %s.", l.Cmd)
		cancel()
	}()
	if _, err := ssh.S.RunLocalWithContext(ctx, "sh", args, lgName); err != nil {
		errC <- err
	}
	log.Logger.Info(action)
	l.IsOver = true
}

func (l *Load) captureLoadLog(name string, errC chan error, ldC chan string) {
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
	log.Logger.Infof("%s after %d minutes...", msg, cnt)
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			cnt--
			if cnt == 0 {
				return
			}
			log.Logger.Infof("after %d minutes...", cnt)
		}
	}
}

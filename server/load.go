package server

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"os/exec"
	"pictorial/log"
	"time"
)

type load struct {
	cmd      string
	interval int64
	sleep    time.Duration
}

var ld load

func (l *load) run(lgName string, errC chan error) {
	lf, err := os.Create(lgName)
	if err != nil {
		errC <- err
	}
	defer lf.Close()
	log.Logger.Info(l.cmd)
	cmd := exec.Command("sh", "-c", l.cmd)
	cmd.Stdout = io.MultiWriter(lf)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err = cmd.Run(); err != nil {
		errC <- fmt.Errorf("[load] failed: %v: %s", err, stderr.String())
	}
}

func (l *load) captureLoaderLog(name string, errC chan error, ldC chan string) {
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

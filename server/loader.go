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

type loader struct {
	path     string
	interval int64
}

var ld loader

func (l *loader) run(lgName string, errC chan error) {
	lf, err := os.Create(lgName)
	if err != nil {
		errC <- err
	}
	defer lf.Close()
	log.Logger.Info(l.path)
	cmd := exec.Command("sh", "-c", l.path)
	cmd.Stdout = io.MultiWriter(lf)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err = cmd.Run(); err != nil {
		errC <- fmt.Errorf("[LOADER] failed: %v: %s", err, stderr.String())
	}
}

func (l *loader) captureLoaderLog(name string, errC chan error, ldC chan string) {
	time.Sleep(1 * time.Second)
	t, err := log.Tail(name)
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

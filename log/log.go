package log

import (
	"bytes"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/hpcloud/tail"
	"github.com/sirupsen/logrus"
)

var Logger = logrus.New()

func InitLogger(name string) {
	if err := os.Remove(name); err != nil {
		if os.IsExist(err) {
			panic(err)
		}
	}
	f, err := os.OpenFile(name, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		panic(err)
	}
	Logger.SetOutput(f)
	Logger.SetLevel(logrus.InfoLevel)
	Logger.SetFormatter(&customFormatter{})
}

type customFormatter struct{}

func (f *customFormatter) Format(entry *logrus.Entry) ([]byte, error) {
	b := &bytes.Buffer{}
	timestamp := time.Now().Format("01-02 15:04:05.00")
	b.WriteString(fmt.Sprintf("[%s] ", timestamp))
	level := strings.ToUpper(entry.Level.String())
	b.WriteString(fmt.Sprintf("[%s] ", level))
	msg := entry.Message
	b.WriteString(fmt.Sprintf("%q\n", msg))
	return b.Bytes(), nil
}

func Track(logName string) (*tail.Tail, error) {
	return tail.TailFile(logName, tail.Config{
		ReOpen: true,
		Follow: true,
		Poll:   false,
	})
}

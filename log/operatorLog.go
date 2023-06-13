package log

import (
	"bytes"
	"fmt"
	"github.com/sirupsen/logrus"
	"os"
	"path/filepath"
	"time"
)

var OLogger = logrus.New()

func CreateOperatorLog(result string) {
	log := filepath.Join(result, "o.log")
	f, err := os.OpenFile(log, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		panic(err)
	}
	OLogger.SetOutput(f)
	OLogger.SetLevel(logrus.InfoLevel)
	OLogger.SetFormatter(&operatorFormatter{})
}

type operatorFormatter struct{}

func (o *operatorFormatter) Format(entry *logrus.Entry) ([]byte, error) {
	b := &bytes.Buffer{}
	timestamp := time.Now().Format("01-02 15:04:05.00")
	b.WriteString(fmt.Sprintf("[%s] ", timestamp))
	msg := entry.Message
	b.WriteString(fmt.Sprintf("%q\n", msg))
	return b.Bytes(), nil
}

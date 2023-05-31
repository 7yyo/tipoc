package mysql

import (
	"bytes"
	"fmt"
	"github.com/go-mysql-org/go-mysql/client"
	"github.com/go-mysql-org/go-mysql/mysql"
	"io"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"pictorial/log"
	"regexp"
	"strings"
)

type MySQL struct {
	User     string
	Password string
	Host     string
	Port     string
}

var M MySQL

func (m *MySQL) ExecuteSQL(s string) (*mysql.Result, error) {
	addr := net.JoinHostPort(m.Host, m.Port)
	conn, err := client.Connect(addr, m.User, m.Password, "")
	if err != nil {
		return nil, err
	}
	defer conn.Close()
	return conn.Execute(s)
}

func (m *MySQL) args(sql string) []string {
	args := []string{
		fmt.Sprintf("-u%s", m.User),
		fmt.Sprintf("-p%s", m.Password),
		fmt.Sprintf("-h%s", m.Host),
		fmt.Sprintf("-P%s", m.Port),
		fmt.Sprintf("-e %s", sql),
		"-vvv",
		"--comments",
	}
	if m.Password == "" {
		args = append(args, "--skip-password")
	}
	return args
}

func (m *MySQL) ResetDB() error {
	if _, err := m.ExecuteSQL("DROP DATABASE IF EXISTS poc"); err != nil {
		return err
	}
	if _, err := m.ExecuteSQL("CREATE DATABASE poc"); err != nil {
		return err
	}
	log.Logger.Info("reset DB.")
	return nil
}

func (m *MySQL) ExecuteAndWrite(s, rd, name string, idx int32) error {
	var stdout, stderr bytes.Buffer
	var err error
	cmdArgs := m.args(s)
	cmd := exec.Command("mysql", cmdArgs...)
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("%s: %s", stdout.String(), err.Error())
	}
	var fName string
	fName = filepath.Join(rd, fmt.Sprintf("%s_%d.result", name, idx))
	f, _ := os.Create(fName)
	_, err = io.WriteString(f, processResult(stdout.String()))
	if err != nil {
		return err
	}
	_, err = io.WriteString(f, processError(stderr.String()))
	if err != nil {
		return err
	}
	if stderr.String() != "" && stderr.String() != SqlWarn {
		if err != nil {
			return fmt.Errorf("%s: %s", processError(stderr.String()), err.Error())
		} else {
			return fmt.Errorf("%s", processError(stderr.String()))
		}
	}
	return nil
}

const resultLine = "--------------"
const bye = "Bye"
const SqlWarn = "mysql: [Warning] Using a password on the command line interface can be insecure."

func isQueryOutput(input string) bool {
	re1 := regexp.MustCompile(`^Query OK, \d+ row(s)? affected \([\d.]+ (sec|ms)\)$`)
	re2 := regexp.MustCompile(`^Query OK, \d+ rows affected, \d+ warning(s)? \([\d.]+ (sec|ms)\)$`)
	re3 := regexp.MustCompile(`^\d+ row(s)? in set \([\d.]+ (sec|ms)\)$`)
	re4 := regexp.MustCompile(`^(\d+) row(s)? in set \((\d+) (min|sec)( \d+\.\d+ sec)?\)$`)
	re5 := regexp.MustCompile(`(\d+) row affected, (\d+) warning`)
	re6 := regexp.MustCompile(`^\d+ row(s) in set, \d+ warning \(\d+\.\d+ sec\)$`)
	return re1.MatchString(input) ||
		re2.MatchString(input) ||
		re3.MatchString(input) ||
		re4.MatchString(input) ||
		re5.MatchString(input) ||
		re6.MatchString(input) ||
		strings.HasPrefix("ERROR", input)
}

func isDMLResultOutput(input string) bool {
	re1 := regexp.MustCompile(`^Records: \d+  Duplicates: \d+  Warnings: \d+$`)
	re2 := regexp.MustCompile(`^Rows matched: \d+  Changed: \d+  Warnings: \d+$`)
	return re1.MatchString(input) || re2.MatchString(input)
}

func processResult(s string) string {
	lines := strings.Split(s, "\n")
	var cnt int
	var first bool
	var result []string
	var out strings.Builder
	out.WriteString("\n\n")
	for i := 0; i < len(lines); i++ {
		value := lines[i]
		switch {
		case value == resultLine:
			if cnt%2 == 0 {
				first = false
			}
			first = true
			cnt++
		case cnt%2 != 0:
			if strings.HasPrefix(value, "SELECT SLEEP") {
				i += 9
				first = false
				cnt++
				continue
			}
			if first {
				value = fmt.Sprintf("mysql> %s", value)
				first = false
			} else {
				value = fmt.Sprintf("    -> %s", value)
			}
			if lines[i+1] == resultLine {
				result = append(result, value+";")
			} else {
				result = append(result, value)
			}
		case value == bye:
			continue
		case value != "":
			if isQueryOutput(value) {
				result = append(result, fmt.Sprintf("%s\n", value))
			} else if isDMLResultOutput(value) {
				result[len(result)-1] = strings.TrimSuffix(result[len(result)-1], "\n")
				result = append(result, fmt.Sprintf("%s\n", value))
			} else {
				result = append(result, value)
			}
		}
	}
	for _, r := range result {
		out.WriteString(r)
		out.WriteString("\n")
	}
	return out.String()
}

func processError(s string) string {
	ss := strings.Split(s, "\n")
	var r strings.Builder
	for _, s := range ss {
		if s != "" && s != SqlWarn {
			r.WriteString(s)
		}
	}
	return r.String()
}

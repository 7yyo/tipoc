package mysql

import (
	"bytes"
	"fmt"
	"github.com/go-mysql-org/go-mysql/client"
	"github.com/go-mysql-org/go-mysql/mysql"
	"net"
	"os/exec"
	"pictorial/log"
	"strings"
)

type MySQL struct {
	User     string
	Password string
	Host     string
	Port     string
}

var M MySQL

func (m *MySQL) ExecuteSQL(sql string) (*mysql.Result, error) {
	addr := net.JoinHostPort(m.Host, m.Port)
	conn, err := client.Connect(addr, m.User, m.Password, "")
	if err != nil {
		return nil, err
	}
	defer conn.Close()
	log.Logger.Debug(sql)
	return conn.Execute(sql)
}

const mysqlCmd = "mysql"
const mysqlCli = "mysql>"
const mysqlCliWarp = "    ->"
const bye = "Bye"
const selectSleep = "SELECT SLEEP"
const verbosity = "-vvv"
const comments = "--comments"
const force = "--force"
const skipPassword = "--skip-password"
const SqlWarn = "mysql: [Warning] Using a password on the command line interface can be insecure.\n"

const resultLine = "--------------"

func (m *MySQL) args(sql string, user, password string) []string {
	args := []string{
		fmt.Sprintf("-u%s", user),
		fmt.Sprintf("-p%s", password),
		fmt.Sprintf("-h%s", m.Host),
		fmt.Sprintf("-P%s", m.Port),
		fmt.Sprintf("-e %s", sql),
		verbosity,
		comments,
		force,
	}
	if password == "" {
		args = append(args, skipPassword)
	}
	return args
}

func (m *MySQL) ExecuteForceWithOutput(sql, user, password string) ([]string, error) {
	var stdout, stderr bytes.Buffer
	cmdArgs := m.args(sql, user, password)
	cmd := exec.Command(mysqlCmd, cmdArgs...)
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	errOutput := rewriteErrOutput(stderr.String())
	output := rewriteResultOutput(stdout.String())
	if len(errOutput) != 0 {
		output = outputAppendErrOutput(output, errOutput)
	}
	if stderr.String() != "" && stderr.String() != SqlWarn {
		if err != nil {
			return output, fmt.Errorf("%s: %s", errOutput, err.Error())
		} else {
			return output, fmt.Errorf("%s", errOutput)
		}
	}
	return output, nil
}

const sleepHeight = 9
const bingo = 2

func rewriteResultOutput(s string) []string {
	lines := strings.Split(s, "\n")
	var cnt int
	var first bool
	var result []string
	result = append(result, "")
	for i := 0; i < len(lines); i++ {
		value := strings.TrimLeft(lines[i], " ")
		switch {
		case value == resultLine:
			if cnt%bingo == 0 {
				first = false
			}
			first = true
			cnt++
		case cnt%bingo != 0:
			if strings.HasPrefix(value, selectSleep) {
				i += sleepHeight
				first = false
				cnt++
				continue
			}
			if first {
				value = fmt.Sprintf("%s %s", mysqlCli, value)
				first = false
			} else {
				value = fmt.Sprintf("%s %s", mysqlCliWarp, value)
			}
			if lines[i+1] == resultLine {
				result = append(result, fmt.Sprintf("%s;", value))
			} else {
				result = append(result, value)
			}
		case value == bye:
			continue
		case value != "":
			if isQueryOutput(value) || isEmptySetOutput(value) {
				result = append(result, value)
				result = append(result, "")
			} else if isDMLOutput(value) {
				result[len(result)-1] = value
				result = append(result, "")
			} else {
				result = append(result, value)
			}
		}
	}
	return result
}

func outputAppendErrOutput(output []string, errOutput []string) []string {
	i := 0
	var meet bool
	var newOutPut []string
	for _, o := range output {
		if IsMysqlHeader(o) {
			if meet {
				newOutPut = append(newOutPut, errOutput[i])
				newOutPut = append(newOutPut, "")
				i++
			}
			meet = true
		}
		if isDMLOutput(o) || isEmptySetOutput(o) || isQueryOutput(o) {
			meet = false
		}
		newOutPut = append(newOutPut, o)
	}
	if i == len(errOutput)-1 {
		newOutPut = append(newOutPut, errOutput[i])
		newOutPut = append(newOutPut, "")
	}
	return newOutPut
}

func rewriteErrOutput(s string) []string {
	ss := strings.Split(s, "\n")
	var r []string
	for _, s := range ss {
		if s != "" && s != strings.TrimSuffix(SqlWarn, "\n") {
			r = append(r, s)
		}
	}
	return r
}

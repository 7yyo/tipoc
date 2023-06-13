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
	"strconv"
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
	log.Logger.Debug(s)
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
	if _, err := m.ExecuteSQL("SET GLOBAL validate_password.enable = OFF;"); err != nil {
		return err
	}
	return nil
}

const resultFile = "%s_%d.result"

func (m *MySQL) ExecuteAndWrite(s, rd, name string, idx int32) error {
	var stdout, stderr bytes.Buffer
	cmdArgs := m.args(s)
	cmd := exec.Command("mysql", cmdArgs...)
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	fName := filepath.Join(rd, fmt.Sprintf(resultFile, name, idx))
	f, _ := os.Create(fName)
	resultOutput := updateResultOutput(stdout.String())
	errOutput := updateErrOutput(stderr.String())
	_, _ = io.WriteString(f, resultOutput)
	_, _ = io.WriteString(f, errOutput)
	if stderr.String() != "" && stderr.String() != SqlWarn {
		if err != nil {
			return fmt.Errorf("%s: %s", errOutput, err.Error())
		} else {
			return fmt.Errorf("%s", errOutput)
		}
	}
	return nil
}

func (m *MySQL) ExecuteUserAndWrite(s, fName, user, password string) error {
	var stdout, stderr bytes.Buffer
	cmdArgs := []string{
		fmt.Sprintf("-u%s", user),
		fmt.Sprintf("-p%s", password),
		fmt.Sprintf("-h%s", m.Host),
		fmt.Sprintf("-P%s", m.Port),
		fmt.Sprintf("-e %s", s),
		"-vvv",
		"--comments",
	}
	if password == "" {
		cmdArgs = append(cmdArgs, "--skip-password")
	}
	cmd := exec.Command("mysql", cmdArgs...)
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	log.Logger.Debug(cmd)
	err := cmd.Run()
	f, _ := os.Create(fName)
	resultOutput := updateResultOutput(stdout.String())
	errOutput := updateErrOutput(stderr.String())
	_, _ = io.WriteString(f, resultOutput)
	_, _ = io.WriteString(f, errOutput)
	if stderr.String() != "" && stderr.String() != SqlWarn {
		if err != nil {
			return fmt.Errorf("%s: %s", errOutput, err.Error())
		} else {
			return fmt.Errorf("%s", errOutput)
		}
	}
	return nil
}

const resultLine = "--------------"
const bye = "Bye"
const SqlWarn = "mysql: [Warning] Using a password on the command line interface can be insecure.\n"

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

func isEmptySet(input string) bool {
	return strings.Contains(input, "Empty set")
}

func isDMLResultOutput(input string) bool {
	re1 := regexp.MustCompile(`^Records: \d+  Duplicates: \d+  Warnings: \d+$`)
	re2 := regexp.MustCompile(`^Rows matched: \d+  Changed: \d+  Warnings: \d+$`)
	return re1.MatchString(input) || re2.MatchString(input)
}

const sleep = "SELECT SLEEP"
const sleepHeight = 9
const bingo = 2

func updateResultOutput(s string) string {
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
			if cnt%bingo == 0 {
				first = false
			}
			first = true
			cnt++
		case cnt%bingo != 0:
			if strings.HasPrefix(value, sleep) {
				i += sleepHeight
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
			if isQueryOutput(value) || isEmptySet(value) {
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

func updateErrOutput(s string) string {
	ss := strings.Split(s, "\n")
	var r strings.Builder
	for _, s := range ss {
		if s != "" && s != strings.TrimSuffix(SqlWarn, "\n") {
			r.WriteString(s)
		}
	}
	return r.String()
}

func (m *MySQL) GetPdAddr() (string, error) {
	rs, err := m.ExecuteSQL("SELECT * FROM information_schema.cluster_info WHERE type = 'pd'")
	if err != nil {
		return "", err
	}
	defer rs.Close()
	if rs == nil {
		return "", fmt.Errorf("please confirm that the [pd] exists in the cluster")
	}
	pd := string(rs.Values[0][1].AsString())
	log.Logger.Debug("pd = %s", pd)
	return pd, nil
}

func (m *MySQL) GetTiDBHostStatusPort() (string, string, error) {
	rs, err := m.ExecuteSQL("SELECT * FROM information_schema.tidb_servers_info")
	if err != nil {
		return "", "", err
	}
	defer rs.Close()
	if rs == nil {
		return "", "", fmt.Errorf("please confirm that the [tidb] exists in the cluster")
	}
	host := string(rs.Values[0][1].AsString())
	statusPort := strconv.FormatInt(rs.Values[0][3].AsInt64(), 10)
	return host, statusPort, nil
}

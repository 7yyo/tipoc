package ssh

import (
	"bufio"
	"bytes"
	"fmt"
	"golang.org/x/crypto/ssh"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"pictorial/log"
	"runtime"
	"strings"
	"time"
)

type SSH struct {
	User     string
	Password string
	SshPort  string
	LogC     chan string
	Cluster
	Key
}

type Cluster struct {
	Name   string
	Plugin string
}

type Key struct {
	Public  string
	Private string
}

var S SSH

func (s *SSH) AddSSHKey() error {
	tiupRoot, err := s.WhichTiup()
	if err != nil {
		return err
	}
	s.Key.Private = privateKeyPath(tiupRoot, s.Cluster.Name)
	s.Key.Public = publicKeyPath(tiupRoot, S.Cluster.Name)
	return nil
}

func privateKeyPath(root, clusterName string) string {
	return path.Join(root, "storage", "cluster", "clusters", clusterName, "ssh", "id_rsa")
}

func (s *SSH) ParsePrivateKey() (ssh.Signer, error) {
	file, err := ioutil.ReadFile(s.Key.Private)
	if err != nil {
		return nil, err
	}
	return ssh.ParsePrivateKey(file)
}

func publicKeyPath(root, clusterName string) string {
	return path.Join(root, "storage", "cluster", "clusters", clusterName, "ssh", "id_rsa.pub")
}

func (s *SSH) NewSshClient(host string) (*ssh.Client, error) {
	rs, err := s.ParsePrivateKey()
	sshConfig := newSshConfig(s.User)
	if err != nil {
		log.Logger.Warnf("parse private key: %s failed", s.Key.Private)
		if s.Password != "" {
			log.Logger.Warnf("retry password: %s", s.Password)
			sshConfig.Auth = []ssh.AuthMethod{ssh.Password(s.Password)}
		} else {
			return nil, fmt.Errorf("ssh failed, private key: %s, password: %s, please check", s.Private, s.Password)
		}
	} else {
		sshConfig.Auth = []ssh.AuthMethod{ssh.PublicKeys(rs)}
	}
	addr := fmt.Sprintf("%s:%s", host, s.SshPort)
	return ssh.Dial("tcp", addr, sshConfig)
}

func newSshConfig(user string) *ssh.ClientConfig {
	return &ssh.ClientConfig{
		User:            user,
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}
}

const failedMsg = "'%s' error: %w: %s, %s"
const warnMsg = "'%s' warn: %w: %s, %s"

func (s *SSH) RunSSH(h, c string) ([]byte, error) {
	sc, err := s.NewSshClient(h)
	if err != nil {
		return nil, err
	}
	defer sc.Close()
	ss, err := sc.NewSession()
	if err != nil {
		return nil, err
	}
	defer ss.Close()
	var stdout, stderr bytes.Buffer
	ss.Stdout = &stdout
	ss.Stderr = &stderr
	err = ss.Run(c)
	s.LogC <- formatCommand(c)
	s.LogC <- formatStdout(stdout)
	s.LogC <- formatStderr(stderr)
	if err != nil {
		if _, ok := err.(*ssh.ExitError); ok {
			return nil, fmt.Errorf(failedMsg, c, err, stdout.String(), stderr.String())
		} else {
			return nil, fmt.Errorf(warnMsg, c, err, stdout.String(), stderr.String())
		}
	}
	return stdout.Bytes(), nil
}

func (s *SSH) RunLocal(c string) ([]byte, error) {
	cmd := exec.Command("bash", "-c", c)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	s.LogC <- formatCommand(c)
	s.LogC <- formatStdout(stdout)
	s.LogC <- formatStderr(stderr)
	if err != nil {
		if _, ok := err.(*ssh.ExitError); ok {
			return nil, fmt.Errorf(failedMsg, c, err, stdout.String(), stderr.String())
		} else {
			return nil, fmt.Errorf(warnMsg, c, err, stdout.String(), stderr.String())
		}
	}
	return stdout.Bytes(), nil
}

func (s *SSH) RunLocalWithArg(c string, arg []string) ([]byte, error) {
	cmd := exec.Command(c, arg...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	if err != nil {
		if _, ok := err.(*ssh.ExitError); ok {
			return nil, fmt.Errorf(failedMsg, c, err, stdout.String(), stderr.String())
		} else {
			return nil, fmt.Errorf(warnMsg, c, err, stdout.String(), stderr.String())
		}
	}
	return stdout.Bytes(), nil
}

func (s *SSH) RunLocalWithWrite(c string, arg []string, fName string) ([]byte, error) {
	f, err := os.Create(fName)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	cmd := exec.Command(c, arg...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = io.MultiWriter(f)
	cmd.Stderr = &stderr
	err = cmd.Run()
	if err != nil {
		if _, ok := err.(*ssh.ExitError); ok {
			return nil, fmt.Errorf(failedMsg, c, err, stdout.String(), stderr.String())
		} else {
			return nil, fmt.Errorf(warnMsg, c, err, stdout.String(), stderr.String())
		}
	}
	return stdout.Bytes(), nil
}

func (s *SSH) DirWalk(host, target string) ([]byte, error) {
	c := fmt.Sprintf("ls -A %s", target)
	return s.RunSSH(host, c)
}

func (s *SSH) Transfer(t1, t2 string) ([]byte, error) {
	c := fmt.Sprintf("scp -o StrictHostKeyChecking=no -i %s %s %s", s.Key.Private, t1, t2)
	return s.RunLocal(c)
}

func (s *SSH) UnZip(host, obj, path string) ([]byte, error) {
	c := fmt.Sprintf("unzip %s -d %s", obj, path)
	return s.RunSSH(host, c)
}

func (s *SSH) Restart(role string) ([]byte, error) {
	c := fmt.Sprintf("tiup cluster restart %s --yes", s.Cluster.Name)
	if role != "" {
		c += fmt.Sprintf(" -R %s", role)
	}
	return s.RunLocal(c)
}

func (s *SSH) ScaleIn(addr string) ([]byte, error) {
	c := fmt.Sprintf("tiup cluster scale-in %s -N %s --yes", s.Cluster.Name, addr)
	return s.RunLocal(c)
}

func (s *SSH) Remove(host, o string) ([]byte, error) {
	c := fmt.Sprintf("rm -r -f %s", o)
	return s.RunSSH(host, c)
}

func (s *SSH) Kill9(host, p string) ([]byte, error) {
	c := fmt.Sprintf("sudo kill -9 %s", p)
	return s.RunSSH(host, c)
}

func (s *SSH) Mv(host, source, to string) ([]byte, error) {
	c := fmt.Sprintf("mv %s %s", source, to)
	return s.RunSSH(host, c)
}

const noToAlways = "sudo sed -i 's/no/always/g' %s"
const alwaysToNo = "sudo sed -i 's/always/no/g' %s"
const reloadSystemd = "sudo systemctl daemon-reload"
const (
	Always = iota
	No
)

func (s *SSH) Systemd(host string, w int, f string) ([]byte, error) {
	var cmd string
	switch w {
	case Always:
		cmd = fmt.Sprintf(noToAlways, f)
	case No:
		cmd = fmt.Sprintf(alwaysToNo, f)
	}
	if _, err := s.RunSSH(host, cmd); err != nil {
		return nil, err
	}
	return s.RunSSH(host, reloadSystemd)
}

func (s *SSH) GetProcessIDByPs(host, psCmd string) ([]string, error) {
	c := fmt.Sprintf("ps aux | grep '%s' | grep -v grep | awk '{print $2}'\n", psCmd)
	o, err := s.RunSSH(host, c)
	if err != nil {
		return nil, err
	}
	id := make([]string, 0)
	sc := bufio.NewScanner(strings.NewReader(string(o)))
	for sc.Scan() {
		if sc.Text() != "" {
			id = append(id, sc.Text())
		}
	}
	return id, nil
}

func (s *SSH) GetProcessIDByPort(host, port string) (string, error) {
	c := fmt.Sprintf("sudo fuser -n tcp %s/tcp | tail -n 1", port)
	p, _ := s.RunSSH(host, c)
	return strings.TrimSpace(string(p)), nil
}

func (s *SSH) CheckClusterName() error {
	if !isLinux() {
		return nil
	}
	tiup, err := s.WhichTiup()
	if err != nil {
		return err
	}
	arg := []string{filepath.Join(tiup, "storage", "cluster", "clusters")}
	o, err := s.RunLocalWithArg("ls", arg)
	if err != nil {
		return err
	}
	if !strings.Contains(string(o), s.Cluster.Name) {
		return fmt.Errorf("cluster: %s is not exists", s.Cluster.Name)
	}
	return nil
}

func (s *SSH) WhichTiup() (string, error) {
	up, err := exec.LookPath("tiup")
	if err != nil {
		return "", err
	}
	base := strings.Trim(path.Dir(up), "/bin")
	return filepath.Join("/", base), nil
}

func isLinux() bool {
	return runtime.GOOS == "linux"
}

const ShellLog = "shell.log"

func (s *SSH) CommandListener() {
	if err := os.Remove(ShellLog); err != nil {
		if !os.IsNotExist(err) {
			panic(err)
		}
	}
	f, err := os.Create(ShellLog)
	if err != nil {
		panic(err)
	}
	for {
		select {
		case s := <-s.LogC:
			_, _ = f.WriteString(s)
		}
	}
}

func CleanShellLog() error {
	return os.Remove(ShellLog)
}

func formatCommand(c string) string {
	return fmt.Sprintf("[%s] [localhost] %s\n", dateFormat(), c)
}

func formatStdout(stdout bytes.Buffer) string {
	out := strings.ReplaceAll(stdout.String(), "\n", "")
	return fmt.Sprintf("[%s] [stdout] %s\n", dateFormat(), out)
}

func formatStderr(stderr bytes.Buffer) string {
	err := strings.ReplaceAll(stderr.String(), "\n", "")
	return fmt.Sprintf("[%s] [stderr] %s\n", dateFormat(), err)
}

func dateFormat() string {
	now := time.Now()
	year, month, day := now.Date()
	hour, min, sec := now.Clock()
	return fmt.Sprintf("%d-%02d-%02d_%02d:%02d:%02d", year, int(month), day, hour, min, sec)
}

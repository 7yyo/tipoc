package ssh

import (
	"bufio"
	"bytes"
	"fmt"
	"golang.org/x/crypto/ssh"
	"io/ioutil"
	"os/exec"
	"path"
	"path/filepath"
	"pictorial/log"
	"runtime"
	"strings"
)

type SSH struct {
	User     string
	Password string
	SshPort  string
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

func (s *SSH) GetSSHKey() error {
	tiupRoot, err := s.whichTiup()
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
	if err != nil {
		return nil, err
	}
	config := &ssh.ClientConfig{
		User: s.User,
		Auth: []ssh.AuthMethod{
			ssh.PublicKeys(rs),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}
	addr := fmt.Sprintf("%s:%s", host, s.SshPort)
	return ssh.Dial("tcp", addr, config)
}

const failedMsg = "'%s' failed: %w: %s, %s"

func (s *SSH) RunSSH(h, c string) ([]byte, error) {
	log.Logger.Debug(c)
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
	if err := ss.Run(c); err != nil {
		return stdout.Bytes(), fmt.Errorf(failedMsg, c, err, stdout.String(), stderr.String())
	}
	if stderr.Len() != 0 {
		return stdout.Bytes(), fmt.Errorf(failedMsg, c, err, stdout.String(), stderr.String())
	}
	return stdout.Bytes(), nil
}

func RunLocal(c string) ([]byte, error) {
	log.Logger.Debug(c)
	cmd := exec.Command("bash", "-c", c)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	if err != nil {
		return stdout.Bytes(), fmt.Errorf(failedMsg, c, err, stdout.String(), stderr.String())
	}
	if stderr.Len() != 0 && err != nil {
		return stdout.Bytes(), fmt.Errorf(failedMsg, c, err, stdout.String(), stderr.String())
	}
	return stdout.Bytes(), nil
}

func RunLocalWithArg(c string, arg []string) ([]byte, error) {
	cmd := exec.Command(c, arg...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf(failedMsg, c, err, stdout.String(), stderr.String())
	}
	return stdout.Bytes(), nil
}

func (s *SSH) DirWalk(host, target string) ([]byte, error) {
	c := fmt.Sprintf("ls -A %s", target)
	return s.RunSSH(host, c)
}

func (s *SSH) Transfer(t1, t2 string) ([]byte, error) {
	c := fmt.Sprintf("scp -o StrictHostKeyChecking=no -i %s %s %s", s.Key.Private, t1, t2)
	return RunLocal(c)
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
	return RunLocal(c)
}

func (s *SSH) Remove(host, o string) ([]byte, error) {
	c := fmt.Sprintf("rm -r -f %s", o)
	return s.RunSSH(host, c)
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

func (s *SSH) GetProcessIDByPort(host, port string) ([]byte, error) {
	c := fmt.Sprintf("sudo fuser -n tcp %s/tcp | tail -n 1", port)
	return s.RunSSH(host, c)
}

func (s *SSH) CheckClusterName() error {
	if !isLinux() {
		return nil
	}
	tiup, err := s.whichTiup()
	if err != nil {
		return err
	}
	arg := []string{filepath.Join(tiup, "storage", "cluster", "clusters")}
	o, err := RunLocalWithArg("ls", arg)
	if err != nil {
		return err
	}
	if !strings.Contains(string(o), s.Cluster.Name) {
		return fmt.Errorf("cluster: %s is not exists", s.Cluster.Name)
	}
	return nil
}

func (s *SSH) whichTiup() (string, error) {
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

package ssh

import (
	"bufio"
	"bytes"
	"fmt"
	"golang.org/x/crypto/ssh"
	"os/exec"
	"path"
	"pictorial/log"
	"strings"
)

type SSH struct {
	User        string
	Password    string
	SshPort     string
	PrivateKey  string
	PublicKey   string
	ClusterName string
	Carry
}

type Carry struct {
	Plugin string
}

var S SSH

func (s *SSH) ApplySSHKey() error {
	tiup, err := exec.LookPath("tiup")
	if err != nil {
		return err
	}
	envRoot := path.Dir(tiup)
	s.PrivateKey = privateKeyPath(path.Dir(envRoot), S.ClusterName)
	s.PublicKey = publicKeyPath(path.Dir(envRoot), S.ClusterName)
	log.Logger.Debug(s.PublicKey)
	log.Logger.Debug(s.PrivateKey)
	return nil
}

func privateKeyPath(root, clusterName string) string {
	return path.Join(root, "storage", "cluster", "clusters", clusterName, "ssh", "id_rsa")
}

func publicKeyPath(root, clusterName string) string {
	return path.Join(root, "storage", "cluster", "clusters", clusterName, "ssh", "id_rsa.pub")
}

func (s *SSH) NewSshClient(host string) (*ssh.Client, error) {
	config := &ssh.ClientConfig{
		User: s.User,
		Auth: []ssh.AuthMethod{
			ssh.Password(s.Password),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}
	addr := fmt.Sprintf("%s:%s", host, s.SshPort)
	return ssh.Dial("tcp", addr, config)
}

const runFailedMsg = "'%s' failed: %w: %s, %s"

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
		return stdout.Bytes(), fmt.Errorf(runFailedMsg, c, err, stdout.String(), stderr.String())
	}
	if stderr.Len() != 0 {
		return stdout.Bytes(), fmt.Errorf(runFailedMsg, c, err, stdout.String(), stderr.String())
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
		return stdout.Bytes(), fmt.Errorf(runFailedMsg, c, err, stdout.String(), stderr.String())
	}
	if stderr.Len() != 0 && err != nil {
		return stdout.Bytes(), fmt.Errorf(runFailedMsg, c, err, stdout.String(), stderr.String())
	}
	return stderr.Bytes(), nil
}

func (s *SSH) GetProcessID(host, psCmd string) ([]string, error) {
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

func (s *SSH) DirWalk(host, target string) ([]byte, error) {
	c := fmt.Sprintf("ls -A %s", target)
	return s.RunSSH(host, c)
}

func (s *SSH) Transfer(t1, t2 string) ([]byte, error) {
	c := fmt.Sprintf("scp -i %s %s %s", s.PrivateKey, t1, t2)
	return RunLocal(c)
}

func (s *SSH) UnZip(host, obj, path string) ([]byte, error) {
	c := fmt.Sprintf("unzip %s -d %s", obj, path)
	return s.RunSSH(host, c)
}

func (s *SSH) Restart(role string) ([]byte, error) {
	c := fmt.Sprintf("tiup cluster restart %s --yes", s.ClusterName)
	if role != "" {
		c += fmt.Sprintf(" -R %s", role)
	}
	return RunLocal(c)
}

func (s *SSH) Remove(host, o string) ([]byte, error) {
	c := fmt.Sprintf("rm -r -f %s", o)
	return s.RunSSH(host, c)
}

func (s *SSH) GetProcessIDByPort(host, port string) ([]byte, error) {
	c := fmt.Sprintf("sudo fuser -n tcp %s/tcp | tail -n 1", port)
	return s.RunSSH(host, c)
}

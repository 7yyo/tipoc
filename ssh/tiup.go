package ssh

import (
	"fmt"
	"golang.org/x/crypto/ssh"
	"io/ioutil"
	"os/exec"
	"path"
	"path/filepath"
	"strings"
)

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

func publicKeyPath(root, clusterName string) string {
	return path.Join(root, "storage", "cluster", "clusters", clusterName, "ssh", "id_rsa.pub")
}

func (s *SSH) ParsePrivateKey() (ssh.Signer, error) {
	file, err := ioutil.ReadFile(s.Key.Private)
	if err != nil {
		return nil, err
	}
	return ssh.ParsePrivateKey(file)
}

func (s *SSH) WhichTiup() (string, error) {
	up, err := exec.LookPath("tiup")
	if err != nil {
		return "", err
	}
	base := strings.Trim(path.Dir(up), "/bin")
	return filepath.Join("/", base), nil
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

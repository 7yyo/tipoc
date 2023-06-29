package ssh

import (
	"bufio"
	"fmt"
	"strings"
)

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

func (s *SSH) YumInstall(host, repo string) ([]byte, error) {
	c := fmt.Sprintf("sudo yum install -y %s", repo)
	return s.RunSSH(host, c)
}

func (s *SSH) GetProcessIDByPort(host, port string) (string, error) {
	c := fmt.Sprintf("sudo fuser -n tcp %s/tcp | tail -n 1", port)
	p, _ := s.RunSSH(host, c)
	return strings.TrimSpace(string(p)), nil
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

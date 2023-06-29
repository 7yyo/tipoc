package ssh

import (
	"bytes"
	"context"
	"fmt"
	"golang.org/x/crypto/ssh"
	"io"
	"os"
	"os/exec"
	"pictorial/log"
	"runtime"
	"strings"
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
	s.LogC <- formatCommand(c, h)
	err = ss.Run(c)
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
	s.LogC <- formatCommand(c, "127.0.0.1")
	err := cmd.Run()
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

func (s *SSH) RunSSHWithContext(ctx context.Context, host, c string) ([]byte, error) {

	sc, err := s.NewSshClient(host)
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

	s.LogC <- formatCommand(c, host)
	s.LogC <- formatStdout(stdout)
	s.LogC <- formatStderr(stderr)

	errC := make(chan error)
	go func() {
		errC <- ss.Run(c)
	}()

	select {
	case <-ctx.Done():
		if err := ss.Signal(ssh.SIGKILL); err != nil {
			return stdout.Bytes(), err
		}
	case err := <-errC:
		if err != nil {
			if _, ok := err.(*ssh.ExitError); ok {
				return nil, fmt.Errorf(failedMsg, c, err, stdout.String(), stderr.String())
			} else {
				return nil, fmt.Errorf(warnMsg, c, err, stdout.String(), stderr.String())
			}
		}
	}
	return stdout.Bytes(), nil
}

func (s *SSH) RunLocalWithContext(ctx context.Context, c string, arg []string, fName string) ([]byte, error) {

	cmd := exec.Command(c, arg...)
	var stdout, stderr bytes.Buffer
	cmd.Stderr = &stderr

	if fName != "" {
		f, err := os.Create(fName)
		if err != nil {
			return nil, err
		}
		defer f.Close()
		cmd.Stdout = io.MultiWriter(f)
	} else {
		cmd.Stdout = &stdout
	}

	s.LogC <- formatCommand(c, "127.0.0.1")
	s.LogC <- formatStdout(stdout)
	s.LogC <- formatStderr(stderr)

	errC := make(chan error)
	go func() {
		errC <- cmd.Run()
	}()

	select {
	case <-ctx.Done():
		if err := cmd.Process.Kill(); err != nil {
			return nil, err
		}
	case err := <-errC:
		if err != nil {
			if _, ok := err.(*ssh.ExitError); ok {
				return nil, fmt.Errorf(failedMsg, c, err, stdout.String(), stderr.String())
			} else {
				return nil, fmt.Errorf(warnMsg, c, err, stdout.String(), stderr.String())
			}
		}
	}
	return stdout.Bytes(), nil
}

func isLinux() bool {
	return runtime.GOOS == "linux"
}

const ShellLog = "shell.log"

func (s *SSH) ShellListener() {
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

func formatCommand(c string, host string) string {
	return fmt.Sprintf("[%s] [%s] %s\n", log.DateFormat(), host, c)
}

func formatStdout(stdout bytes.Buffer) string {
	out := strings.ReplaceAll(stdout.String(), "\n", "")
	return fmt.Sprintf("[%s] [stdout] %s\n", log.DateFormat(), out)
}

func formatStderr(stderr bytes.Buffer) string {
	err := strings.ReplaceAll(stderr.String(), "\n", "")
	return fmt.Sprintf("[%s] [stderr] %s\n", log.DateFormat(), err)
}

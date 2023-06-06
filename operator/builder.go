package operator

import (
	"fmt"
	"pictorial/ssh"
)

const (
	kill           = "KILL"
	crash          = "CRASH"
	dataCorrupted  = "DATA_CORRUPTED"
	recoverSystemd = "RECOVER_SYSTEMD"
	scaleIn        = "SCALE_IN"
	reboot         = "REBOOT"
)

type Builder struct {
	Host        string
	Port        string
	Tp          string
	ComponentTp string
	DeployPath  string
}

type Operator interface {
	Execute() error
}

func (b *Builder) Build() (Operator, error) {
	switch b.Tp {
	case kill:
		return b.BuildKill()
	case crash:
		return b.BuildCrash()
	case dataCorrupted:
		return b.BuildDataCorrupted()
	case recoverSystemd:
		return b.BuildRecoverSystemd()
	case scaleIn:
		return b.BuildScaleIn()
	case reboot:
		return b.BuildReboot()
	default:
		return nil, fmt.Errorf("unknown operator: %s", b.Tp)
	}
}

func (b *Builder) BuildKill() (Operator, error) {
	return &killOperator{
		host:        b.Host,
		port:        b.Port,
		componentTp: b.ComponentTp,
	}, nil
}

func (b *Builder) BuildCrash() (Operator, error) {
	return &crashOperator{
		host:        b.Host,
		port:        b.Port,
		componentTp: b.ComponentTp,
	}, nil
}

func (b *Builder) BuildRecoverSystemd() (Operator, error) {
	return &recoverSystemdOperator{
		host:        b.Host,
		port:        b.Port,
		componentTp: b.ComponentTp,
	}, nil
}

func (b *Builder) BuildScaleIn() (Operator, error) {
	return &scaleInOperator{
		host:        b.Host,
		port:        b.Port,
		clusterName: ssh.S.ClusterName,
	}, nil
}

func (b *Builder) BuildDataCorrupted() (Operator, error) {
	return &dataCorruptedOperator{
		host:        b.Host,
		port:        b.Port,
		componentTp: b.ComponentTp,
		deployPath:  b.DeployPath,
	}, nil
}

func (b *Builder) BuildReboot() (Operator, error) {
	return &rebootOperator{
		host: b.Host,
	}, nil
}

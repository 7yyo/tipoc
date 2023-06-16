package operator

import (
	"fmt"
	"pictorial/ssh"
)

const (
	kill           = "kill"
	crash          = "crash"
	dataCorrupted  = "data_corrupted"
	recoverSystemd = "recover_systemd"
	scaleIn        = "scale_in"
	reboot         = "reboot"
)

type Builder struct {
	Host       string
	Port       string
	OType      string
	CType      string
	DeployPath string
}

type Operator interface {
	Execute() error
}

func (b *Builder) Build() (Operator, error) {
	switch b.OType {
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
		return nil, fmt.Errorf("unknown operator: %s", b.OType)
	}
}

func (b *Builder) BuildKill() (Operator, error) {
	return &killOperator{
		host:        b.Host,
		port:        b.Port,
		componentTp: b.CType,
	}, nil
}

func (b *Builder) BuildCrash() (Operator, error) {
	return &crashOperator{
		host:       b.Host,
		port:       b.Port,
		cType:      b.CType,
		deployPath: b.DeployPath,
	}, nil
}

func (b *Builder) BuildRecoverSystemd() (Operator, error) {
	return &recoverSystemdOperator{
		host:       b.Host,
		port:       b.Port,
		cType:      b.CType,
		deployPath: b.DeployPath,
	}, nil
}

func (b *Builder) BuildScaleIn() (Operator, error) {
	return &scaleInOperator{
		host:        b.Host,
		port:        b.Port,
		clusterName: ssh.S.Cluster.Name,
	}, nil
}

func (b *Builder) BuildDataCorrupted() (Operator, error) {
	return &dataCorruptedOperator{
		host:       b.Host,
		port:       b.Port,
		cType:      b.CType,
		deployPath: b.DeployPath,
	}, nil
}

func (b *Builder) BuildReboot() (Operator, error) {
	return &rebootOperator{
		host: b.Host,
	}, nil
}

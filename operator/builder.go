package operator

import (
	"fmt"
	"pictorial/comp"
	"pictorial/ssh"
	"pictorial/widget"
)

type Builder struct {
	Host       string
	Port       string
	OType      widget.OType
	CType      comp.CType
	DeployPath string
	StopC      chan bool
}

type Operator interface {
	Execute() error
}

func (b *Builder) Build() (Operator, error) {
	switch b.OType {
	case widget.ScaleIn:
		return b.BuildScaleIn()
	case widget.RecoverSystemd:
		return b.BuildRecoverSystemd()
	case widget.Kill:
		return b.BuildKill()
	case widget.DataCorrupted:
		return b.BuildDataCorrupted()
	case widget.Crash:
		return b.BuildCrash()
	case widget.Reboot:
		return b.BuildReboot()
	case widget.DiskFull:
		return b.BuildDiskFull()
	default:
		return nil, fmt.Errorf("unknown operator: %s", b.OType)
	}
}

func (b *Builder) BuildKill() (Operator, error) {
	return &killOperator{
		host:  b.Host,
		port:  b.Port,
		cType: b.CType,
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
		cType:       b.CType,
		deployPath:  b.DeployPath,
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

func (b *Builder) BuildDiskFull() (Operator, error) {
	return &diskFullOperator{
		host:       b.Host,
		port:       b.Port,
		cType:      b.CType,
		deployPath: b.DeployPath,
		stopC:      b.StopC,
	}, nil
}

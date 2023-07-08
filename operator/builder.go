package operator

import (
	"context"
	"fmt"
	"pictorial/comp"
	"pictorial/ssh"
)

type Builder struct {
	Host string
	Port string
	OType
	comp.CType
	DeployPath string
	Ctx        context.Context
}

type OType int

const (
	Script OType = iota
	SafetyScript
	OtherScript
	ScaleIn
	Kill
	DataCorrupted
	Crash
	RecoverSystemd
	Disaster
	Reboot
	DiskFull
	LoadDataTPCC
	LoadDataImportInto
	LoadData
	LoadDataSelectIntoOutFile
	DataDistribution
	OnlineDDLAddIndex
	InstallSysBench
)

func GetOTypeValue(o OType) string {
	switch o {
	case Script:
		return "script"
	case SafetyScript:
		return "safetyScript"
	case OtherScript:
		return "otherScript"
	case ScaleIn:
		return "scale_in"
	case Kill:
		return "kill"
	case DataCorrupted:
		return "data_corrupted"
	case Crash:
		return "crash"
	case RecoverSystemd:
		return "recover_systemd"
	case Disaster:
		return "disaster"
	case Reboot:
		return "reboot"
	case DiskFull:
		return "disk_full"
	case LoadDataTPCC:
		return "load_data_tpc-c"
	case LoadDataImportInto:
		return "import_into"
	case LoadData:
		return "load_data"
	case LoadDataSelectIntoOutFile:
		return "select_into_outfile"
	case DataDistribution:
		return "data_distribution"
	case OnlineDDLAddIndex:
		return "online_ddl_add_index"
	case InstallSysBench:
		return "install_sys_bench"
	default:
		return ""
	}
}

type Operator interface {
	Execute() error
}

func (b *Builder) Build() (Operator, error) {
	switch b.OType {
	case ScaleIn:
		return b.BuildScaleIn()
	case RecoverSystemd:
		return b.BuildRecoverSystemd()
	case Kill:
		return b.BuildKill()
	case DataCorrupted:
		return b.BuildDataCorrupted()
	case Crash:
		return b.BuildCrash()
	case Reboot:
		return b.BuildReboot()
	case DiskFull:
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
		ctx:        b.Ctx,
	}, nil
}

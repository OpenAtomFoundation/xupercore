package contract

import "github.com/OpenAtomFoundation/xupercore/global/protos"

const (
	maxResourceLimit = 0xFFFFFFFF
)

// Limits describes the usage or limit of resources
type Limits struct {
	Cpu    int64
	Memory int64
	Disk   int64
	XFee   int64
}

// // TotalGas converts resource to gas
func (l *Limits) TotalGas(gasPrice *protos.GasPrice) int64 {
	cpuGas := roundup(l.Cpu, gasPrice.GetCpuRate())
	memGas := roundup(l.Memory, gasPrice.GetMemRate())
	diskGas := roundup(l.Disk, gasPrice.GetDiskRate())
	feeGas := roundup(l.XFee, gasPrice.GetXfeeRate())
	return cpuGas + memGas + diskGas + feeGas
}

// Add accumulates resource limits, returns self.
func (l *Limits) Add(l1 Limits) *Limits {
	l.Cpu += l1.Cpu
	l.Memory += l1.Memory
	l.Disk += l1.Disk
	l.XFee += l1.XFee
	return l
}

// Sub sub limits from l
func (l *Limits) Sub(l1 Limits) *Limits {
	l.Cpu -= l1.Cpu
	l.Memory -= l1.Memory
	l.Disk -= l1.Disk
	l.XFee -= l1.XFee
	return l
}

// Exceed judge whether resource exceeds l1
func (l Limits) Exceed(l1 Limits) bool {
	return l.Cpu > l1.Cpu ||
		l.Memory > l1.Memory ||
		l.Disk > l1.Disk ||
		l.XFee > l1.XFee
}

// MaxLimits describes the maximum limit of resources
var MaxLimits = Limits{
	Cpu:    maxResourceLimit,
	Memory: maxResourceLimit,
	Disk:   maxResourceLimit,
	XFee:   maxResourceLimit,
}

// // FromPbLimits converts []*pb.ResourceLimit to Limits
func FromPbLimits(rlimits []*protos.ResourceLimit) Limits {
	limits := Limits{}
	for _, l := range rlimits {
		switch l.GetType() {
		case protos.ResourceType_CPU:
			limits.Cpu = l.GetLimit()
		case protos.ResourceType_MEMORY:
			limits.Memory = l.GetLimit()
		case protos.ResourceType_DISK:
			limits.Disk = l.GetLimit()
		case protos.ResourceType_XFEE:
			limits.XFee = l.GetLimit()
		}
	}
	return limits
}

// // FromPbLimits converts Limits to []*pb.ResourceLimit
func ToPbLimits(limits Limits) []*protos.ResourceLimit {
	return []*protos.ResourceLimit{
		{Type: protos.ResourceType_CPU, Limit: limits.Cpu},
		{Type: protos.ResourceType_MEMORY, Limit: limits.Memory},
		{Type: protos.ResourceType_DISK, Limit: limits.Disk},
		{Type: protos.ResourceType_XFEE, Limit: limits.XFee},
	}
}

func roundup(n, scale int64) int64 {
	if scale == 0 {
		return 0
	}
	return (n + scale - 1) / scale
}

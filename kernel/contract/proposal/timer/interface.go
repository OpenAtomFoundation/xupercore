package timer

import (
	pb "github.com/xuperchain/xupercore/protos"
)

type TimerManager interface {
	GetTimerTasks(blockHeight int64) (*pb.Acl, error)
}

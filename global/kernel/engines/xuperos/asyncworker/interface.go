package asyncworker

import "github.com/OpenAtomFoundation/xupercore/global/kernel/engines/xuperos/common"

type AsyncWorker interface {
	RegisterHandler(contract string, event string, handler func(ctx common.TaskContext) error)
}

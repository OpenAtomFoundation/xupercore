package asyncworker

import "github.com/xuperchain/xupercore/kernel/engines/xuperos/common"

type AsyncWorker interface {
	RegisterHandler(contract string, event string, handler common.TaskHandler)
}

package bridge

import (
	"github.com/OpenAtomFoundation/xupercore/global/kernel/contract"
	"github.com/OpenAtomFoundation/xupercore/global/protos"
)

func eventsResourceUsed(events []*protos.ContractEvent) contract.Limits {
	var size int64
	for _, event := range events {
		size += int64(len(event.Contract) + len(event.Name) + len(event.Body))
	}
	return contract.Limits{
		Disk: size,
	}
}

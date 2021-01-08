package reader

import (
	xctx "github.com/xuperchain/xupercore/kernel/common/xcontext"
	"github.com/xuperchain/xupercore/kernel/engines/xuperos/common"
	"github.com/xuperchain/xupercore/lib/logs"
	"github.com/xuperchain/xupercore/protos"
)

type ContractReader interface {
	// 查询该链合约统计数据
	QueryContractStatData() (*protos.ContractStatData, error)
	// 查询账户下合约状态
	GetAccountContracts(account string) ([]*protos.ContractStatus, error)
	// 查询地址下合约状态
	GetAddressContracts(address string, needContent bool) (map[string][]*protos.ContractStatus, error)
	// 查询地址下账户
	GetAccountByAK(address string) ([]string, error)
	// 查询合约账户ACL
	QueryAccountACL(account string) (*protos.Acl, bool, error)
	// 查询合约方法ACL
	QueryContractMethodACL(contract, method string) (*protos.Acl, bool, error)
}

type contractReader struct {
	chainCtx *common.ChainCtx
	baseCtx  xctx.XContext
	log      logs.Logger
}

func NewContractReader(chainCtx *common.ChainCtx, baseCtx xctx.XContext) ContractReader {
	if chainCtx == nil || baseCtx == nil {
		return nil
	}

	reader := &contractReader{
		chainCtx: chainCtx,
		baseCtx:  baseCtx,
		log:      baseCtx.GetLog(),
	}

	return reader
}

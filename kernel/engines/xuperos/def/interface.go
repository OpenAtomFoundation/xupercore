package def

import (
	"github.com/xuperchain/xupercore/bcs/ledger/xledger/ledger"
	"github.com/xuperchain/xupercore/bcs/ledger/xledger/state"
	"github.com/xuperchain/xupercore/kernel/consensus"
	"github.com/xuperchain/xupercore/kernel/contract"
	"github.com/xuperchain/xupercore/kernel/network"
	aclBase "github.com/xuperchain/xupercore/kernel/permission/acl/base"
	cryptoBase "github.com/xuperchain/xupercore/lib/crypto/client/base"

	"github.com/xuperchain/xupercore/kernel/engines"
)

type Chain interface {
	Context() *ChainCtx
	Start()
	Stop()
	ProcTx(request *pb.TxStatus) *pb.CommonReply
	ProcBlock(request *pb.Block) error
	PreExec(request *pb.InvokeRPCRequest) (*pb.InvokeResponse, error)
	SetRelyAgent(ChainRelyAgent) error
}

// 定义xuperos引擎对外暴露接口
// 依赖接口而不是依赖具体实现
type Engine interface {
	Context() *EngineCtx
	engines.BCEngine
	Get(string) Chain
	Set(string, Chain)
	GetChains() []string
	CreateChain(string, []byte) error
	RegisterChain(string) error
	UnloadChain(string) error
	SetRelyAgent(EngineRelyAgent) error
}

// 定义引擎对各组件依赖接口约束
type EngineRelyAgent interface {
	CreateNetwork() (network.Network, error)
}

// 定义链对各组件依赖接口约束
type ChainRelyAgent interface {
	CreateLedger() (*ledger.Ledger, error)
	CreateState() (*state.XuperState, error)
	CreateContract() (contract.Manager, error)
	CreateConsensus() (consensus.ConsensusInterface, error)
	CreateCrypto() (cryptoBase.CryptoClient, error)
	CreateAcl() (aclBase.AclManager, error)
}

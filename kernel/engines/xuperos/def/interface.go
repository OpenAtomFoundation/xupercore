// 面向接口编程
package def

import (
	"context"
	"crypto/ecdsa"
	"github.com/xuperchain/xuperchain/core/pb"
	"github.com/xuperchain/xuperchain/core/vat"
	"github.com/xuperchain/xupercore/bcs/ledger/xledger/ledger"
	"github.com/xuperchain/xupercore/kernel/engines/xuperos/reader"
	nctx "github.com/xuperchain/xupercore/kernel/network/context"
	"math/big"

	//"github.com/xuperchain/xupercore/bcs/ledger/xledger/pb"
	"github.com/xuperchain/xupercore/kernel/common/xcontext"
	"github.com/xuperchain/xupercore/kernel/consensus/base"
	cctx "github.com/xuperchain/xupercore/kernel/consensus/context"
	"github.com/xuperchain/xupercore/kernel/contract"
	"github.com/xuperchain/xupercore/kernel/engines"
	"github.com/xuperchain/xupercore/kernel/network/p2p"
	netPB "github.com/xuperchain/xupercore/kernel/network/pb"
	"github.com/xuperchain/xupercore/lib/storage/kvdb"
)

type Chain interface {
	Start()
	Stop()

	Context() *ChainCtx
	Status() int
	Reader() reader.Reader
	SetRelyAgent(ChainRelyAgent) error

	ProcTx(request *pb.TxStatus) error
	ProcBlock(request *pb.Block) error
	PreExec(request *pb.InvokeRPCRequest) (*pb.InvokeResponse, error)
}

// 定义该引擎对各组件依赖接口约束
type ChainRelyAgent interface {
	CreateLedger() (XLedger, error)
	CreateState() (XState, error)
	CreateContract() (XContract, error)
	CreateConsensus() (XConsensus, error)
	CreateCrypto() (XCrypto, error)
	CreateAcl() (XAcl, error)
}

// 定义xuperos引擎对外暴露接口
// 依赖接口而不是依赖具体实现
type Engine interface {
	engines.BCEngine
	Context() *EngineCtx
	SetRelyAgent(EngineRelyAgent) error
	Get(string) Chain
	Set(string, Chain)
	GetChains() []string
	//CreateChain(string, []byte) (Chain, error)
	CreateChain(string, []byte) error
	RegisterChain(string) error
	UnloadChain(string) error
}

// 定义该引擎对各组件依赖接口约束
type EngineRelyAgent interface {
	CreateNetwork() (XNetwork, error)
}

// 定义引擎对网络组件依赖接口约束
type XNetwork interface {
	Start()
	Stop()

	NewSubscriber(netPB.XuperMessage_MessageType, interface{}, ...p2p.SubscriberOption) p2p.Subscriber
	Register(p2p.Subscriber) error
	UnRegister(p2p.Subscriber) error

	SendMessage(context.Context, *netPB.XuperMessage, ...p2p.OptionFunc) error
	SendMessageWithResponse(context.Context, *netPB.XuperMessage, ...p2p.OptionFunc) ([]*netPB.XuperMessage, error)

	Context() nctx.DomainCtx
	P2PState() *p2p.State
}

// 定义引擎对账本组件依赖接口约束
type XLedger interface {
	// 打开账本
	//Open(param *kvdb.KVParameter, crypto XCrypto) (XLedger, error)

	// NewLedger(storePath string, xlog log.Logger, otherPaths []string, kvEngineType string, cryptoType string) (*Ledger, error)
	Close()
	Truncate(lastBlockId []byte) error
	GetMeta() *pb.LedgerMeta
	GetGenesisBlock() *ledger.GenesisBlock

	// ConfirmBlock submit a block to ledger
	ConfirmBlock(block *pb.InternalBlock, isRoot bool) ledger.ConfirmStatus
	// ExistBlock check if a block exists in the ledger
	ExistBlock(blockID []byte) bool
	// FormatFakeBlock format fake block for contract pre-execution without signing
	FormatFakeBlock(
		txs []*pb.Transaction,
		addr *Address,
		height int64, preHash []byte, timestamp int64,
		consensus []byte) (*pb.InternalBlock, error)
	// FormatMinerBlock format block for miner
	FormatMinerBlock(
		txs []*pb.Transaction, failedTxs map[string]string,
		addr *Address,
		height int64, preHash []byte, timestamp int64,
		consensus []byte) (*pb.InternalBlock, error)

	// IsValidTx valid transactions of coinbase in block
	IsValidTx(tx *pb.Transaction, block *pb.InternalBlock) bool

	GetMaxBlockSize() int64
	GetNoFee() bool

	//// GetPendingBlock get block from pending table
	//GetPendingBlock(blockID []byte) (*pb.Block, error)
	//// SavePendingBlock put block into pending table
	//SavePendingBlock(block *pb.Block) error

	// for branch
	//GetLDB()
	//SetMeta(meta *pb.LedgerMeta)
	//GetBranchInfo()
	//HandleFork()
	//RemoveBlocks()

	// for reader
	// tx
	QueryTransaction(txId []byte) (*pb.Transaction, error)
	// block
	QueryBlock(blockId []byte) (*pb.InternalBlock, error)
	QueryBlockHeader(blockId []byte) (*pb.InternalBlock, error)
	QueryBlockByHeight(height int64) (*pb.InternalBlock, error)
	// branch
	GetBranchInfo(blockId []byte, blockHeight int64) ([]string, error)
}

type XState interface {
	New(bcName string, param *kvdb.KVParameter, addr *Address, ledger XLedger, crypto XCrypto)

	GetOfflineTx() chan []*pb.Transaction
	GetLatestBlockId() []byte
	MaxTxSizePerBlock() int
	GetVATList(height int64, maxCount int, timestamp int64) ([]*pb.Transaction, error)
	GetUnconfirmedTx(bool) ([]*pb.Transaction, error)
	PreExec(req *pb.InvokeRPCRequest) (*pb.InvokeResponse, error)
	// HasTx 查询一笔交易是否在unconfirm表
	HasTx(txId []byte) (*pb.Transaction, bool)

	// MakeUtxoVM 构建一个UtxoVM对象，定制版
	MakeUtxoVM()
	Close()
	// DoTx 执行一个交易, 影响utxo表和unconfirm-transaction表
	DoTx(tx *pb.Transaction) error
	// GenerateAwardTx 生成系统奖励的交易, 比如矿工挖矿所得
	GenerateAwardTx(address []byte, awardAmount string, desc []byte) (*pb.Transaction, error)
	// GenerateTx 根据一个原始订单, 得到UTXO格式的交易, 相当于预执行, 会在内存中锁定一段时间UTXO, 但是不修改kv存储
	//GenerateTx(txReq *pb.TxData) (*pb.Transaction, error)

	// IsAsync return current async state
	IsAsync() bool
	// IsAsyncBlock return current async state
	IsAsyncBlock() bool
	// NewBatch return batch instance
	NewBatch() kvdb.Batch
	// NotifyFinishBlockGen notify to finish generating block
	NotifyFinishBlockGen()
	// PlayAndRepost 执行一个新收到的block，要求block的pre_hash必须是当前vm的latest_block
	PlayAndRepost(blockID []byte, needRepost bool, isRootTx bool) error
	// PlayForMiner 进行合约预执行
	PlayForMiner(blockID []byte, batch kvdb.Batch) error
	// PreExec the Xuper3 contract model uses previous execution to generate RWSets
	//PreExec(req *pb.InvokeRPCRequest, hd *global.XContext) (*pb.InvokeResponse, error)

	RegisterVAT(name string, vat vat.VATInterface, whiteList map[string]bool)
	//RegisterVM(name string, vm contract.ContractInterface, priv int) bool
	//RegisterVM3(module string, vm contract.VirtualMachine) error
	// RollBackUnconfirmedTx 回滚本地未确认交易
	RollBackUnconfirmedTx() (map[string]bool, []*pb.Transaction, error)
	// SetBlockGenEvent set if try to generate block in async mode
	SetBlockGenEvent()
	SetMaxConfirmedDelay(seconds uint32)
	SetModifyBlockAddr(addr string)
	StartAsyncBlockMode()
	StartAsyncWriter()
	// TxOfRunningContractGenerate 预执行当前的交易里面的合约
	TxOfRunningContractGenerate(txlist []*pb.Transaction, pendingBlock *pb.InternalBlock, outerBatch kvdb.Batch, ctxInit bool) ([]*pb.Transaction, kvdb.Batch, error)
	// VerifyTx check the tx signature and permission
	VerifyTx(tx *pb.Transaction) (bool, error)
	// Walk 从当前的latestBlockid 游走到 blockid, 会触发utxo状态的回滚
	Walk(blockid []byte, ledgerPrune bool) error

	// for reader

	// tx
	QueryTransaction(txId []byte) (*pb.Transaction, error)
	// meta
	GetMeta() *pb.UtxoMeta
	// account
	QueryAccountContainAK(address string) ([]string, error)
	QueryAccountACLWithConfirmed(accountName string) (*pb.Acl, bool, error)
	QueryContractMethodACLWithConfirmed(contractName string, methodName string) (*pb.Acl, bool, error)
	QueryContractStatData() (*pb.ContractStatData, error)
	QueryUtxoRecord(accountName string, displayCount int64) (*pb.UtxoRecordDetail, error)
	QueryTxFromForbiddenWithConfirmed(txId []byte) (bool, bool, error)
	GetAccountContracts(account string) ([]string, error)
	GetContractStatus(contractName string) (*pb.ContractStatus, error)
	// balance
	GetBalance(address string) (*big.Int, error)
	GetFrozenBalance(address string) (*big.Int, error)
	GetBalanceDetail(address string) ([]*pb.TokenFrozenDetail, error)

	//GetAccountContracts()
	//GetBalance()
	//GetBalanceDetail()
	//GetContractStatus()
	//GetFromTable()
	//GetFrozenBalance()

	//GetMeta()
	//GetTotal()
	//GetUnconfirmedTx()
	//GetVATList()
	//GetXModel()
	//
	//QueryAccountACLWithConfirmed()
	//QueryAccountContainAK()
	//QueryContractMethodACLWithConfirmed()
	//QueryContractStatData()
	//QueryTxFromForbiddenWithConfirmed()
	//QueryUtxoRecord()
	//
	//ScanWithPrefix()
}

type XContract interface {
	NewContext(*contract.ContextConfig) (contract.Context, error)
}

// 定义引擎对共识组件依赖接口约束
type XConsensus interface {
	//NewPluggableConsensus()

	// CompeteMaster 返回是否为矿工以及是否需要进行SyncBlock
	CompeteMaster(height int64) (bool, bool, error)
	// CheckMinerMatch 当前block是否合法
	CheckMinerMatch(ctx xcontext.BaseCtx, block cctx.BlockInterface) (bool, error)
	// ProcessBeforeMiner 开始挖矿前进行相应的处理, 返回是否需要truncate, 返回写consensusStorage, 返回err
	ProcessBeforeMiner(timestamp int64) (bool, []byte, error)
	// CalculateBlock 矿工挖矿时共识需要做的工作, 如PoW时共识需要完成存在性证明
	CalculateBlock(block cctx.BlockInterface) error
	// ProcessConfirmBlock 用于确认块后进行相应的处理
	ProcessConfirmBlock(block cctx.BlockInterface) error
	// GetStatus 获取区块链共识信息
	Status() base.ConsensusStatus

	//// Type return the consensus type of a specific height
	//Type(height int64) string
	//// Version return the consensus version of a specific height
	//Version(height int64) int64
	//// ProcessBeforeMiner preprocessing before mining
	//ProcessBeforeMiner(height int64, timestamp int64) (map[string]interface{}, bool)
	//// ProcessConfirmBlock process after block has been confirmed
	//ProcessConfirmBlock(block *pb.InternalBlock) error
	//// CheckMinerMatch check whether the block is valid
	////CheckMinerMatch(header *pb.Header, in *pb.InternalBlock) (bool, error)
	//// CompeteMaster confirm whether the node is a miner or not
	//CompeteMaster(height int64) (bool, bool)
	//// GetCoreMiners get the information of core miners
	//GetCoreMiners() []*consensus.MinerInfo
	//// GetVATWhiteList the specific implementation of interface VAT
	//GetVATWhiteList() map[string]bool

	// for reader
	//GetStatus()
}

// 定引擎义对权限组件依赖接口约束
type XAcl interface {
}

// 定义引擎对加密组件依赖接口约束
type XCrypto interface {
	// 从导出的私钥文件读取私钥
	GetEcdsaPrivateKeyFromJsonStr(keyStr string) (*ecdsa.PrivateKey, error)
	// 从导出的公钥文件读取公钥
	GetEcdsaPublicKeyFromJsonStr(key string) (*ecdsa.PublicKey, error)
	// 使用ECC公钥来验证签名
	VerifyECDSA(k *ecdsa.PublicKey, signature, msg []byte) (valid bool, err error)
	// 验证钱包地址是否和指定的公钥match。如果成功，返回true和对应的版本号；如果失败，返回false和默认的版本号0
	VerifyAddressUsingPublicKey(address string, pub *ecdsa.PublicKey) (bool, uint8)
}

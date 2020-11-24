package pow

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"sync"

	"github.com/xuperchain/xupercore/kernel/common/xcontext"
	"github.com/xuperchain/xupercore/kernel/consensus"
	"github.com/xuperchain/xupercore/kernel/consensus/base"
	"github.com/xuperchain/xupercore/kernel/consensus/context"
)

var (
	PoWBlockItemErr   = errors.New("invalid block structure, pls check item nonce & targetbits")
	OODMineErr        = errors.New("mining height is out of date")
	TryTooMuchMineErr = errors.New("mining max tries threshold")
	InternalErr       = errors.New("Consensus module found internal error")
)

const MAX_TRIES = ^uint64(0) // mining时的最大尝试次数

func init() {
	consensus.Register("pow", NewPoWConsensus)
}

// 目前未定义pb结构
// PoWStorage pow占用block中consensusStorage json串的格式
type PoWStorage struct {
	/* TargetBits 以一个uint32类型解析，16进制格式为0xFFFFFFFF
	 * 真正difficulty按照Bitcoin标准转化，将TargetBits转换为一个uint256 bits的大数
	 */
	TargetBits uint32 `json:"targetBits,omitempty"`
}

// PoWConsensus pow具体结构
type PoWConsensus struct {
	// Pluggable Consensus传递的上下文, PoW并不使用P2p interface
	ctx    context.ConsensusCtx
	status *PoWStatus
	config *PoWConfig

	targetBits        uint32
	sigc              chan bool
	defaultDifficulty *big.Int
	maxDifficulty     *big.Int
}

/* PoWConfig pow需要解析的创始块解析格式
 *  Bitcoin推荐 {
 *    AdjustHeightGap: 2016,
 *	  MaxTarget: 0x1d00FFFF,
 *  }
 */
type PoWConfig struct {
	DefaultTarget        uint32 `json:"defaultTarget"`
	AdjustHeightGap      int32  `json:"adjustHeightGap"`
	ExpectedPeriodMilSec int32  `json:"expectedPeriod"`
	MaxTarget            uint32 `json:"maxTarget"`
}

// PoWStatus 实现了ConsensusStatus接口
type PoWStatus struct {
	beginHeight int64
	mutex       sync.RWMutex
	newHeight   int64
	index       int
	miner       MinerInfo
}

// MinerInfo 针对GetCurrentValidatorsInfo json串解析
type MinerInfo struct {
	Address string `json:"address"`
}

// GetVersion 返回pow所在共识version
func (s *PoWStatus) GetVersion() int64 {
	return 0
}

// GetConsensusBeginInfo 返回该实例初始高度
func (s *PoWStatus) GetConsensusBeginInfo() int64 {
	return s.beginHeight
}

// GetStepConsensusIndex 获取共识item所在consensus slice中的index
func (s *PoWStatus) GetStepConsensusIndex() int {
	return s.index
}

// GetConsensusName 获取共识类型
func (s *PoWStatus) GetConsensusName() string {
	return "pow"
}

// GetCurrentTerm 获取当前状态机term
func (s *PoWStatus) GetCurrentTerm() int64 {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	return s.newHeight
}

// GetCurrentValidatorsInfo 获取当前矿工信息
func (s *PoWStatus) GetCurrentValidatorsInfo() []byte {
	info, err := json.Marshal(s.miner)
	if err != nil {
		return nil
	}
	return info
}

// ParseConsensusStorage PoW parse专有存储的逻辑，即targetBits
func (pow *PoWConsensus) ParseConsensusStorage(block context.BlockInterface) (interface{}, error) {
	store := PoWStorage{}
	err := json.Unmarshal(block.GetConsensusStorage(), &store)
	if err != nil {
		pow.ctx.BCtx.XLog.Error("PoW::ParseConsensusStorage invalid consensus storage", "err", err)
		return nil, err
	}
	return store, nil
}

// CalculateBlock 挖矿过程
func (pow *PoWConsensus) CalculateBlock(block context.BlockInterface) error {
	return pow.mining(block)
}

// NewPoWConsensus 初始化实例
func NewPoWConsensus(cCtx context.ConsensusCtx, cCfg context.ConsensusConfig) base.ConsensusImplInterface {
	// 解析config中需要的字段
	if cCtx.BCtx.XLog == nil {
		return nil
	}
	// TODO:cCtx.BcName需要注册表吗？
	if cCtx.CryptoClient == nil {
		cCtx.BCtx.XLog.Error("PoW::NewPoWConsensus::CryptoClient in context is nil")
		return nil
	}
	if cCtx.Ledger == nil {
		cCtx.BCtx.XLog.Error("PoW::NewPoWConsensus::Ledger in context is nil")
		return nil
	}
	if cCfg.ConsensusName != "pow" {
		cCtx.BCtx.XLog.Error("PoW::NewPoWConsensus::consensus name in config is wrong", "name", cCfg.ConsensusName)
		return nil
	}
	config := &PoWConfig{}
	err := json.Unmarshal([]byte(cCfg.Config), config)
	if err != nil {
		cCtx.BCtx.XLog.Error("PoW::NewPoWConsensus::pow struct unmarshal error", "error", err)
		return nil
	}
	// 通过MaxTarget和DefaultTarget解析maxDifficulty和DefaultDifficulty
	md, fNegative, fOverflow := SetCompact(config.MaxTarget)
	if fNegative || fOverflow {
		cCtx.BCtx.XLog.Error("PoW::NewPoWConsensus::pow set MaxTarget error", "fNegative", fNegative, "fOverflow", fOverflow)
		return nil
	}
	dd, fNegative, fOverflow := SetCompact(config.DefaultTarget)
	if fNegative || fOverflow {
		cCtx.BCtx.XLog.Error("PoW::NewPoWConsensus::pow set Default error", "fNegative", fNegative, "fOverflow", fOverflow)
		return nil
	}
	// newHeight取上一共识的最高值，因为此时BeginHeight也许并为生产出来
	pow := &PoWConsensus{
		ctx:    cCtx,
		config: config,
		status: &PoWStatus{
			beginHeight: cCfg.BeginHeight,
			newHeight:   cCfg.BeginHeight - 1,
			index:       cCfg.Index,
			miner: MinerInfo{
				Address: cCtx.P2p.GetLocalAddress(),
			},
		},
		sigc:              make(chan bool, 1),
		maxDifficulty:     md,
		defaultDifficulty: dd,
	}
	return pow
}

// CompeteMaster PoW单一节点都为矿工，故返回为true
func (pow *PoWConsensus) CompeteMaster(height int64) (bool, bool, error) {
	return true, true, nil
}

// CheckMinerMatch 验证区块，包括merkel根和hash
func (pow *PoWConsensus) CheckMinerMatch(ctx xcontext.BaseCtx, block context.BlockInterface) (bool, error) {
	// TODO: 报错统一打出矿工地址
	// 检查区块是否有targetBits字段
	in, err := pow.ParseConsensusStorage(block)
	if err != nil {
		ctx.XLog.Warn("PoW::CheckMinerMatch::ParseConsensusStorage err", "logid", ctx.XLog.GetLogId(), "err", err, "blockId", block.GetBlockid())
		return false, nil
	}
	s, ok := in.(PoWStorage)
	if !ok {
		ctx.XLog.Warn("PoW::CheckMinerMatch::transfer PoWStorage err", "logid", ctx.XLog.GetLogId(), "blockId", block.GetBlockid())
		return false, nil
	}
	// 检查区块的区块头是否和和区块中的targetBits字段匹配
	if !pow.IsProofed(block.GetBlockid(), s.TargetBits) {
		ctx.XLog.Warn("PoW::CheckMinerMatch::the actual difficulty of block received doesn't match its' blockid", "logid", ctx.XLog.GetLogId(), "blockid", fmt.Sprintf("%x", block.GetBlockid()))
		return false, nil
	}
	// 检查区块的区块头是否hash正确
	if !bytes.Equal(block.MakeBlockId(), block.GetBlockid()) {
		ctx.XLog.Warn("PoW::CheckMinerMatch::equal blockid error", "logid", ctx.XLog.GetLogId())
		return false, nil
	}
	// 验证merkel根是否计算正确
	if err := pow.ctx.Ledger.VerifyMerkle(block); err != nil {
		ctx.XLog.Warn("PoW::CheckMinerMatch::VerifyMerkle error", "logid", ctx.XLog.GetLogId(), "error", err)
		return false, nil
	}
	// 验证difficulty是否正确
	targetBits, err := pow.refreshDifficulty(block.GetPreHash(), block.GetHeight())
	if err != nil {
		ctx.XLog.Warn("PoW::CheckMinerMatch::refreshDifficulty err", "logid", ctx.XLog.GetLogId(), "error", err)
		return false, nil
	}
	if targetBits != s.TargetBits {
		ctx.XLog.Warn("PoW::CheckMinerMatch::unexpected target bits", "logid", ctx.XLog.GetLogId(), "expect", targetBits, "got", s.TargetBits)
		return false, nil
	}
	// 验证时间戳是否正确
	preBlock, err := pow.ctx.Ledger.QueryBlock(block.GetPreHash())
	if err != nil {
		ctx.XLog.Warn("PoW::CheckMinerMatch::get preblock error", "logid", ctx.XLog.GetLogId())
		return false, nil
	}
	if block.GetTimestamp() < preBlock.GetTimestamp() {
		ctx.XLog.Warn("PoW::CheckMinerMatch::unexpected block timestamp", "logid", ctx.XLog.GetLogId(), "pre", preBlock.GetTimestamp(), "next", block.GetTimestamp())
		return false, nil
	}
	// 验证前导0
	if !pow.IsProofed(block.GetBlockid(), targetBits) {
		ctx.XLog.Warn("PoW::CheckMinerMatch::blockid IsProofed error", "logid", ctx.XLog.GetLogId())
		return false, nil
	}
	//验证签名
	//1 验证一下签名和公钥是不是能对上
	k, err := pow.ctx.CryptoClient.GetEcdsaPublicKeyFromJSON(block.GetPubkey())
	if err != nil {
		ctx.XLog.Warn("PoW::CheckMinerMatch::get ecdsa from block error", "logid", ctx.XLog.GetLogId(), "error", err)
		return false, nil
	}
	//Todo 跟address比较
	chkResult, _ := pow.ctx.CryptoClient.VerifyAddressUsingPublicKey(string(block.GetProposer()), k)
	if chkResult == false {
		ctx.XLog.Warn("PoW::CheckMinerMatch::address is not match publickey", "logid", ctx.XLog.GetLogId())
		return false, nil
	}
	//2 验证一下签名是否正确
	valid, err := pow.ctx.CryptoClient.VerifyECDSA(k, block.GetSign(), block.GetBlockid())
	if err != nil {
		ctx.XLog.Warn("PoW::CheckMinerMatch::verifyECDSA error", "logid", ctx.XLog.GetLogId(), "error", err)
	}
	return valid, nil
}

// ProcessBeforeMiner 更新下一次pow挖矿时的targetBits
func (pow *PoWConsensus) ProcessBeforeMiner(timestamp int64) (bool, []byte, error) {
	pow.status.mutex.RLock()
	tipHeight := pow.status.newHeight
	pow.status.mutex.RUnlock()
	preBlock, err := pow.ctx.Ledger.QueryBlockByHeight(tipHeight)
	if err != nil {
		pow.ctx.BCtx.XLog.Error("PoW::ProcessBeforeMiner::cannnot find preBlock", "logid", pow.ctx.BCtx.XLog.GetLogId())
		return false, nil, InternalErr
	}
	bits, err := pow.refreshDifficulty(preBlock.GetBlockid(), tipHeight+1)
	if err != nil {
		pow.Stop()
	}
	pow.targetBits = bits
	return false, nil, nil
}

// ProcessConfirmBlock 此处更新最新的block高度
func (pow *PoWConsensus) ProcessConfirmBlock(block context.BlockInterface) error {
	pow.status.mutex.Lock()
	defer pow.status.mutex.Unlock()
	if block.GetHeight() > pow.status.newHeight {
		pow.status.newHeight = block.GetHeight()
	}
	return nil
}

// GetConsensusStatus 获取pow实例状态
func (pow *PoWConsensus) GetConsensusStatus() (base.ConsensusStatus, error) {
	// TODO:精简接口，这块不需要返回一个err
	return pow.status, nil
}

// Stop 立即停止当前挖矿
func (pow *PoWConsensus) Stop() error {
	// 发送停止信号
	pow.sigc <- true
	return nil
}

// Start 重启实例
func (pow *PoWConsensus) Start() error {
	return nil
}

/* refreshDifficulty 计算difficulty in bitcoin
 * reference of bitcoin's pow: https://github.com/bitcoin/bitcoin/blob/master/src/pow.cpp#L49
 */
func (pow *PoWConsensus) refreshDifficulty(tipHash []byte, nextHeight int64) (uint32, error) {
	// 未到调整高度0 + Gap，直接返回default
	if nextHeight <= int64(pow.config.AdjustHeightGap) {
		return pow.config.DefaultTarget, nil
	}
	// 检查block结构是否合法，获取上一区块difficulty
	preBlock, err := pow.ctx.Ledger.QueryBlockHeader(tipHash)
	if err != nil {
		return pow.config.DefaultTarget, nil
	}
	in, err := pow.ParseConsensusStorage(preBlock)
	if err != nil {
		pow.ctx.BCtx.XLog.Error("PoW::refreshDifficulty::ParseConsensusStorage err", "err", err, "blockId", tipHash)
		return 0, err
	}
	s, ok := in.(PoWStorage)
	if !ok {
		pow.ctx.BCtx.XLog.Error("PoW::refreshDifficulty::transfer PoWStorage err")
		return 0, PoWBlockItemErr
	}
	prevTargetBits := s.TargetBits
	// 未到调整时机直接返回上一difficulty
	if nextHeight%int64(pow.config.AdjustHeightGap) != 0 {
		return prevTargetBits, nil
	}

	farBlock := preBlock
	// preBlock已经回溯过一次，因此回溯总量-1，获取
	for i := int32(0); i < pow.config.AdjustHeightGap-1; i++ {
		prevBlock, err := pow.ctx.Ledger.QueryBlockHeader(farBlock.GetPreHash())
		if err != nil {
			return pow.config.DefaultTarget, nil
		}
		farBlock = prevBlock
	}
	expectedTimeSpan := pow.config.ExpectedPeriodMilSec * pow.config.AdjustHeightGap
	// ATTENTION: 此处并没有针对任意的Timestamp类型，目前只能是timestamp为nano类型
	actualTimeSpan := int32((preBlock.GetTimestamp() - farBlock.GetTimestamp()) / 1e6)
	pow.ctx.BCtx.XLog.Debug("PoW::refreshDifficulty::timespan diff", "expectedTimeSpan", expectedTimeSpan, "actualTimeSpan", actualTimeSpan)
	//at most adjust two bits, left or right direction
	if actualTimeSpan < expectedTimeSpan/4 {
		actualTimeSpan = expectedTimeSpan / 4
	}
	if actualTimeSpan > expectedTimeSpan*4 {
		actualTimeSpan = expectedTimeSpan * 4
	}
	difficulty, _, _ := SetCompact(prevTargetBits) // prevTargetBits一定在之前检查过
	difficulty.Mul(difficulty, big.NewInt(int64(actualTimeSpan)))
	difficulty.Div(difficulty, big.NewInt(int64(expectedTimeSpan)))
	if difficulty.Cmp(pow.maxDifficulty) == 1 {
		pow.ctx.BCtx.XLog.Debug("PoW::refreshDifficulty::retarget", "newTargetBits", pow.config.MaxTarget)
		return pow.config.MaxTarget, nil
	}
	newTargetBits, ok := GetCompact(difficulty)
	if !ok {
		pow.ctx.BCtx.XLog.Error("PoW::refreshDifficulty::difficulty GetCompact err")
		return prevTargetBits, nil
	}
	pow.ctx.BCtx.XLog.Info("PoW::refreshDifficulty::adjust targetBits", "height", nextHeight, "targetBits", newTargetBits, "prevTargetBits", prevTargetBits)
	return newTargetBits, nil
}

//IsProofed check workload proof
func (pow *PoWConsensus) IsProofed(blockID []byte, targetBits uint32) bool {
	d, fNegative, fOverflow := SetCompact(targetBits)
	if fNegative || fOverflow || d.Cmp(pow.maxDifficulty) == 1 { // d > maxDifficulty
		return false
	}
	hash := new(big.Int)
	hash.SetBytes(blockID)
	if hash.Cmp(d) == 1 { // hash > d
		return false
	}
	return true
}

/* SetCompact 将一个uint32的target转换为一个difficulty
 */
func SetCompact(nCompact uint32) (*big.Int, bool, bool) {
	nSize := nCompact >> 24
	nWord := new(big.Int)
	u := new(big.Int)
	nCompactInt := big.NewInt(int64(nCompact))
	// 0x00800000是一个符号位，故nWord仅为后23位
	lowBits := big.NewInt(0x007fffff)
	nWord.And(nCompactInt, lowBits)
	if nSize <= 3 {
		nWord.Rsh(nWord, uint(8*(3-nSize)))
		u = nWord
	} else {
		u = nWord
		u.Lsh(u, uint(8*(nSize-3)))
	}
	pfNegative := nWord.Cmp(big.NewInt(0)) != 0 && (nCompact&0x00800000) != 0
	pfOverflow := nWord.Cmp(big.NewInt(0)) != 0 && ((nSize > 34) ||
		(nWord.Cmp(big.NewInt(0xff)) == 1 && nSize > 33) ||
		(nWord.Cmp(big.NewInt(0xffff)) == 1 && nSize > 32))
	return u, pfNegative, pfOverflow
}

/* GetCompact 将一个256bits的大数转换为一个target
 */
func GetCompact(number *big.Int) (uint32, bool) {
	nSize := (number.BitLen() + 7) / 8
	nCompact := uint32(0)
	low64Int := new(big.Int)
	low64Int.SetUint64(0xFFFFFFFFFFFFFFFF)
	low64Int.And(low64Int, number)
	low64 := low64Int.Uint64()
	if nSize <= 3 {
		nCompact = uint32(low64 << uint64(8*(3-nSize)))
	} else {
		bn := new(big.Int)
		bn.Rsh(number, uint(8*(nSize-3)))
		low64Int.SetUint64(0xFFFFFFFFFFFFFFFF)
		low64Int.And(low64Int, bn)
		low64 := low64Int.Uint64()
		nCompact = uint32(low64)
	}
	// The 0x00800000 bit denotes the sign.
	// Thus, if it is already set, divide the mantissa by 256 and increase the exponent.
	if nCompact&0x00800000 > 0 {
		nCompact >>= 8
		nSize++
	}
	if (nCompact&0xFF800000) != 0 || nSize > 256 {
		return 0, false
	}
	nCompact |= uint32(nSize) << 24
	return nCompact, true
}

// mining 为带副作用的函数，将直接对block进行操作，更改其原始值
func (pow *PoWConsensus) mining(block context.BlockInterface) error {
	gussNonce := uint64(0)
	tries := MAX_TRIES
	for {
		select {
		case <-pow.sigc:
			pow.ctx.BCtx.XLog.Info("PoW::mining::be killed by new consensus or internal error")
			return OODMineErr
		default:
		}
		pow.status.mutex.RLock()
		newHeight := pow.status.newHeight
		pow.status.mutex.RUnlock()
		if newHeight >= block.GetHeight() {
			return OODMineErr
		}
		if tries == 0 {
			return TryTooMuchMineErr
		}
		if err := block.SetItem("nonce", gussNonce); err != nil {
			return PoWBlockItemErr
		}
		if pow.IsProofed(block.MakeBlockId(), pow.targetBits) {
			block.SetItem("blockid", block.MakeBlockId())
			return nil
		}
		gussNonce++
		tries--
	}
}

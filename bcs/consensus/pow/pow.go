package pow

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"

	"github.com/xuperchain/xupercore/kernel/common/xcontext"
	"github.com/xuperchain/xupercore/kernel/consensus"
	"github.com/xuperchain/xupercore/kernel/consensus/base"
	"github.com/xuperchain/xupercore/kernel/consensus/context"
	"github.com/xuperchain/xupercore/kernel/consensus/def"
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

// PoWConsensus pow具体结构
type PoWConsensus struct {
	// Pluggable Consensus传递的上下文, PoW并不使用P2p interface
	ctx    context.ConsensusCtx
	status *PoWStatus
	config *PoWConfig

	targetBits    uint32
	sigc          chan bool
	maxDifficulty *big.Int
}

// NewPoWConsensus 初始化实例
func NewPoWConsensus(cCtx context.ConsensusCtx, cCfg def.ConsensusConfig) base.ConsensusImplInterface {
	// 解析config中需要的字段
	if cCtx.XLog == nil {
		return nil
	}
	// TODO:cCtx.BcName需要注册表吗？
	if cCtx.Crypto == nil || cCtx.Address == nil {
		cCtx.XLog.Error("PoW::NewPoWConsensus::CryptoClient in context is nil")
		return nil
	}
	if cCtx.Ledger == nil {
		cCtx.XLog.Error("PoW::NewPoWConsensus::Ledger in context is nil")
		return nil
	}
	if cCfg.ConsensusName != "pow" {
		cCtx.XLog.Error("PoW::NewPoWConsensus::consensus name in config is wrong", "name", cCfg.ConsensusName)
		return nil
	}
	config, err := unmarshalPowConfig([]byte(cCfg.Config))
	if err != nil {
		cCtx.XLog.Error("PoW::NewPoWConsensus::pow struct unmarshal error", "error", err)
		return nil
	}
	// newHeight取上一共识的最高值，因为此时BeginHeight也许并为生产出来
	pow := &PoWConsensus{
		ctx:    cCtx,
		config: config,
		status: &PoWStatus{
			startHeight: cCfg.StartHeight,
			newHeight:   cCfg.StartHeight - 1,
			index:       cCfg.Index,
			miner: ValidatorsInfo{
				Validators: []string{cCtx.Address.Address},
			},
		},
		sigc: make(chan bool, 1),
	}
	target := config.DefaultTarget
	// 重启时需要重新更新目标target
	if cCtx.Ledger.GetTipBlock().GetHeight() > cCfg.StartHeight {
		bits, err := pow.refreshDifficulty(cCtx.Ledger.GetTipBlock().GetBlockid(), cCtx.Ledger.GetTipBlock().GetHeight()+1)
		if err != nil {
			cCtx.XLog.Error("PoW::NewPoWConsensus::refreshDifficulty err", "error", err)
			return nil
		}
		target = bits
		cCtx.XLog.Debug("PoW::NewPoWConsensus::refreshDifficulty after restart.")
	}
	// 通过MaxTarget和DefaultTarget解析maxDifficulty和DefaultDifficulty
	md, fNegative, fOverflow := SetCompact(config.MaxTarget)
	if fNegative || fOverflow {
		cCtx.XLog.Error("PoW::NewPoWConsensus::pow set MaxTarget error", "fNegative", fNegative, "fOverflow", fOverflow)
		return nil
	}
	_, fNegative, fOverflow = SetCompact(target)
	if fNegative || fOverflow {
		cCtx.XLog.Error("PoW::NewPoWConsensus::pow set Default error", "fNegative", fNegative, "fOverflow", fOverflow)
		return nil
	}
	pow.targetBits = target
	pow.maxDifficulty = md
	cCtx.XLog.Debug("Pow::NewPoWConsensus::create a pow instance successfully.", "pow", pow)
	return pow
}

// ParseConsensusStorage PoW parse专有存储的逻辑，即targetBits
func (pow *PoWConsensus) ParseConsensusStorage(block context.BlockInterface) (interface{}, error) {
	store := PoWStorage{}
	b, err := block.GetConsensusStorage()
	if err != nil {
		return nil, err
	}
	err = json.Unmarshal(b, &store)
	if err != nil {
		pow.ctx.XLog.Error("PoW::ParseConsensusStorage invalid consensus storage", "err", err)
		return nil, err
	}
	return store, nil
}

// CalculateBlock 挖矿过程
func (pow *PoWConsensus) CalculateBlock(block context.BlockInterface) error {
	return pow.mining(block)
}

// CompeteMaster PoW单一节点都为矿工，故返回为true
func (pow *PoWConsensus) CompeteMaster(height int64) (bool, bool, error) {
	pow.ctx.XLog.Debug("PoW::CompeteMaster", "targetBits", pow.targetBits)
	return true, true, nil
}

// CheckMinerMatch 验证区块，包括merkel根和hash
// ATTENTION: TODO: 上层需要先检查VerifyMerkle(block)
func (pow *PoWConsensus) CheckMinerMatch(ctx xcontext.XContext, block context.BlockInterface) (bool, error) {
	// 检查区块是否有targetBits字段
	in, err := pow.ParseConsensusStorage(block)
	if err != nil {
		ctx.GetLog().Warn("PoW::CheckMinerMatch::ParseConsensusStorage err", "err", err,
			"blockId", block.GetBlockid(), "miner", string(block.GetProposer()))
		return false, err
	}
	s, ok := in.(PoWStorage)
	if !ok {
		ctx.GetLog().Warn("PoW::CheckMinerMatch::transfer PoWStorage err", "blockId", block.GetBlockid(), "miner", string(block.GetProposer()))
		return false, err
	}
	// 检查区块的区块头是否和和区块中的targetBits字段匹配
	if !pow.IsProofed(block.GetBlockid(), s.TargetBits) {
		ctx.GetLog().Warn("PoW::CheckMinerMatch::the actual difficulty of block received doesn't match its' blockid",
			"blockid", fmt.Sprintf("%x", block.GetBlockid()), "miner", string(block.GetProposer()))
		return false, err
	}
	// 检查区块的区块头是否hash正确
	id, err := block.MakeBlockId()
	if err != nil {
		ctx.GetLog().Warn("PoW::CheckMinerMatch::make blockid error", "error", err, "miner", string(block.GetProposer()))
		return false, err
	}
	if !bytes.Equal(id, block.GetBlockid()) {
		ctx.GetLog().Warn("PoW::CheckMinerMatch::equal blockid error", "miner", string(block.GetProposer()))
		return false, err
	}
	// 验证difficulty是否正确
	targetBits, err := pow.refreshDifficulty(block.GetPreHash(), block.GetHeight())
	if err != nil {
		ctx.GetLog().Warn("PoW::CheckMinerMatch::refreshDifficulty err", "error", err, "miner", string(block.GetProposer()))
		return false, err
	}
	if targetBits != s.TargetBits {
		ctx.GetLog().Warn("PoW::CheckMinerMatch::unexpected target bits", "expect", targetBits, "got", s.TargetBits, "miner", string(block.GetProposer()))
		return false, err
	}
	// 验证时间戳是否正确
	preBlock, err := pow.ctx.Ledger.QueryBlock(block.GetPreHash())
	if err != nil {
		ctx.GetLog().Warn("PoW::CheckMinerMatch::get preblock error", "miner", string(block.GetProposer()))
		return false, err
	}
	if block.GetTimestamp() < preBlock.GetTimestamp() {
		ctx.GetLog().Warn("PoW::CheckMinerMatch::unexpected block timestamp",
			"pre", preBlock.GetTimestamp(), "next", block.GetTimestamp(), "miner", string(block.GetProposer()))
		return false, err
	}
	// 验证前导0
	if !pow.IsProofed(block.GetBlockid(), targetBits) {
		ctx.GetLog().Warn("PoW::CheckMinerMatch::blockid IsProofed error", "miner", string(block.GetProposer()))
		return false, err
	}
	// 验证签名
	// 1 验证一下签名和公钥是不是能对上
	k, err := pow.ctx.Crypto.GetEcdsaPublicKeyFromJsonStr(block.GetPublicKey())
	if err != nil {
		ctx.GetLog().Warn("PoW::CheckMinerMatch::get ecdsa from block error", "error", err, "miner", string(block.GetProposer()))
		return false, err
	}
	// 跟address比较
	chkResult, _ := pow.ctx.Crypto.VerifyAddressUsingPublicKey(string(block.GetProposer()), k)
	if chkResult == false {
		ctx.GetLog().Warn("PoW::CheckMinerMatch::address is not match publickey", "miner", string(block.GetProposer()))
		return false, err
	}
	// 2 验证一下签名是否正确
	valid, err := pow.ctx.Crypto.VerifyECDSA(k, block.GetSign(), block.GetBlockid())
	if err != nil {
		ctx.GetLog().Warn("PoW::CheckMinerMatch::verifyECDSA error", "error", err, "miner", string(block.GetProposer()))
	}
	return valid, err
}

// ProcessBeforeMiner 更新下一次pow挖矿时的targetBits
func (pow *PoWConsensus) ProcessBeforeMiner(timestamp int64) ([]byte, []byte, error) {
	pow.status.mutex.Lock()
	tipHeight := pow.status.newHeight
	pow.status.mutex.Unlock()
	preBlock, err := pow.ctx.Ledger.QueryBlockByHeight(tipHeight)
	if err != nil {
		pow.ctx.XLog.Error("PoW::ProcessBeforeMiner::cannnot find preBlock", "logid", pow.ctx.XLog.GetLogId())
		return nil, nil, InternalErr
	}
	bits, err := pow.refreshDifficulty(preBlock.GetBlockid(), tipHeight+1)
	if err != nil {
		pow.Stop()
	}
	pow.targetBits = bits
	store := &PoWStorage{
		TargetBits: bits,
	}
	by, err := json.Marshal(store)
	if err != nil {
		return nil, nil, err
	}
	return nil, by, nil
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

// refreshDifficulty 计算difficulty in bitcoin
// reference of bitcoin's pow: https://github.com/bitcoin/bitcoin/blob/master/src/pow.cpp#L49
func (pow *PoWConsensus) refreshDifficulty(tipHash []byte, nextHeight int64) (uint32, error) {
	// 未到调整高度0 + Gap，直接返回default
	if nextHeight <= int64(pow.config.AdjustHeightGap) {
		return pow.config.DefaultTarget, nil
	}
	// 检查block结构是否合法，获取上一区块difficulty
	preBlock, err := pow.ctx.Ledger.QueryBlock(tipHash)
	if err != nil {
		return pow.config.DefaultTarget, nil
	}
	in, err := pow.ParseConsensusStorage(preBlock)
	if err != nil {
		pow.ctx.XLog.Error("PoW::refreshDifficulty::ParseConsensusStorage err", "err", err, "blockId", tipHash)
		return 0, err
	}
	s, ok := in.(PoWStorage)
	if !ok {
		pow.ctx.XLog.Error("PoW::refreshDifficulty::transfer PoWStorage err")
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
		prevBlock, err := pow.ctx.Ledger.QueryBlock(farBlock.GetPreHash())
		if err != nil {
			return pow.config.DefaultTarget, nil
		}
		farBlock = prevBlock
	}
	expectedTimeSpan := pow.config.ExpectedPeriodMilSec * pow.config.AdjustHeightGap
	// ATTENTION: 此处并没有针对任意的Timestamp类型，目前只能是timestamp为nano类型
	actualTimeSpan := int32((preBlock.GetTimestamp() - farBlock.GetTimestamp()) / 1e6)
	pow.ctx.XLog.Debug("PoW::refreshDifficulty::timespan diff", "expectedTimeSpan", expectedTimeSpan, "actualTimeSpan", actualTimeSpan)
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
	if difficulty.Cmp(pow.maxDifficulty) == -1 {
		pow.ctx.XLog.Debug("PoW::refreshDifficulty::retarget", "newTargetBits", pow.config.MaxTarget)
		return pow.config.MaxTarget, nil
	}
	newTargetBits, ok := GetCompact(difficulty)
	if !ok {
		pow.ctx.XLog.Error("PoW::refreshDifficulty::difficulty GetCompact err")
		return prevTargetBits, nil
	}
	pow.ctx.XLog.Debug("PoW::refreshDifficulty::adjust targetBits", "height", nextHeight, "targetBits", newTargetBits, "prevTargetBits", prevTargetBits)
	return newTargetBits, nil
}

//IsProofed check workload proof
func (pow *PoWConsensus) IsProofed(blockID []byte, targetBits uint32) bool {
	d, fNegative, fOverflow := SetCompact(targetBits)
	if fNegative || fOverflow || d.Cmp(pow.maxDifficulty) == -1 { // d > maxDifficulty
		return false
	}
	hash := new(big.Int)
	hash.SetBytes(blockID)
	if hash.Cmp(d) == 1 { // hash > d
		return false
	}
	return true
}

// mining 为带副作用的函数，将直接对block进行操作，更改其原始值
func (pow *PoWConsensus) mining(block context.BlockInterface) error {
	gussNonce := int32(0)
	tries := MAX_TRIES
	for {
		select {
		case <-pow.sigc:
			pow.ctx.XLog.Debug("PoW::mining::be killed by new consensus or internal error")
			return OODMineErr
		default:
		}
		pow.status.mutex.Lock()
		newHeight := pow.status.newHeight
		pow.status.mutex.Unlock()
		if newHeight >= block.GetHeight() {
			return OODMineErr
		}
		if tries == 0 {
			return TryTooMuchMineErr
		}
		if err := block.SetItem("nonce", gussNonce); err != nil {
			return PoWBlockItemErr
		}
		bid, err := block.MakeBlockId()
		if err != nil {
			continue
		}
		if pow.IsProofed(bid, pow.targetBits) {
			block.SetItem("blockid", bid)
			return nil
		}
		gussNonce++
		tries--
	}
}

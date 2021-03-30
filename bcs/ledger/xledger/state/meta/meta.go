package meta

import (
	"errors"
	"fmt"
	"sync"

	"github.com/golang/protobuf/proto"
	"github.com/xuperchain/xupercore/bcs/ledger/xledger/def"
	"github.com/xuperchain/xupercore/bcs/ledger/xledger/ledger"
	"github.com/xuperchain/xupercore/bcs/ledger/xledger/state/context"
	pb "github.com/xuperchain/xupercore/bcs/ledger/xledger/xldgpb"
	"github.com/xuperchain/xupercore/lib/logs"
	"github.com/xuperchain/xupercore/lib/storage/kvdb"
	"github.com/xuperchain/xupercore/protos"
)

type Meta struct {
	log       logs.Logger
	Ledger    *ledger.Ledger
	Meta      *pb.UtxoMeta  // utxo meta
	MetaTmp   *pb.UtxoMeta  // tmp utxo meta
	MutexMeta *sync.Mutex   // access control for meta
	MetaTable kvdb.Database // 元数据表，会持久化保存latestBlockid
}

var (
	ErrProposalParamsIsNegativeNumber    = errors.New("negative number for proposal parameter is not allowed")
	ErrProposalParamsIsNotPositiveNumber = errors.New("negative number of zero for proposal parameter is not allowed")
	ErrGetReservedContracts              = errors.New("Get reserved contracts error")
	// TxSizePercent max percent of txs' size in one block
	TxSizePercent = 0.8
)

// reservedArgs used to get contractnames from InvokeRPCRequest
type reservedArgs struct {
	ContractNames string
}

func NewMeta(sctx *context.StateCtx, stateDB kvdb.Database) (*Meta, error) {
	obj := &Meta{
		log:       sctx.XLog,
		Ledger:    sctx.Ledger,
		Meta:      &pb.UtxoMeta{},
		MetaTmp:   &pb.UtxoMeta{},
		MutexMeta: &sync.Mutex{},
		MetaTable: kvdb.NewTable(stateDB, pb.MetaTablePrefix),
	}

	var loadErr error
	// load consensus parameters
	obj.Meta.MaxBlockSize, loadErr = obj.LoadMaxBlockSize()
	if loadErr != nil {
		sctx.XLog.Warn("failed to load maxBlockSize from disk", "loadErr", loadErr)
		return nil, loadErr
	}
	obj.Meta.ForbiddenContract, loadErr = obj.LoadForbiddenContract()
	if loadErr != nil {
		sctx.XLog.Warn("failed to load forbiddenContract from disk", "loadErr", loadErr)
		return nil, loadErr
	}
	obj.Meta.ReservedContracts, loadErr = obj.LoadReservedContracts()
	if loadErr != nil {
		sctx.XLog.Warn("failed to load reservedContracts from disk", "loadErr", loadErr)
		return nil, loadErr
	}
	obj.Meta.NewAccountResourceAmount, loadErr = obj.LoadNewAccountResourceAmount()
	if loadErr != nil {
		sctx.XLog.Warn("failed to load newAccountResourceAmount from disk", "loadErr", loadErr)
		return nil, loadErr
	}
	// load irreversible block height & slide window parameters
	obj.Meta.IrreversibleBlockHeight, loadErr = obj.LoadIrreversibleBlockHeight()
	if loadErr != nil {
		sctx.XLog.Warn("failed to load irreversible block height from disk", "loadErr", loadErr)
		return nil, loadErr
	}
	obj.Meta.IrreversibleSlideWindow, loadErr = obj.LoadIrreversibleSlideWindow()
	if loadErr != nil {
		sctx.XLog.Warn("failed to load irreversibleSlide window from disk", "loadErr", loadErr)
		return nil, loadErr
	}
	// load gas price
	obj.Meta.GasPrice, loadErr = obj.LoadGasPrice()
	if loadErr != nil {
		sctx.XLog.Warn("failed to load gas price from disk", "loadErr", loadErr)
		return nil, loadErr
	}
	// load group chain
	obj.Meta.GroupChainContract, loadErr = obj.LoadGroupChainContract()
	if loadErr != nil {
		sctx.XLog.Warn("failed to load groupchain from disk", "loadErr", loadErr)
		return nil, loadErr
	}
	newMeta := proto.Clone(obj.Meta).(*pb.UtxoMeta)
	obj.MetaTmp = newMeta

	return obj, nil
}

// GetNewAccountResourceAmount get account for creating an account
func (t *Meta) GetNewAccountResourceAmount() int64 {
	t.MutexMeta.Lock()
	defer t.MutexMeta.Unlock()
	return t.Meta.GetNewAccountResourceAmount()
}

// LoadNewAccountResourceAmount load newAccountResourceAmount into memory
func (t *Meta) LoadNewAccountResourceAmount() (int64, error) {
	newAccountResourceAmountBuf, findErr := t.MetaTable.Get([]byte(ledger.NewAccountResourceAmountKey))
	if findErr == nil {
		utxoMeta := &pb.UtxoMeta{}
		err := proto.Unmarshal(newAccountResourceAmountBuf, utxoMeta)
		return utxoMeta.GetNewAccountResourceAmount(), err
	} else if def.NormalizedKVError(findErr) == def.ErrKVNotFound {
		genesisNewAccountResourceAmount := t.Ledger.GetNewAccountResourceAmount()
		if genesisNewAccountResourceAmount < 0 {
			return genesisNewAccountResourceAmount, ErrProposalParamsIsNegativeNumber
		}
		return genesisNewAccountResourceAmount, nil
	}

	return int64(0), findErr
}

// UpdateNewAccountResourceAmount ...
func (t *Meta) UpdateNewAccountResourceAmount(newAccountResourceAmount int64, batch kvdb.Batch) error {
	if newAccountResourceAmount < 0 {
		return ErrProposalParamsIsNegativeNumber
	}
	tmpMeta := &pb.UtxoMeta{}
	newMeta := proto.Clone(tmpMeta).(*pb.UtxoMeta)
	newMeta.NewAccountResourceAmount = newAccountResourceAmount
	newAccountResourceAmountBuf, pbErr := proto.Marshal(newMeta)
	if pbErr != nil {
		t.log.Warn("failed to marshal pb meta")
		return pbErr
	}
	err := batch.Put([]byte(pb.MetaTablePrefix+ledger.NewAccountResourceAmountKey), newAccountResourceAmountBuf)
	if err == nil {
		t.log.Info("Update newAccountResourceAmount succeed")
	}
	t.MutexMeta.Lock()
	defer t.MutexMeta.Unlock()
	t.MetaTmp.NewAccountResourceAmount = newAccountResourceAmount
	return err
}

// GetMaxBlockSize get max block size effective in Utxo
func (t *Meta) GetMaxBlockSize() int64 {
	t.MutexMeta.Lock()
	defer t.MutexMeta.Unlock()
	return t.Meta.GetMaxBlockSize()
}

// LoadMaxBlockSize load maxBlockSize into memory
func (t *Meta) LoadMaxBlockSize() (int64, error) {
	maxBlockSizeBuf, findErr := t.MetaTable.Get([]byte(ledger.MaxBlockSizeKey))
	if findErr == nil {
		utxoMeta := &pb.UtxoMeta{}
		err := proto.Unmarshal(maxBlockSizeBuf, utxoMeta)
		return utxoMeta.GetMaxBlockSize(), err
	} else if def.NormalizedKVError(findErr) == def.ErrKVNotFound {
		genesisMaxBlockSize := t.Ledger.GetMaxBlockSize()
		if genesisMaxBlockSize <= 0 {
			return genesisMaxBlockSize, ErrProposalParamsIsNotPositiveNumber
		}
		return genesisMaxBlockSize, nil
	}

	return int64(0), findErr
}

func (t *Meta) MaxTxSizePerBlock() (int, error) {
	maxBlkSize := t.GetMaxBlockSize()
	return int(float64(maxBlkSize) * TxSizePercent), nil
}

func (t *Meta) UpdateMaxBlockSize(maxBlockSize int64, batch kvdb.Batch) error {
	if maxBlockSize <= 0 {
		return ErrProposalParamsIsNotPositiveNumber
	}
	tmpMeta := &pb.UtxoMeta{}
	newMeta := proto.Clone(tmpMeta).(*pb.UtxoMeta)
	newMeta.MaxBlockSize = maxBlockSize
	maxBlockSizeBuf, pbErr := proto.Marshal(newMeta)
	if pbErr != nil {
		t.log.Warn("failed to marshal pb meta")
		return pbErr
	}
	err := batch.Put([]byte(pb.MetaTablePrefix+ledger.MaxBlockSizeKey), maxBlockSizeBuf)
	if err == nil {
		t.log.Info("Update maxBlockSize succeed")
	}
	t.MutexMeta.Lock()
	defer t.MutexMeta.Unlock()
	t.MetaTmp.MaxBlockSize = maxBlockSize
	return err
}

func (t *Meta) GetReservedContracts() []*protos.InvokeRequest {
	t.MutexMeta.Lock()
	defer t.MutexMeta.Unlock()
	return t.Meta.ReservedContracts
}

func (t *Meta) LoadReservedContracts() ([]*protos.InvokeRequest, error) {
	reservedContractsBuf, findErr := t.MetaTable.Get([]byte(ledger.ReservedContractsKey))
	if findErr == nil {
		utxoMeta := &pb.UtxoMeta{}
		err := proto.Unmarshal(reservedContractsBuf, utxoMeta)
		return utxoMeta.GetReservedContracts(), err
	} else if def.NormalizedKVError(findErr) == def.ErrKVNotFound {
		return t.Ledger.GetReservedContracts()
	}
	return nil, findErr
}

//when to register to kernel method
func (t *Meta) UpdateReservedContracts(params []*protos.InvokeRequest, batch kvdb.Batch) error {
	if params == nil {
		return fmt.Errorf("invalid reservered contract requests")
	}
	tmpNewMeta := &pb.UtxoMeta{}
	newMeta := proto.Clone(tmpNewMeta).(*pb.UtxoMeta)
	newMeta.ReservedContracts = params
	paramsBuf, pbErr := proto.Marshal(newMeta)
	if pbErr != nil {
		t.log.Warn("failed to marshal pb meta")
		return pbErr
	}
	err := batch.Put([]byte(pb.MetaTablePrefix+ledger.ReservedContractsKey), paramsBuf)
	if err == nil {
		t.log.Info("Update reservered contract succeed")
	}
	t.MutexMeta.Lock()
	defer t.MutexMeta.Unlock()
	t.MetaTmp.ReservedContracts = params
	return err
}

func (t *Meta) GetForbiddenContract() *protos.InvokeRequest {
	t.MutexMeta.Lock()
	defer t.MutexMeta.Unlock()
	return t.Meta.GetForbiddenContract()
}

func (t *Meta) GetGroupChainContract() *protos.InvokeRequest {
	t.MutexMeta.Lock()
	defer t.MutexMeta.Unlock()
	return t.Meta.GetGroupChainContract()
}

func (t *Meta) LoadGroupChainContract() (*protos.InvokeRequest, error) {
	groupChainContractBuf, findErr := t.MetaTable.Get([]byte(ledger.GroupChainContractKey))
	if findErr == nil {
		utxoMeta := &pb.UtxoMeta{}
		err := proto.Unmarshal(groupChainContractBuf, utxoMeta)
		return utxoMeta.GetGroupChainContract(), err
	} else if def.NormalizedKVError(findErr) == def.ErrKVNotFound {
		requests, err := t.Ledger.GetGroupChainContract()
		if len(requests) > 0 {
			return requests[0], err
		}
		return nil, errors.New("unexpected error")
	}
	return nil, findErr
}

func (t *Meta) LoadForbiddenContract() (*protos.InvokeRequest, error) {
	forbiddenContractBuf, findErr := t.MetaTable.Get([]byte(ledger.ForbiddenContractKey))
	if findErr == nil {
		utxoMeta := &pb.UtxoMeta{}
		err := proto.Unmarshal(forbiddenContractBuf, utxoMeta)
		return utxoMeta.GetForbiddenContract(), err
	} else if def.NormalizedKVError(findErr) == def.ErrKVNotFound {
		requests, err := t.Ledger.GetForbiddenContract()
		if len(requests) > 0 {
			return requests[0], err
		}
		return nil, errors.New("unexpected error")
	}
	return nil, findErr
}

func (t *Meta) UpdateForbiddenContract(param *protos.InvokeRequest, batch kvdb.Batch) error {
	if param == nil {
		return fmt.Errorf("invalid forbidden contract request")
	}
	tmpNewMeta := &pb.UtxoMeta{}
	newMeta := proto.Clone(tmpNewMeta).(*pb.UtxoMeta)
	newMeta.ForbiddenContract = param
	paramBuf, pbErr := proto.Marshal(newMeta)
	if pbErr != nil {
		t.log.Warn("failed to marshal pb meta")
		return pbErr
	}
	err := batch.Put([]byte(pb.MetaTablePrefix+ledger.ForbiddenContractKey), paramBuf)
	if err == nil {
		t.log.Info("Update forbidden contract succeed")
	}
	t.MutexMeta.Lock()
	defer t.MutexMeta.Unlock()
	t.MetaTmp.ForbiddenContract = param
	return err
}

func (t *Meta) LoadIrreversibleBlockHeight() (int64, error) {
	irreversibleBlockHeightBuf, findErr := t.MetaTable.Get([]byte(ledger.IrreversibleBlockHeightKey))
	if findErr == nil {
		utxoMeta := &pb.UtxoMeta{}
		err := proto.Unmarshal(irreversibleBlockHeightBuf, utxoMeta)
		return utxoMeta.GetIrreversibleBlockHeight(), err
	} else if def.NormalizedKVError(findErr) == def.ErrKVNotFound {
		return int64(0), nil
	}
	return int64(0), findErr
}

func (t *Meta) LoadIrreversibleSlideWindow() (int64, error) {
	irreversibleSlideWindowBuf, findErr := t.MetaTable.Get([]byte(ledger.IrreversibleSlideWindowKey))
	if findErr == nil {
		utxoMeta := &pb.UtxoMeta{}
		err := proto.Unmarshal(irreversibleSlideWindowBuf, utxoMeta)
		return utxoMeta.GetIrreversibleSlideWindow(), err
	} else if def.NormalizedKVError(findErr) == def.ErrKVNotFound {
		genesisSlideWindow := t.Ledger.GetIrreversibleSlideWindow()
		// negative number is not meaningful
		if genesisSlideWindow < 0 {
			return genesisSlideWindow, ErrProposalParamsIsNegativeNumber
		}
		return genesisSlideWindow, nil
	}
	return int64(0), findErr
}

func (t *Meta) GetIrreversibleBlockHeight() int64 {
	t.MutexMeta.Lock()
	defer t.MutexMeta.Unlock()
	return t.Meta.IrreversibleBlockHeight
}

func (t *Meta) GetIrreversibleSlideWindow() int64 {
	t.MutexMeta.Lock()
	defer t.MutexMeta.Unlock()
	return t.Meta.IrreversibleSlideWindow
}

func (t *Meta) UpdateIrreversibleBlockHeight(nextIrreversibleBlockHeight int64, batch kvdb.Batch) error {
	tmpMeta := &pb.UtxoMeta{}
	newMeta := proto.Clone(tmpMeta).(*pb.UtxoMeta)
	newMeta.IrreversibleBlockHeight = nextIrreversibleBlockHeight
	irreversibleBlockHeightBuf, pbErr := proto.Marshal(newMeta)
	if pbErr != nil {
		t.log.Warn("failed to marshal pb meta")
		return pbErr
	}
	err := batch.Put([]byte(pb.MetaTablePrefix+ledger.IrreversibleBlockHeightKey), irreversibleBlockHeightBuf)
	if err != nil {
		return err
	}
	t.log.Info("Update irreversibleBlockHeight succeed")
	t.MutexMeta.Lock()
	defer t.MutexMeta.Unlock()
	t.MetaTmp.IrreversibleBlockHeight = nextIrreversibleBlockHeight
	return nil
}

func (t *Meta) UpdateNextIrreversibleBlockHeight(blockHeight int64, curIrreversibleBlockHeight int64, curIrreversibleSlideWindow int64, batch kvdb.Batch) error {
	// negative number for irreversible slide window is not allowed.
	if curIrreversibleSlideWindow < 0 {
		return ErrProposalParamsIsNegativeNumber
	}
	// slideWindow为开启,不需要更新IrreversibleBlockHeight
	if curIrreversibleSlideWindow == 0 {
		return nil
	}
	// curIrreversibleBlockHeight小于0, 不符合预期，报警
	if curIrreversibleBlockHeight < 0 {
		t.log.Warn("update irreversible block height error, should be here")
		return errors.New("curIrreversibleBlockHeight is less than 0")
	}
	nextIrreversibleBlockHeight := blockHeight - curIrreversibleSlideWindow
	// 下一个不可逆高度小于当前不可逆高度，直接返回
	// slideWindow变大或者发生区块回滚
	if nextIrreversibleBlockHeight <= curIrreversibleBlockHeight {
		return nil
	}
	// 正常升级
	// slideWindow不变或变小
	if nextIrreversibleBlockHeight > curIrreversibleBlockHeight {
		err := t.UpdateIrreversibleBlockHeight(nextIrreversibleBlockHeight, batch)
		return err
	}

	return errors.New("unexpected error")
}

func (t *Meta) UpdateNextIrreversibleBlockHeightForPrune(blockHeight int64, curIrreversibleBlockHeight int64, curIrreversibleSlideWindow int64, batch kvdb.Batch) error {
	// negative number for irreversible slide window is not allowed.
	if curIrreversibleSlideWindow < 0 {
		return ErrProposalParamsIsNegativeNumber
	}
	// slideWindow为开启,不需要更新IrreversibleBlockHeight
	if curIrreversibleSlideWindow == 0 {
		return nil
	}
	// curIrreversibleBlockHeight小于0, 不符合预期，报警
	if curIrreversibleBlockHeight < 0 {
		t.log.Warn("update irreversible block height error, should be here")
		return errors.New("curIrreversibleBlockHeight is less than 0")
	}
	nextIrreversibleBlockHeight := blockHeight - curIrreversibleSlideWindow
	if nextIrreversibleBlockHeight <= 0 {
		nextIrreversibleBlockHeight = 0
	}
	err := t.UpdateIrreversibleBlockHeight(nextIrreversibleBlockHeight, batch)
	return err
}

func (t *Meta) UpdateIrreversibleSlideWindow(nextIrreversibleSlideWindow int64, batch kvdb.Batch) error {
	if nextIrreversibleSlideWindow < 0 {
		return ErrProposalParamsIsNegativeNumber
	}
	tmpMeta := &pb.UtxoMeta{}
	newMeta := proto.Clone(tmpMeta).(*pb.UtxoMeta)
	newMeta.IrreversibleSlideWindow = nextIrreversibleSlideWindow
	irreversibleSlideWindowBuf, pbErr := proto.Marshal(newMeta)
	if pbErr != nil {
		t.log.Warn("failed to marshal pb meta")
		return pbErr
	}
	err := batch.Put([]byte(pb.MetaTablePrefix+ledger.IrreversibleSlideWindowKey), irreversibleSlideWindowBuf)
	if err != nil {
		return err
	}
	t.log.Info("Update irreversibleSlideWindow succeed")
	t.MutexMeta.Lock()
	defer t.MutexMeta.Unlock()
	t.MetaTmp.IrreversibleSlideWindow = nextIrreversibleSlideWindow
	return nil
}

// GetGasPrice get gas rate to utxo
func (t *Meta) GetGasPrice() *protos.GasPrice {
	t.MutexMeta.Lock()
	defer t.MutexMeta.Unlock()
	return t.Meta.GetGasPrice()
}

// LoadGasPrice load gas rate
func (t *Meta) LoadGasPrice() (*protos.GasPrice, error) {
	gasPriceBuf, findErr := t.MetaTable.Get([]byte(ledger.GasPriceKey))
	if findErr == nil {
		utxoMeta := &pb.UtxoMeta{}
		err := proto.Unmarshal(gasPriceBuf, utxoMeta)
		return utxoMeta.GetGasPrice(), err
	} else if def.NormalizedKVError(findErr) == def.ErrKVNotFound {
		nofee := t.Ledger.GetNoFee()
		if nofee {
			gasPrice := &protos.GasPrice{
				CpuRate:  0,
				MemRate:  0,
				DiskRate: 0,
				XfeeRate: 0,
			}
			return gasPrice, nil

		} else {
			gasPrice := t.Ledger.GetGasPrice()
			cpuRate := gasPrice.CpuRate
			memRate := gasPrice.MemRate
			diskRate := gasPrice.DiskRate
			xfeeRate := gasPrice.XfeeRate
			if cpuRate < 0 || memRate < 0 || diskRate < 0 || xfeeRate < 0 {
				return nil, ErrProposalParamsIsNegativeNumber
			}
			// To be compatible with the old version v3.3
			// If GasPrice configuration is missing or value euqals 0, support a default value
			if cpuRate == 0 && memRate == 0 && diskRate == 0 && xfeeRate == 0 {
				gasPrice = &protos.GasPrice{
					CpuRate:  1000,
					MemRate:  1000000,
					DiskRate: 1,
					XfeeRate: 1,
				}
			}
			return gasPrice, nil
		}
	}
	return nil, findErr
}

// UpdateGasPrice update gasPrice parameters
func (t *Meta) UpdateGasPrice(nextGasPrice *protos.GasPrice, batch kvdb.Batch) error {
	// check if the parameters are valid
	cpuRate := nextGasPrice.GetCpuRate()
	memRate := nextGasPrice.GetMemRate()
	diskRate := nextGasPrice.GetDiskRate()
	xfeeRate := nextGasPrice.GetXfeeRate()
	if cpuRate < 0 || memRate < 0 || diskRate < 0 || xfeeRate < 0 {
		return ErrProposalParamsIsNegativeNumber
	}
	tmpMeta := &pb.UtxoMeta{}
	newMeta := proto.Clone(tmpMeta).(*pb.UtxoMeta)
	newMeta.GasPrice = nextGasPrice
	gasPriceBuf, pbErr := proto.Marshal(newMeta)
	if pbErr != nil {
		t.log.Warn("failed to marshal pb meta")
		return pbErr
	}
	err := batch.Put([]byte(pb.MetaTablePrefix+ledger.GasPriceKey), gasPriceBuf)
	if err != nil {
		return err
	}
	t.log.Info("Update gas price succeed")
	t.MutexMeta.Lock()
	defer t.MutexMeta.Unlock()
	t.MetaTmp.GasPrice = nextGasPrice
	return nil
}

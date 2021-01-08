package meta

import (
	"errors"
	"html/template"
	"sync"

	"github.com/golang/protobuf/proto"
	"github.com/xuperchain/xupercore/bcs/ledger/xledger/def"
	"github.com/xuperchain/xupercore/bcs/ledger/xledger/ledger"
	pb "github.com/xuperchain/xupercore/bcs/ledger/xledger/xldgpb"
	"github.com/xuperchain/xupercore/lib/logs"
	"github.com/xuperchain/xupercore/lib/storage/kvdb"
)

type Meta struct {
	log       logs.Logger
	Meta      *pb.UtxoMeta  // utxo meta
	MetaTmp   *pb.UtxoMeta  // tmp utxo meta
	MutexMeta *sync.Mutex   // access control for meta
	MetaTable kvdb.Database // 元数据表，会持久化保存latestBlockid
}

var (
	ErrProposalParamsIsNegativeNumber    = errors.New("negative number for proposal parameter is not allowed")
	ErrProposalParamsIsNotPositiveNumber = errors.New("negative number of zero for proposal parameter is not allowed")
	// TxSizePercent max percent of txs' size in one block
	TxSizePercent = 0.8
)

func NewMeta(sctx *def.StateCtx, stateDB kvdb.Database) (*Meta, nil) {
	return &Meta{
		log:       sctx.XLog,
		meta:      &pb.UtxoMeta{},
		metaTmp:   &pb.UtxoMeta{},
		mutexMeta: &sync.Map{},
		metaTable: kvdb.NewTable(stateDB, xldgpb.MetaTablePrefix),
	}, nil
}

// GetNewAccountResourceAmount get account for creating an account
func (t *Meta) GetNewAccountResourceAmount() int64 {
	t.mutexMeta.Lock()
	defer t.mutexMeta.Unlock()
	return t.meta.GetNewAccountResourceAmount()
}

// LoadNewAccountResourceAmount load newAccountResourceAmount into memory
func (t *Meta) LoadNewAccountResourceAmount() (int64, error) {
	newAccountResourceAmountBuf, findErr := t.metaTable.Get([]byte(ledger.NewAccountResourceAmountKey))
	if findErr == nil {
		utxoMeta := &pb.UtxoMeta{}
		err := proto.Unmarshal(newAccountResourceAmountBuf, utxoMeta)
		return utxoMeta.GetNewAccountResourceAmount(), err
	} else if common.NormalizedKVError(findErr) == common.ErrKVNotFound {
		genesisNewAccountResourceAmount := ledger.ledger.GetNewAccountResourceAmount()
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
	err := batch.Put([]byte(xldgpb.MetaTablePrefix+ledger.NewAccountResourceAmountKey), newAccountResourceAmountBuf)
	if err == nil {
		t.log.Info("Update newAccountResourceAmount succeed")
	}
	t.mutexMeta.Lock()
	defer t.mutexMeta.Unlock()
	t.metaTmp.NewAccountResourceAmount = newAccountResourceAmount
	return err
}

// GetMaxBlockSize get max block size effective in Utxo
func (t *Meta) GetMaxBlockSize() int64 {
	t.mutexMeta.Lock()
	defer t.mutexMeta.Unlock()
	return t.meta.GetMaxBlockSize()
}

// LoadMaxBlockSize load maxBlockSize into memory
func (t *Meta) LoadMaxBlockSize() (int64, error) {
	maxBlockSizeBuf, findErr := t.metaTable.Get([]byte(ledger.MaxBlockSizeKey))
	if findErr == nil {
		utxoMeta := &pb.UtxoMeta{}
		err := proto.Unmarshal(maxBlockSizeBuf, utxoMeta)
		return utxoMeta.GetMaxBlockSize(), err
	} else if common.NormalizedKVError(findErr) == common.ErrKVNotFound {
		genesisMaxBlockSize := ledger.ledger.GetMaxBlockSize()
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
	err := batch.Put([]byte(xldgpb.MetaTablePrefix+ledger.MaxBlockSizeKey), maxBlockSizeBuf)
	if err == nil {
		t.log.Info("Update maxBlockSize succeed")
	}
	t.mutexMeta.Lock()
	defer t.mutexMeta.Unlock()
	t.metaTmp.MaxBlockSize = maxBlockSize
	return err
}

func (t *Meta) GetReservedContracts() []*pb.InvokeRequest {
	t.mutexMeta.Lock()
	defer t.mutexMeta.Unlock()
	return t.meta.ReservedContracts
}

func (t *Meta) LoadReservedContracts() ([]*pb.InvokeRequest, error) {
	reservedContractsBuf, findErr := t.metaTable.Get([]byte(ledger.ReservedContractsKey))
	if findErr == nil {
		utxoMeta := &pb.UtxoMeta{}
		err := proto.Unmarshal(reservedContractsBuf, utxoMeta)
		return utxoMeta.GetReservedContracts(), err
	} else if common.NormalizedKVError(findErr) == common.ErrKVNotFound {
		return ledger.ledger.GetReservedContracts()
	}
	return nil, findErr
}

//when to register to kernel method
func (t *Meta) UpdateReservedContracts(params []*pb.InvokeRequest, batch kvdb.Batch) error {
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
	err := batch.Put([]byte(xldgpb.MetaTablePrefix+ledger.ReservedContractsKey), paramsBuf)
	if err == nil {
		t.log.Info("Update reservered contract succeed")
	}
	t.mutexMeta.Lock()
	defer t.mutexMeta.Unlock()
	t.metaTmp.ReservedContracts = params
	return err
}

func (t *Meta) GetForbiddenContract() *pb.InvokeRequest {
	t.mutexMeta.Lock()
	defer t.mutexMeta.Unlock()
	return t.meta.GetForbiddenContract()
}

func (t *Meta) GetGroupChainContract() *pb.InvokeRequest {
	t.mutexMeta.Lock()
	defer t.mutexMeta.Unlock()
	return t.meta.GetGroupChainContract()
}

func (t *Meta) LoadGroupChainContract() (*pb.InvokeRequest, error) {
	groupChainContractBuf, findErr := t.metaTable.Get([]byte(ledger.GroupChainContractKey))
	if findErr == nil {
		utxoMeta := &pb.UtxoMeta{}
		err := proto.Unmarshal(groupChainContractBuf, utxoMeta)
		return utxoMeta.GetGroupChainContract(), err
	} else if common.NormalizedKVError(findErr) == common.ErrKVNotFound {
		requests, err := ledger.ledger.GetGroupChainContract()
		if len(requests) > 0 {
			return requests[0], err
		}
		return nil, errors.New("unexpected error")
	}
	return nil, findErr
}

func (t *Meta) LoadForbiddenContract() (*pb.InvokeRequest, error) {
	forbiddenContractBuf, findErr := t.metaTable.Get([]byte(ledger.ForbiddenContractKey))
	if findErr == nil {
		utxoMeta := &pb.UtxoMeta{}
		err := proto.Unmarshal(forbiddenContractBuf, utxoMeta)
		return utxoMeta.GetForbiddenContract(), err
	} else if common.NormalizedKVError(findErr) == common.ErrKVNotFound {
		requests, err := t.ledger.GetForbiddenContract()
		if len(requests) > 0 {
			return requests[0], err
		}
		return nil, errors.New("unexpected error")
	}
	return nil, findErr
}

func (t *Meta) UpdateForbiddenContract(param *pb.InvokeRequest, batch kvdb.Batch) error {
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
	err := batch.Put([]byte(xldgpb.MetaTablePrefix+ledger.ForbiddenContractKey), paramBuf)
	if err == nil {
		t.log.Info("Update forbidden contract succeed")
	}
	t.mutexMeta.Lock()
	defer t.mutexMeta.Unlock()
	t.metaTmp.ForbiddenContract = param
	return err
}

func (t *Meta) LoadIrreversibleBlockHeight() (int64, error) {
	irreversibleBlockHeightBuf, findErr := t.metaTable.Get([]byte(ledger.IrreversibleBlockHeightKey))
	if findErr == nil {
		utxoMeta := &pb.UtxoMeta{}
		err := proto.Unmarshal(irreversibleBlockHeightBuf, utxoMeta)
		return utxoMeta.GetIrreversibleBlockHeight(), err
	} else if common.NormalizedKVError(findErr) == common.ErrKVNotFound {
		return int64(0), nil
	}
	return int64(0), findErr
}

func (t *Meta) LoadIrreversibleSlideWindow() (int64, error) {
	irreversibleSlideWindowBuf, findErr := t.metaTable.Get([]byte(ledger.IrreversibleSlideWindowKey))
	if findErr == nil {
		utxoMeta := &pb.UtxoMeta{}
		err := proto.Unmarshal(irreversibleSlideWindowBuf, utxoMeta)
		return utxoMeta.GetIrreversibleSlideWindow(), err
	} else if common.NormalizedKVError(findErr) == common.ErrKVNotFound {
		genesisSlideWindow := ledger.ledger.GetIrreversibleSlideWindow()
		// negative number is not meaningful
		if genesisSlideWindow < 0 {
			return genesisSlideWindow, ErrProposalParamsIsNegativeNumber
		}
		return genesisSlideWindow, nil
	}
	return int64(0), findErr
}

func (t *Meta) GetIrreversibleBlockHeight() int64 {
	t.mutexMeta.Lock()
	defer t.mutexMeta.Unlock()
	return t.meta.IrreversibleBlockHeight
}

func (t *Meta) GetIrreversibleSlideWindow() int64 {
	t.mutexMeta.Lock()
	defer t.mutexMeta.Unlock()
	return t.meta.IrreversibleSlideWindow
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
	err := batch.Put([]byte(xldgpb.MetaTablePrefix+ledger.IrreversibleBlockHeightKey), irreversibleBlockHeightBuf)
	if err != nil {
		return err
	}
	t.log.Info("Update irreversibleBlockHeight succeed")
	t.mutexMeta.Lock()
	defer t.mutexMeta.Unlock()
	t.metaTmp.IrreversibleBlockHeight = nextIrreversibleBlockHeight
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

func (t *Meta) updateNextIrreversibleBlockHeightForPrune(blockHeight int64, curIrreversibleBlockHeight int64, curIrreversibleSlideWindow int64, batch kvdb.Batch) error {
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
	err := batch.Put([]byte(xldgpb.MetaTablePrefix+ledger.IrreversibleSlideWindowKey), irreversibleSlideWindowBuf)
	if err != nil {
		return err
	}
	t.log.Info("Update irreversibleSlideWindow succeed")
	t.mutexMeta.Lock()
	defer t.mutexMeta.Unlock()
	t.metaTmp.IrreversibleSlideWindow = nextIrreversibleSlideWindow
	return nil
}

// GetGasPrice get gas rate to utxo
func (t *Meta) GetGasPrice() *pb.GasPrice {
	t.mutexMeta.Lock()
	defer t.mutexMeta.Unlock()
	return t.meta.GetGasPrice()
}

// LoadGasPrice load gas rate
func (t *Meta) LoadGasPrice() (*pb.GasPrice, error) {
	gasPriceBuf, findErr := t.metaTable.Get([]byte(ledger.GasPriceKey))
	if findErr == nil {
		utxoMeta := &pb.UtxoMeta{}
		err := proto.Unmarshal(gasPriceBuf, utxoMeta)
		return utxoMeta.GetGasPrice(), err
	} else if common.NormalizedKVError(findErr) == common.ErrKVNotFound {
		nofee := ledger.ledger.GetNoFee()
		if nofee {
			gasPrice := &pb.GasPrice{
				CpuRate:  0,
				MemRate:  0,
				DiskRate: 0,
				XfeeRate: 0,
			}
			return gasPrice, nil

		} else {
			gasPrice := ledger.ledger.GetGasPrice()
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
				gasPrice = &pb.GasPrice{
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
func (t *Meta) UpdateGasPrice(nextGasPrice *pb.GasPrice, batch kvdb.Batch) error {
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
	err := batch.Put([]byte(xldgpb.MetaTablePrefix+ledger.GasPriceKey), gasPriceBuf)
	if err != nil {
		return err
	}
	t.log.Info("Update gas price succeed")
	t.mutexMeta.Lock()
	defer t.mutexMeta.Unlock()
	t.metaTmp.GasPrice = nextGasPrice
	return nil
}

func (t *Meta) GetReservedContractRequests(req []*pb.InvokeRequest, isPreExec bool) ([]*pb.InvokeRequest, error) {
	MetaReservedContracts := t.GetReservedContracts()
	if MetaReservedContracts == nil {
		return nil, nil
	}
	reservedContractstpl := MetaReservedContracts
	t.log.Info("MetaReservedContracts", "reservedContracts", reservedContractstpl)

	// if all reservedContracts have not been updated, return nil, nil
	ra := &reservedArgs{}
	if isPreExec || len(reservedContractstpl) == 0 {
		ra = genArgs(req)
	} else {
		// req should contrain reservedContracts, so the len of req should no less than reservedContracts
		if len(req) < len(reservedContractstpl) {
			t.log.Warn("req should contain reservedContracts")
			return nil, ErrGetReservedContracts
		} else if len(req) > len(reservedContractstpl) {
			ra = genArgs(req[len(reservedContractstpl):])
		}
	}

	reservedContracts := []*pb.InvokeRequest{}
	for _, rc := range reservedContractstpl {
		rctmp := *rc
		rctmp.Args = make(map[string][]byte)
		for k, v := range rc.GetArgs() {
			buf := new(bytes.Buffer)
			tpl := template.Must(template.New("value").Parse(string(v)))
			tpl.Execute(buf, ra)
			rctmp.Args[k] = buf.Bytes()
		}
		reservedContracts = append(reservedContracts, &rctmp)
	}
	return reservedContracts, nil
}

package pow

import (
	"encoding/json"
	"math/big"
	"strconv"
	"testing"

	bmock "github.com/xuperchain/xupercore/bcs/consensus/mock"
	cctx "github.com/xuperchain/xupercore/kernel/consensus/context"
	"github.com/xuperchain/xupercore/kernel/consensus/def"
	kmock "github.com/xuperchain/xupercore/kernel/consensus/mock"
)

var (
	target    uint32 = 0x1903a30c
	minTarget uint32 = 0x207FFFFF
	//Bitcoin值 mTarget   uint32 = 0x1d00FFFF
	mTarget uint32 = 0x207FFFFF
)

func getPoWConsensusConf() []byte {
	c := PoWConfig{
		DefaultTarget:        target,
		AdjustHeightGap:      1,
		ExpectedPeriodMilSec: 3000,
		MaxTarget:            mTarget,
	}
	j, _ := json.Marshal(c)
	return j
}

func prepare() (*cctx.ConsensusCtx, error) {
	l := kmock.NewFakeLedger(getPoWConsensusConf())
	cCtx, err := bmock.NewConsensusCtx(l)
	cCtx.Ledger = l
	return cCtx, err
}

func getConsensusConf() def.ConsensusConfig {
	return def.ConsensusConfig{
		ConsensusName: "pow",
		Config:        string(getPoWConsensusConf()),
		StartHeight:   1,
		Index:         0,
	}
}

func getWrongConsensusConf() def.ConsensusConfig {
	return def.ConsensusConfig{
		ConsensusName: "pow2",
		Config:        string(getPoWConsensusConf()),
		StartHeight:   1,
		Index:         0,
	}
}

func TestNewPoWConsensus(t *testing.T) {
	cCtx, err := prepare()
	if err != nil {
		t.Error("prepare error")
		return
	}
	conf := getConsensusConf()
	i := NewPoWConsensus(*cCtx, conf)
	if i == nil {
		t.Error("NewPoWConsensus error")
		return
	}
	if i := NewPoWConsensus(*cCtx, getWrongConsensusConf()); i != nil {
		t.Error("NewPoWConsensus check name error")
	}
}

func TestGetConsensusStatus(t *testing.T) {
	cCtx, err := prepare()
	if err != nil {
		t.Error("prepare error")
		return
	}
	conf := getConsensusConf()
	i := NewPoWConsensus(*cCtx, conf)
	status, _ := i.GetConsensusStatus()
	if status.GetVersion() != 0 {
		t.Error("GetVersion error")
		return
	}
	if status.GetStepConsensusIndex() != 0 {
		t.Error("GetStepConsensusIndex error")
		return
	}
	if status.GetConsensusBeginInfo() != 1 {
		t.Error("GetConsensusBeginInfo error")
		return
	}
	if status.GetConsensusName() != "pow" {
		t.Error("GetConsensusName error")
		return
	}
	vb := status.GetCurrentValidatorsInfo()
	m := MinerInfo{}
	err = json.Unmarshal(vb, &m)
	if err != nil {
		t.Error("GetCurrentValidatorsInfo unmarshal error", "error", err)
		return
	}
	if m.Address != bmock.Miner {
		t.Error("GetCurrentValidatorsInfo error", "address", m.Address)
	}
}

func TestParseConsensusStorage(t *testing.T) {
	ps := PoWStorage{
		TargetBits: target,
	}
	b, err := json.Marshal(ps)
	if err != nil {
		t.Error("ParseConsensusStorage Unmarshal error", "error", err)
		return
	}
	cCtx, err := prepare()
	if err != nil {
		t.Error("prepare error", err)
		return
	}
	b1, err := bmock.NewBlockWithStorage(1, cCtx.Crypto, cCtx.Address, b)
	if err != nil {
		t.Error("NewBlockWithStorage error", err)
		return
	}
	conf := getConsensusConf()
	pow := NewPoWConsensus(*cCtx, conf)

	i, err := pow.ParseConsensusStorage(b1)
	if err != nil {
		t.Error("ParseConsensusStorage error", "error", err)
		return
	}
	s, ok := i.(PoWStorage)
	if !ok {
		t.Error("ParseConsensusStorage transfer error")
		return
	}
	if s.TargetBits != target {
		t.Error("ParseConsensusStorage transfer error", "target", target)
	}
}

func TestSetCompact(t *testing.T) {
	bigint, pfNegative, pfOverflow := SetCompact(target)
	if pfNegative || pfOverflow {
		t.Error("TestSetCompact overflow or negative")
		return
	}
	var strings []string
	for _, word := range bigint.Bits() {
		s := strconv.FormatUint(uint64(word), 16)
		strings = append(strings, s)
	}
	if bigint.BitLen() > 256 {
		t.Error("TestSetCompact overflow", "bigint.BitLen()", bigint.BitLen(), "string", strings)
		return
	}
	// t := 0x0000000000000003A30C00000000000000000000000000000000000000000000, 对应target为0x1903a30c
	b := big.NewInt(0x0000000000000003A30C00000000)
	b.Lsh(b, 144)
	if b.Cmp(bigint) != 0 {
		t.Error("TestSetCompact equal err", "bigint", bigint, "b", b)
	}
}

func TestGetCompact(t *testing.T) {
	b := big.NewInt(0x0000000000000003A30C00000000)
	b.Lsh(b, 144)
	target, _ := GetCompact(b)
	if target != 0x1903a30c {
		t.Error("TestGetCompact error", "target", target)
		return
	}
}

func TestIsProofed(t *testing.T) {
	cCtx, err := prepare()
	if err != nil {
		t.Error("prepare error", err)
		return
	}
	conf := getConsensusConf()
	i := NewPoWConsensus(*cCtx, conf)
	pow, ok := i.(*PoWConsensus)
	if !ok {
		t.Error("TestIsProofed transfer error")
		return
	}
	// t := 0x0000000000000003A30C00000000000000000000000000000000000000000000, 对应target为0x1903a30c
	b := big.NewInt(0x0000000000000003A30C00000000)
	b.Lsh(b, 144)
	blockid := b.Bytes()
	if !pow.IsProofed(blockid, pow.config.MaxTarget) {
		t.Error("TestIsProofed error")
	}
}

func TestMining(t *testing.T) {
	cCtx, err := prepare()
	if err != nil {
		t.Error("prepare error", err)
		return
	}
	conf := getConsensusConf()
	i := NewPoWConsensus(*cCtx, conf)
	powC, ok := i.(*PoWConsensus)
	if !ok {
		t.Error("TestMining transfer error")
		return
	}
	powC.targetBits = minTarget
	ps := PoWStorage{
		TargetBits: minTarget,
	}
	by, _ := json.Marshal(ps)
	B, err := bmock.NewBlockWithStorage(1, cCtx.Crypto, cCtx.Address, by)
	if err != nil {
		t.Error("NewBlockWithStorage error", err)
		return
	}
	err = powC.mining(B)
	if err != nil {
		t.Error("TestMining mining error", "blockId", B.GetBlockid(), "err", err)
	}
}

func TestRefreshDifficulty(t *testing.T) {
	cCtx, err := prepare()
	if err != nil {
		t.Error("prepare error", err)
		return
	}
	conf := getConsensusConf()
	i := NewPoWConsensus(*cCtx, conf)
	powC, ok := i.(*PoWConsensus)
	if !ok {
		t.Error("TestRefreshDifficulty transfer error")
		return
	}
	genesisB, err := bmock.NewBlock(0, cCtx.Crypto, cCtx.Address)
	if err != nil {
		t.Error("NewBlock error", err)
		return
	}
	l, ok := powC.ctx.Ledger.(*kmock.FakeLedger)
	err = l.Put(genesisB)
	if err != nil {
		t.Error("TestRefreshDifficulty put genesis err", "err", err)
		return
	}

	powC.targetBits = minTarget
	ps := PoWStorage{
		TargetBits: minTarget,
	}
	by, _ := json.Marshal(ps)
	B1, err := bmock.NewBlockWithStorage(1, cCtx.Crypto, cCtx.Address, by)
	if err != nil {
		t.Error("NewBlockWithStorage error", err)
		return
	}
	err = powC.mining(B1)
	if err != nil {
		t.Error("TestRefreshDifficulty mining error", "blockId", B1.GetBlockid(), "err", err)
		return
	}
	err = l.Put(B1)
	if err != nil {
		t.Error("TestRefreshDifficulty put B1 err", "err", err)
		return
	}
	B2, err := bmock.NewBlockWithStorage(1, cCtx.Crypto, cCtx.Address, by)
	if err != nil {
		t.Error("NewBlockWithStorage error", err)
		return
	}
	err = powC.mining(B2)
	if err != nil {
		t.Error("TestRefreshDifficulty mining error", "blockId", B2.GetBlockid(), "err", err)
		return
	}
	err = l.Put(B2)
	if err != nil {
		t.Error("TestRefreshDifficulty put B1 err", "err", err)
		return
	}

	target, err := powC.refreshDifficulty(B2.GetBlockid(), 3)
	if err != nil {
		t.Error("TestRefreshDifficulty refreshDifficulty err", "err", err, "target", target)
		return
	}
}

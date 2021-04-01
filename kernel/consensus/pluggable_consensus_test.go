package consensus

import (
	"encoding/json"
	"path/filepath"
	"strconv"
	"testing"
	"time"

	"github.com/xuperchain/xupercore/kernel/common/xcontext"
	"github.com/xuperchain/xupercore/kernel/consensus/base"
	"github.com/xuperchain/xupercore/kernel/consensus/context"
	cctx "github.com/xuperchain/xupercore/kernel/consensus/context"
	"github.com/xuperchain/xupercore/kernel/consensus/def"
	"github.com/xuperchain/xupercore/kernel/consensus/mock"
	"github.com/xuperchain/xupercore/kernel/contract"
	"github.com/xuperchain/xupercore/lib/logs"
	"github.com/xuperchain/xupercore/lib/utils"
)

var (
	_     = Register("fake", NewFakeConsensus)
	_     = Register("another", NewAnotherConsensus)
	Miner = "xuper5"
)

type FakeSMRStruct struct{}

func (smr *FakeSMRStruct) GetCurrentValidatorsInfo() []byte {
	return nil
}

func (smr *FakeSMRStruct) GetCurrentTerm() int64 {
	return int64(0)
}

type stateMachineInterface interface {
	GetCurrentValidatorsInfo() []byte
	GetCurrentTerm() int64
}

type FakeConsensusStatus struct {
	version            int64
	beginHeight        int64
	stepConsensusIndex int
	consensusName      string
	smr                stateMachineInterface
}

func (s *FakeConsensusStatus) GetVersion() int64 {
	return s.version
}

func (s *FakeConsensusStatus) GetConsensusBeginInfo() int64 {
	return s.beginHeight
}

func (s *FakeConsensusStatus) GetStepConsensusIndex() int {
	return s.stepConsensusIndex
}

func (s *FakeConsensusStatus) GetConsensusName() string {
	return s.consensusName
}

func (s *FakeConsensusStatus) GetCurrentValidatorsInfo() []byte {
	return s.smr.GetCurrentValidatorsInfo()
}

func (s *FakeConsensusStatus) GetCurrentTerm() int64 {
	return s.smr.GetCurrentTerm()
}

type FakeConsensusImp struct {
	smr    FakeSMRStruct
	status *FakeConsensusStatus
}

func NewFakeConsensus(cCtx cctx.ConsensusCtx, cCfg def.ConsensusConfig) base.ConsensusImplInterface {
	status := &FakeConsensusStatus{
		beginHeight:   cCfg.StartHeight,
		consensusName: cCfg.ConsensusName,
	}
	return &FakeConsensusImp{
		status: status,
	}
}

func (con *FakeConsensusImp) CompeteMaster(height int64) (bool, bool, error) {
	return true, true, nil
}

func (con *FakeConsensusImp) CheckMinerMatch(ctx xcontext.XContext, block cctx.BlockInterface) (bool, error) {

	return true, nil
}

func (con *FakeConsensusImp) ProcessBeforeMiner(timestamp int64) ([]byte, []byte, error) {
	return nil, nil, nil
}

func (con *FakeConsensusImp) ProcessConfirmBlock(block cctx.BlockInterface) error {
	return nil
}

func (con *FakeConsensusImp) GetConsensusStatus() (base.ConsensusStatus, error) {
	return con.status, nil
}

func (con *FakeConsensusImp) CalculateBlock(block cctx.BlockInterface) error {
	return nil
}

func (con *FakeConsensusImp) ParseConsensusStorage(block context.BlockInterface) (interface{}, error) {
	return nil, nil
}

func (con *FakeConsensusImp) Stop() error {
	return nil
}

func (con *FakeConsensusImp) Start() error {
	return nil
}

type AnotherConsensusImp struct {
	smr    FakeSMRStruct
	status *FakeConsensusStatus
}

func NewAnotherConsensus(cCtx cctx.ConsensusCtx, cCfg def.ConsensusConfig) base.ConsensusImplInterface {
	status := &FakeConsensusStatus{
		beginHeight:   cCfg.StartHeight,
		consensusName: cCfg.ConsensusName,
	}
	return &AnotherConsensusImp{
		status: status,
	}
}

func (con *AnotherConsensusImp) CompeteMaster(height int64) (bool, bool, error) {
	return true, true, nil
}

func (con *AnotherConsensusImp) CheckMinerMatch(ctx xcontext.XContext, block cctx.BlockInterface) (bool, error) {

	return true, nil
}

func (con *AnotherConsensusImp) ProcessBeforeMiner(timestamp int64) ([]byte, []byte, error) {
	return nil, nil, nil
}

func (con *AnotherConsensusImp) CalculateBlock(block cctx.BlockInterface) error {
	return nil
}

func (con *AnotherConsensusImp) ProcessConfirmBlock(block cctx.BlockInterface) error {
	return nil
}

func (con *AnotherConsensusImp) GetConsensusStatus() (base.ConsensusStatus, error) {
	return con.status, nil
}

func (con *AnotherConsensusImp) ParseConsensusStorage(block context.BlockInterface) (interface{}, error) {
	return nil, nil
}

func (con *AnotherConsensusImp) Stop() error {
	return nil
}

func (con *AnotherConsensusImp) Start() error {
	return nil
}

func NewFakeLogger() logs.Logger {
	confFile := utils.GetCurFileDir()
	confFile = filepath.Join(confFile, "config/log.yaml")
	logDir := utils.GetCurFileDir()
	logDir = filepath.Join(logDir, "logs")

	logs.InitLog(confFile, logDir)
	log, _ := logs.NewLogger("", "consensus_test")
	return log
}

func GetConsensusCtx(ledger *mock.FakeLedger) cctx.ConsensusCtx {
	ctx := cctx.ConsensusCtx{
		BcName: "xuper",
		Ledger: ledger,
		BaseCtx: xcontext.BaseCtx{
			XLog: NewFakeLogger(),
		},
		Contract: &mock.FakeManager{
			R: &mock.FakeRegistry{
				M: make(map[string]contract.KernMethod),
			},
		},
	}
	return ctx
}

func TestNewPluggableConsensus(t *testing.T) {
	// Fake name is 'fake'in consensusConf.
	l := mock.NewFakeLedger(mock.GetGenesisConsensusConf())
	ctx := GetConsensusCtx(l)
	pc, err := NewPluggableConsensus(ctx)
	if err != nil {
		t.Error("NewPluggableConsensus error", err)
		return
	}
	status, err := pc.GetConsensusStatus()
	if err != nil {
		t.Error("GetConsensusStatus error", err)
		return
	}
	if status.GetConsensusName() != "fake" {
		t.Error("GetConsensusName error", err)
		return
	}
	wl := mock.NewFakeLedger(GetWrongConsensusConf())
	wctx := GetConsensusCtx(wl)
	_, err = NewPluggableConsensus(wctx)
	if err == nil {
		t.Error("Empty name error")
	}
}

func GetNewConsensusConf() []byte {
	return []byte("{\"name\":\"another\",\"config\":\"\"}")
}

func GetWrongConsensusConf() []byte {
	return []byte("{\"name\":\"\",\"config\":\"\"}")
}

func NewUpdateArgs() map[string][]byte {
	a := make(map[string]interface{})
	a["name"] = "another"
	a["config"] = map[string]interface{}{
		"miner":  "TeyyPLpp9L7QAcxHangtcHTu7HUZ6iydY",
		"period": "3000",
	}
	ab, _ := json.Marshal(&a)
	r := map[string][]byte{
		"args":   ab,
		"height": []byte(strconv.FormatInt(8, 10)),
	}
	return r
}

func NewUpdateM() map[string]map[string][]byte {
	a := make(map[string]map[string][]byte)
	return a
}

func TestUpdateConsensus(t *testing.T) {
	l := mock.NewFakeLedger(mock.GetGenesisConsensusConf())
	ctx := GetConsensusCtx(l)
	pc, _ := NewPluggableConsensus(ctx)
	newHeight := l.GetTipBlock().GetHeight() + 1
	_, _, err := pc.CompeteMaster(newHeight)
	if err != nil {
		t.Error("CompeteMaster error! height = ", newHeight)
		return
	}
	np, ok := pc.(*PluggableConsensus)
	if !ok {
		t.Error("Transfer PluggableConsensus error!")
		return
	}
	fakeCtx := mock.NewFakeKContext(NewUpdateArgs(), NewUpdateM())
	np.updateConsensus(fakeCtx)
	if len(np.stepConsensus.cons) != 2 {
		t.Error("Update consensus error!")
		return
	}
	status, err := np.GetConsensusStatus()
	if err != nil {
		t.Error("GetConsensusStatus error", err)
		return
	}
	if status.GetConsensusName() != "another" {
		t.Error("GetConsensusName error", err)
		return
	}
	by, err := fakeCtx.Get(contractBucket, []byte(consensusKey))
	if err != nil {
		t.Error("fakeCtx error", err)
		return
	}
	c := map[int]def.ConsensusConfig{}
	err = json.Unmarshal(by, &c)
	if err != nil {
		t.Error("unmarshal error", err)
		return
	}
	if len(c) != 2 {
		t.Error("update error", "len", len(c))
	}
}

func TestCompeteMaster(t *testing.T) {
	// ledger的高度为2
	l := mock.NewFakeLedger(mock.GetGenesisConsensusConf())
	ctx := GetConsensusCtx(l)
	pc, _ := NewPluggableConsensus(ctx)
	newHeight := l.GetTipBlock().GetHeight() + 1
	if newHeight != 3 {
		t.Error("Ledger Meta error, height=", newHeight)
		return
	}
	_, _, err := pc.CompeteMaster(newHeight)
	if err != nil {
		t.Error("CompeteMaster error")
	}
}

func TestCheckMinerMatch(t *testing.T) {
	l := mock.NewFakeLedger(mock.GetGenesisConsensusConf())
	ctx := GetConsensusCtx(l)
	pc, _ := NewPluggableConsensus(ctx)
	newHeight := l.GetTipBlock().GetHeight() + 1
	_, err := pc.CheckMinerMatch(mock.NewXContent(), mock.NewBlock(int(newHeight)))
	if err != nil {
		t.Error("CheckMinerMatch error")
	}
}

func TestCalculateBlock(t *testing.T) {
	l := mock.NewFakeLedger(mock.GetGenesisConsensusConf())
	ctx := GetConsensusCtx(l)
	pc, _ := NewPluggableConsensus(ctx)
	newHeight := l.GetTipBlock().GetHeight() + 1
	err := pc.CalculateBlock(mock.NewBlock(int(newHeight)))
	if err != nil {
		t.Error("CalculateBlock error")
	}
}

func TestProcessBeforeMiner(t *testing.T) {
	l := mock.NewFakeLedger(mock.GetGenesisConsensusConf())
	ctx := GetConsensusCtx(l)
	pc, _ := NewPluggableConsensus(ctx)
	_, _, err := pc.ProcessBeforeMiner(time.Now().UnixNano())
	if err != nil {
		t.Error("ProcessBeforeMiner error")
	}
}

func TestProcessConfirmBlock(t *testing.T) {
	l := mock.NewFakeLedger(mock.GetGenesisConsensusConf())
	ctx := GetConsensusCtx(l)
	pc, _ := NewPluggableConsensus(ctx)
	err := pc.ProcessConfirmBlock(mock.NewBlock(3))
	if err != nil {
		t.Error("ProcessConfirmBlock error")
	}
}

func TestGetConsensusStatus(t *testing.T) {
	l := mock.NewFakeLedger(mock.GetGenesisConsensusConf())
	ctx := GetConsensusCtx(l)
	pc, _ := NewPluggableConsensus(ctx)
	_, err := pc.GetConsensusStatus()
	if err != nil {
		t.Error("GetConsensusStatus error")
	}
}

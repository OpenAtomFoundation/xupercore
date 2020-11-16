package consensus

import (
	"path/filepath"

	"github.com/xuperchain/xupercore/kernel/common/xcontext"
	"github.com/xuperchain/xupercore/kernel/consensus/base"
	"github.com/xuperchain/xupercore/kernel/consensus/context"
	cctx "github.com/xuperchain/xupercore/kernel/consensus/context"
	"github.com/xuperchain/xupercore/kernel/consensus/mock"
	"github.com/xuperchain/xupercore/lib/logs"
	"github.com/xuperchain/xupercore/lib/utils"
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

func init() {
	Register("fake", NewFakeConsensus)
	Register("Another", NewAnotherConsensus)
}

func NewFakeConsensus(cCtx cctx.ConsensusCtx, cCfg cctx.ConsensusConfig) base.ConsensusImplInterface {
	status := &FakeConsensusStatus{
		beginHeight:   cCfg.BeginHeight,
		consensusName: cCfg.ConsensusName,
	}
	return &FakeConsensusImp{
		status: status,
	}
}

func (con *FakeConsensusImp) CompeteMaster(height int64) (bool, bool, error) {
	return true, true, nil
}

func (con *FakeConsensusImp) CheckMinerMatch(ctx xcontext.BaseCtx, block cctx.BlockInterface) (bool, error) {

	return true, nil
}

func (con *FakeConsensusImp) ProcessBeforeMiner(timestamp int64) (bool, []byte, error) {
	return true, nil, nil
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

func NewAnotherConsensus(cCtx cctx.ConsensusCtx, cCfg cctx.ConsensusConfig) base.ConsensusImplInterface {
	status := &FakeConsensusStatus{
		beginHeight:   cCfg.BeginHeight,
		consensusName: cCfg.ConsensusName,
	}
	return &AnotherConsensusImp{
		status: status,
	}
}

func (con *AnotherConsensusImp) CompeteMaster(height int64) (bool, bool, error) {
	return true, true, nil
}

func (con *AnotherConsensusImp) CheckMinerMatch(ctx xcontext.BaseCtx, block cctx.BlockInterface) (bool, error) {

	return true, nil
}

func (con *AnotherConsensusImp) ProcessBeforeMiner(timestamp int64) (bool, []byte, error) {
	return true, nil, nil
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

type FakeCryptoClient struct{}

func (cc *FakeCryptoClient) GetEcdsaPublicKeyFromJSON([]byte) ([]byte, error) {
	return nil, nil
}
func (cc *FakeCryptoClient) VerifyAddressUsingPublicKey(string, []byte) (bool, uint8) {
	return true, 0
}
func (cc *FakeCryptoClient) VerifyECDSA([]byte, []byte, []byte) (bool, error) {
	return true, nil
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
		BcName:       "xuper",
		Ledger:       ledger,
		P2p:          &mock.FakeP2p{},
		CryptoClient: &FakeCryptoClient{},
		BCtx: xcontext.BaseCtx{
			XLog: NewFakeLogger(),
		},
	}
	return ctx
}

/*
func TestNewPluggableConsensus(t *testing.T) {
	l := mock.NewFakeLedger()
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
}

func GetNewConsensusConf() []byte {
	return []byte("{\"name\":\"Another\",\"config\":\"\"}")
}

func TestUpdateConsensus(t *testing.T) {
	l := mock.NewFakeLedger()
	ctx := GetConsensusCtx(l)
	pc, _ := NewPluggableConsensus(ctx)
	newHeight := l.GetMeta().GetTrunkHeight() + 1
	_, _, err := pc.CompeteMaster(newHeight)
	if err != nil {
		t.Error("CompeteMaster error! height = ", newHeight)
		return
	}
	// 此时pc.nextHeight == 3
	kctx := mock.NewFakeKContextImpl(GetNewConsensusConf())
	np, ok := pc.(*PluggableConsensus)
	if !ok {
		t.Error("Transfer PluggableConsensus error!")
		return
	}
	f := mock.ContractRegister(np.updateConsensus)
	f(kctx, int64(3))
	if len(np.stepConsensus.cons) != 2 {
		t.Error("Update consensus error!")
		return
	}
	status, err := np.GetConsensusStatus()
	if err != nil {
		t.Error("GetConsensusStatus error", err)
		return
	}
	if status.GetConsensusName() != "Another" {
		t.Error("GetConsensusName error", err)
		return
	}
}

func TestCompeteMaster(t *testing.T) {
	// ledger的高度为2
	l := mock.NewFakeLedger()
	ctx := GetConsensusCtx(l)
	pc, _ := NewPluggableConsensus(ctx)
	newHeight := l.GetMeta().GetTrunkHeight() + 1
	if newHeight != 3 {
		t.Error("Ledger Meta error, height=", newHeight)
		return
	}
	_, _, err := pc.CompeteMaster(newHeight)
	if err != nil {
		t.Error("CompeteMaster error")
	}
}

func TestProcessBeforeMiner(t *testing.T) {
	l := mock.NewFakeLedger()
	ctx := GetConsensusCtx(l)
	pc, _ := NewPluggableConsensus(ctx)
	_, _, err := pc.ProcessBeforeMiner(time.Now().UnixNano())
	if err != nil {
		t.Error("ProcessBeforeMiner error")
	}
}

func TestProcessConfirmBlock(t *testing.T) {
	l := mock.NewFakeLedger()
	ctx := GetConsensusCtx(l)
	pc, _ := NewPluggableConsensus(ctx)
	err := pc.ProcessConfirmBlock(mock.NewBlock(3))
	if err != nil {
		t.Error("ProcessConfirmBlock error")
	}
}

func TestGetConsensusStatus(t *testing.T) {
	l := mock.NewFakeLedger()
	ctx := GetConsensusCtx(l)
	pc, _ := NewPluggableConsensus(ctx)
	_, err := pc.GetConsensusStatus()
	if err != nil {
		t.Error("GetConsensusStatus error")
	}
}
*/

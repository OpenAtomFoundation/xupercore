package parachain

import (
	"encoding/json"
	"errors"
	"fmt"
	//"github.com/xuperchain/xupercore/kernel/engines/xuperos"
	"github.com/xuperchain/xupercore/kernel/engines/xuperos/common"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/xuperchain/xupercore/bcs/ledger/xledger/ledger"
	"github.com/xuperchain/xupercore/bcs/ledger/xledger/state"
	"github.com/xuperchain/xupercore/bcs/ledger/xledger/state/context"
	"github.com/xuperchain/xupercore/bcs/ledger/xledger/tx"
	"github.com/xuperchain/xupercore/bcs/ledger/xledger/utils"
	"github.com/xuperchain/xupercore/bcs/ledger/xledger/xldgpb"
	"github.com/xuperchain/xupercore/kernel/contract"
	"github.com/xuperchain/xupercore/lib/crypto/client"
	lutils "github.com/xuperchain/xupercore/lib/utils"
)

var (
	// ErrBlockChainExist is returned when create an existed block chain
	ErrBlockChainExist = errors.New("BlockChain Exist")
	// ErrCreateBlockChain is returned when create block chain error
	ErrCreateBlockChain = errors.New("Create BlockChain error")
)

type KernMethod struct {
	BcName            string
	MinNewChainAmount int64
	ChainCtx          *common.ChainCtx
}

func NewKernContractMethod(bcName string, minNewChainAmount int64, chainCtx *common.ChainCtx) *KernMethod {
	t := &KernMethod{
		BcName:            bcName,
		MinNewChainAmount: minNewChainAmount,
		ChainCtx:          chainCtx,
	}

	return t
}

type createChainMessage struct {
	BcName   string
	Data     string
	ChainCtx *common.ChainCtx
}

/*func handleCreateChain(ctx asyncTask.TaskContext) error {
	var args createChainMessage
	ctx.ParseArgs(&args)

	err := createLedger(args.ChainCtx, args.BcName, []byte(args.Data))
	if err != nil {
		return err
	}
	chain, err := xuperos.LoadChain(args.ChainCtx.EngCtx, args.BcName)
	if err != nil {
		return err
	}
	go chain.Start()

	return nil
}*/

func (t *KernMethod) CreateBlockChain(ctx contract.KContext) (*contract.Response, error) {
	if t.BcName != t.ChainCtx.EngCtx.EngCfg.RootChain {
		return nil, errors.New("Permission denied to call this contract")
	}
	/*bcName, bcData, err := t.validateCreateBC(ctx.Args())
	if err != nil {
		return nil, err
	}
	message := &createChainMessage{
		ChainCtx: t.ChainCtx,
		BcName:   bcName,
		Data:     bcData,
	}
	ctx.EmitAsyncTask("CreateBlockChain", message)*/

	delta := contract.Limits{
		XFee: t.MinNewChainAmount,
	}
	ctx.AddResourceUsed(delta)

	return &contract.Response{
		Status: 200,
		Body:   []byte("CreateBlockChain success"),
	}, nil
}

func (t *KernMethod) validateCreateBC(args map[string][]byte) (string, string, error) {
	bcName := ""
	bcData := ""
	if args["name"] == nil {
		return bcName, bcData, errors.New("block chain name is empty")
	}
	if args["data"] == nil {
		return bcName, bcData, errors.New("first block data is empty")
	}
	bcName = string(args["name"])
	bcData = string(args["data"])

	// check data format, prevent panic
	bcCfg := &ledger.RootConfig{}
	err := json.Unmarshal(args["data"], bcCfg)
	if err != nil {
		return bcName, bcData, fmt.Errorf("first block data error.err:%v", err)
	}

	return bcName, bcData, nil
}

func createLedger(chainCtx *common.ChainCtx, bcName string, data []byte) error {
	envConf := chainCtx.EngCtx.EnvCfg
	dataDir := envConf.GenDataAbsPath(envConf.ChainDir)
	fullpath := filepath.Join(dataDir, bcName)
	if lutils.PathExists(fullpath) {
		return ErrBlockChainExist
	}
	err := os.MkdirAll(fullpath, 0755)
	if err != nil {
		return err
	}
	rootfile := filepath.Join(fullpath, fmt.Sprintf("%s.json", bcName))
	err = ioutil.WriteFile(rootfile, data, 0666)
	if err != nil {
		os.RemoveAll(fullpath)
		return err
	}
	lctx, err := ledger.NewLedgerCtx(envConf, bcName)
	if err != nil {
		return err
	}
	xledger, err := ledger.CreateLedger(lctx, data)
	if err != nil {
		os.RemoveAll(fullpath)
		return err
	}
	tx, err := tx.GenerateRootTx(data)
	if err != nil {
		os.RemoveAll(fullpath)
		return err
	}
	txlist := []*xldgpb.Transaction{tx}
	b, err := xledger.FormatRootBlock(txlist)
	if err != nil {
		os.RemoveAll(fullpath)
		return ErrCreateBlockChain
	}
	xledger.ConfirmBlock(b, true)
	cryptoType, err := utils.GetCryptoType(data)
	if err != nil {
		os.RemoveAll(fullpath)
		return ErrCreateBlockChain
	}
	crypt, err := client.CreateCryptoClient(cryptoType)
	if err != nil {
		os.RemoveAll(fullpath)
		return ErrCreateBlockChain
	}
	sctx, err := context.NewStateCtx(envConf, bcName, xledger, crypt)
	if err != nil {
		os.RemoveAll(fullpath)
		return err
	}
	handleState, err := state.NewState(sctx)
	if err != nil {
		os.RemoveAll(fullpath)
		return err
	}

	defer xledger.Close()
	defer handleState.Close()
	err = handleState.Play(b.Blockid)
	if err != nil {
		return err
	}
	return nil
}

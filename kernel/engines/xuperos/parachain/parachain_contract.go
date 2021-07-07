package parachain

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/xuperchain/xupercore/kernel/engines/xuperos/common"
	"github.com/xuperchain/xupercore/protos"

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
	ErrGroupNotFound    = errors.New("Group Not Found")
	ErrUnAuthorized     = errors.New("UnAuthorized")
)

const (
	success           = 200
	unAuthorized      = 403
	targetNotFound    = 404
	internalServerErr = 500

	paraChainEventName = "EditParaGroups"
)

type paraChainContract struct {
	BcName            string
	MinNewChainAmount int64
	ChainCtx          *common.ChainCtx
}

func NewParaChainContract(bcName string, minNewChainAmount int64, chainCtx *common.ChainCtx) *paraChainContract {
	t := &paraChainContract{
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

/*
func handleCreateChain(ctx asyncTask.TaskContext) error {
	var args createChainMessage
	ctx.ParseArgs(&args)
	err := createLedger(args.ChainCtx, args.BcName, []byte(args.Data))
	if err != nil {
		return err
	}
	return args.ChainCtx.EngCtx.ChainM.LoadChain(args.BcName)
}
*/

func (p *paraChainContract) createChain(ctx contract.KContext) (*contract.Response, error) {
	if p.BcName != p.ChainCtx.EngCtx.EngCfg.RootChain {
		return nil, errors.New("Permission denied to call this contract")
	}
	bcName, _, err := p.parseArgs(ctx.Args())
	/*
		if err != nil {
			return nil, err
		}
		message := &createChainMessage{
			ChainCtx: t.ChainCtx,
			BcName:   bcName,
			Data:     bcData,
		}
		ctx.EmitAsyncTask("CreateBlockChain", message)
	*/

	// 确保未创建过该链
	chainRes, err := ctx.Get(ParaChainKernelContract, []byte(bcName))
	if chainRes != nil {
		return newContractErrResponse(unAuthorized, ErrBlockChainExist.Error()), ErrBlockChainExist
	}

	// 创建链时，自动写入Group信息
	value := &Group{
		GroupID:    bcName,
		Admin:      []string{ctx.Initiator()},
		Identities: nil,
	}
	value = loadGroupArgs(ctx.Args(), value)
	rawBytes, err := json.Marshal(value)
	if err != nil {
		return newContractErrResponse(internalServerErr, err.Error()), err
	}
	if err := ctx.Put(ParaChainKernelContract,
		[]byte(bcName), rawBytes); err != nil {
		return newContractErrResponse(internalServerErr, err.Error()), err
	}
	delta := contract.Limits{
		XFee: p.MinNewChainAmount,
	}
	ctx.AddResourceUsed(delta)

	return &contract.Response{
		Status: success,
		Body:   []byte("CreateBlockChain success"),
	}, nil
}

func (p *paraChainContract) parseArgs(args map[string][]byte) (string, string, error) {
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

//////////// Group ///////////
type Group struct {
	GroupID    string   `json:"name,omitempty"`
	Admin      []string `json:"admin,omitempty"`
	Identities []string `json:"identities,omitempty"`
}

// methodEditGroup 控制平行链对应的权限管理，被称为平行链群组or群组，旨在向外提供平行链权限信息
func (p *paraChainContract) editGroup(ctx contract.KContext) (*contract.Response, error) {
	group := &Group{}
	group = loadGroupArgs(ctx.Args(), group)
	if group == nil {
		return newContractErrResponse(targetNotFound, ErrGroupNotFound.Error()), ErrGroupNotFound
	}
	// 1. 查看Group群组是否存在
	groupBytes, err := ctx.Get(ParaChainKernelContract, []byte(group.GroupID))
	if err != nil {
		return newContractErrResponse(targetNotFound, ErrGroupNotFound.Error()), err
	}

	// 2. 查看发起者是否有权限修改
	chainGroup := Group{}
	err = json.Unmarshal(groupBytes, &chainGroup)
	if err != nil {
		return newContractErrResponse(internalServerErr, err.Error()), err
	}
	if !isContain(chainGroup.Admin, ctx.Initiator()) {
		return newContractErrResponse(unAuthorized, ErrUnAuthorized.Error()), ErrUnAuthorized
	}

	// 3. 发起修改
	if group.Admin == nil { // 必须要有admin权限
		group.Admin = chainGroup.Admin
	}
	rawBytes, err := json.Marshal(group)
	if err != nil {
		return newContractErrResponse(internalServerErr, err.Error()), err
	}
	if err := ctx.Put(ParaChainKernelContract, []byte(group.GroupID), rawBytes); err != nil {
		return newContractErrResponse(internalServerErr, err.Error()), err
	}

	// 4. 通知event
	e := protos.ContractEvent{
		Name: paraChainEventName,
		Body: rawBytes,
	}
	ctx.AddEvent(&e)

	delta := contract.Limits{
		XFee: p.MinNewChainAmount,
	}
	ctx.AddResourceUsed(delta)
	return &contract.Response{
		Status: success,
		Body:   []byte("Edit Group success"),
	}, nil
}

// methodGetGroup 平行链群组读方法
func (p *paraChainContract) getGroup(ctx contract.KContext) (*contract.Response, error) {
	group := &Group{}
	group = loadGroupArgs(ctx.Args(), group)
	if group == nil {
		return newContractErrResponse(targetNotFound, ErrGroupNotFound.Error()), ErrGroupNotFound
	}
	groupBytes, err := ctx.Get(ParaChainKernelContract, []byte(group.GroupID))
	if err != nil {
		return newContractErrResponse(targetNotFound, ErrGroupNotFound.Error()), err
	}
	err = json.Unmarshal(groupBytes, group)
	if err != nil {
		return newContractErrResponse(internalServerErr, err.Error()), err
	}
	// 仅群组有权限的节点方可访问该key
	if !isContain(group.Admin, ctx.Initiator()) && !isContain(group.Identities, ctx.Initiator()) {
		return newContractErrResponse(unAuthorized, ErrUnAuthorized.Error()), nil
	}
	return &contract.Response{
		Status: success,
		Body:   groupBytes,
	}, nil
}

func loadGroupArgs(args map[string][]byte, group *Group) *Group {
	g := &Group{
		GroupID:    group.GroupID,
		Admin:      group.Admin,
		Identities: group.Identities,
	}
	bcName, ok := args["name"]
	if !ok {
		return nil
	}
	g.GroupID = string(bcName)
	admin, ok := args["admin"]
	if !ok {
		return g
	}
	adminStr := string(admin)
	g.Admin = strings.Split(adminStr, ";")
	ids, ok := args["identities"]
	if !ok {
		return g
	}
	identitiesStr := string(ids)
	g.Identities = strings.Split(identitiesStr, ";")
	return g
}

func isContain(items []string, item string) bool {
	for _, eachItem := range items {
		if eachItem == item {
			return true
		}
	}
	return false
}

func newContractErrResponse(status int, msg string) *contract.Response {
	return &contract.Response{
		Status:  status,
		Message: msg,
	}
}

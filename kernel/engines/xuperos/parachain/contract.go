package parachain

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"github.com/xuperchain/xupercore/bcs/ledger/xledger/ledger"
	"github.com/xuperchain/xupercore/bcs/ledger/xledger/utils"
	"github.com/xuperchain/xupercore/kernel/contract"
	"github.com/xuperchain/xupercore/kernel/engines/xuperos/common"
	kledger "github.com/xuperchain/xupercore/kernel/ledger"
	"github.com/xuperchain/xupercore/protos"
)

var (
	ErrGroupNotFound = errors.New("group not found")
	ErrUnAuthorized  = errors.New("unAuthorized")
	ErrChainNotFound = errors.New("chain not found")
	ErrBcNameEmpty   = errors.New("block chain name is empty")
	ErrBcDataEmpty   = errors.New("first block data is empty")
	ErrAdminEmpty    = errors.New("no administrator")
)

const (
	paraChainEventName  = "EditParaGroups"
	genesisConfigPrefix = "$G_"
)

type paraChainContract struct {
	BcName            string
	MinNewChainAmount int64
	ChainCtx          *common.ChainCtx
	NewChainWhiteList map[string]bool
}

func NewParaChainContract(bcName string, minNewChainAmount int64, newChainWhiteList map[string]bool, chainCtx *common.ChainCtx) *paraChainContract {
	t := &paraChainContract{
		BcName:            bcName,
		MinNewChainAmount: minNewChainAmount,
		ChainCtx:          chainCtx,
		NewChainWhiteList: newChainWhiteList,
	}

	return t
}

type createChainMessage struct {
	BcName        string `json:"name"`
	GenesisConfig string `json:"genesis_config"`
	Group         Group  `json:"group"`
}

type stopChainMessage struct {
	BcName string `json:"name"`
}

type refreshMessage struct {
	BcName        string `json:"name"`
	GenesisConfig string `json:"genesis_config"`
	Group         Group  `json:"group"`
}

// handleCreateChain 创建平行链的异步事件方法，若账本不存在则创建，加载链到引擎，需幂等
func (p *paraChainContract) handleCreateChain(ctx common.TaskContext) error {
	var args createChainMessage
	err := ctx.ParseArgs(&args)
	if err != nil {
		return err
	}

	// 查看当前节点是否有权限创建/获取该平行链
	if !args.Group.hasAccessAuth(p.ChainCtx.Address.Address) {
		return nil
	}
	return p.doCreateChain(args.BcName, args.GenesisConfig)
}

func (p *paraChainContract) doCreateChain(bcName string, bcGenesisConfig string) error {
	if _, err := p.ChainCtx.EngCtx.ChainM.Get(bcName); err == nil {
		p.ChainCtx.XLog.Warn("Chain is running, no need be created", "chain", bcName)
		return nil
	}
	err := utils.CreateLedgerWithData(bcName, []byte(bcGenesisConfig), p.ChainCtx.EngCtx.EnvCfg)
	if err != nil && err != utils.ErrBlockChainExist {
		return err
	}
	if err == utils.ErrBlockChainExist {
		p.ChainCtx.XLog.Warn("Chain created before, load again", "chain", bcName)

	}
	return p.ChainCtx.EngCtx.ChainM.LoadChain(bcName)
}

// handleStopChain 关闭平行链，仅从引擎中卸载，不处理账本，需幂等
func (p *paraChainContract) handleStopChain(ctx common.TaskContext) error {
	var args stopChainMessage
	err := ctx.ParseArgs(&args)
	if err != nil {
		return err
	}
	return p.doStopChain(args.BcName)
}

func (p *paraChainContract) doStopChain(bcName string) error {
	if _, err := p.ChainCtx.EngCtx.ChainM.Get(bcName); err != nil {
		p.ChainCtx.XLog.Warn("Chain hasn't been loaded yet", "chain", bcName)
		return nil
	}
	return p.ChainCtx.EngCtx.ChainM.Stop(bcName)
}

func (p *paraChainContract) handleRefreshChain(ctx common.TaskContext) error {
	var args refreshMessage
	err := ctx.ParseArgs(&args)
	if err != nil {
		return err
	}
	// 根据当前节点目前是否有权限获取该链，决定当前是停掉链还是加载链
	if args.Group.hasAccessAuth(p.ChainCtx.Address.Address) && args.Group.IsParaChainEnable() {
		return p.doCreateChain(args.BcName, args.GenesisConfig)
	}
	return p.doStopChain(args.BcName)
}

func (p *paraChainContract) createChain(ctx contract.KContext) (*contract.Response, error) {
	if p.BcName != p.ChainCtx.EngCtx.EngCfg.RootChain {
		return nil, ErrUnAuthorized
	}
	if len(p.NewChainWhiteList) > 0 && !p.NewChainWhiteList[ctx.Initiator()] {
		return newContractErrResponse(http.StatusForbidden, utils.ErrCreateChainPermission.Error()),
			utils.ErrCreateChainPermission
	}
	bcName, bcData, bcGroup, err := p.parseArgs(ctx.Args())
	if err != nil {
		return nil, err
	}

	// 1. 群组相关字段改写
	// 确保未创建过该链
	chainRes, _ := ctx.Get(ParaChainKernelContract, []byte(bcName))
	if chainRes != nil {
		return newContractErrResponse(http.StatusForbidden, utils.ErrBlockChainExist.Error()), utils.ErrBlockChainExist
	}
	// 创建链时，自动写入Group信息
	group := &Group{
		GroupID:    bcName,
		Admin:      []string{ctx.Initiator()},
		Identities: nil,
		Status:     ParaChainStatusStart,
	}
	if bcGroup != nil {
		group = bcGroup
	}
	rawBytes, err := json.Marshal(group)
	if err != nil {
		return newContractErrResponse(http.StatusInternalServerError, err.Error()), err
	}
	// 写群组信息
	if err := ctx.Put(ParaChainKernelContract,
		[]byte(bcName), rawBytes); err != nil {
		return newContractErrResponse(http.StatusInternalServerError, err.Error()), err
	}
	// 写创世块配置信息
	if err := ctx.Put(ParaChainKernelContract,
		[]byte(genesisConfigPrefix+bcName), []byte(bcData)); err != nil {
		return newContractErrResponse(http.StatusInternalServerError, err.Error()), err
	}

	// 2. 群组注册完毕后，再进行异步事件调用
	// 当该Tx被打包上链时，将运行CreateBlockChain注册的handler，并输入参数
	message := &createChainMessage{
		BcName:        bcName,
		GenesisConfig: bcData,
		Group:         *group,
	}
	err = ctx.EmitAsyncTask("CreateBlockChain", message)
	if err != nil {
		return newContractErrResponse(http.StatusInternalServerError, err.Error()), err
	}

	delta := contract.Limits{
		XFee: p.MinNewChainAmount,
	}
	ctx.AddResourceUsed(delta)

	return &contract.Response{
		Status: http.StatusOK,
		Body:   []byte("CreateBlockChain success"),
	}, nil
}

func (p *paraChainContract) stopChain(ctx contract.KContext) (*contract.Response, error) {
	// 1. 查看输入参数是否正确
	if p.BcName != p.ChainCtx.EngCtx.EngCfg.RootChain {
		return nil, ErrUnAuthorized
	}
	if ctx.Args()["name"] == nil {
		return nil, ErrBcNameEmpty
	}
	bcName := string(ctx.Args()["name"])
	if bcName == "" {
		return nil, ErrBcNameEmpty
	}

	// 2. 查看是否包含相关群组，确保链已经创建过
	groupBytes, err := ctx.Get(ParaChainKernelContract, []byte(bcName))
	if err != nil {
		return newContractErrResponse(http.StatusInternalServerError, err.Error()), err
	}
	if groupBytes == nil {
		return newContractErrResponse(http.StatusForbidden, ErrChainNotFound.Error()), ErrChainNotFound
	}

	// 3. 查看发起者是否有权限停用
	chainGroup := Group{}
	err = json.Unmarshal(groupBytes, &chainGroup)
	if err != nil {
		return newContractErrResponse(http.StatusInternalServerError, err.Error()), err
	}
	if !chainGroup.hasAdminAuth(ctx.Initiator()) {
		return newContractErrResponse(http.StatusForbidden, ErrUnAuthorized.Error()), ErrUnAuthorized
	}

	// 4. 记录群组运行状态，并写入账本
	chainGroup.Status = ParaChainStatusStop
	rawBytes, err := json.Marshal(chainGroup)
	if err != nil {
		return newContractErrResponse(http.StatusInternalServerError, err.Error()), err
	}

	if err := ctx.Put(ParaChainKernelContract,
		[]byte(bcName), rawBytes); err != nil {
		return newContractErrResponse(http.StatusInternalServerError, err.Error()), err
	}

	// 5. 将该链停掉
	message := stopChainMessage{
		BcName: bcName,
	}
	err = ctx.EmitAsyncTask("StopBlockChain", message)
	if err != nil {
		return newContractErrResponse(http.StatusInternalServerError, err.Error()), err
	}

	delta := contract.Limits{
		XFee: p.MinNewChainAmount,
	}
	ctx.AddResourceUsed(delta)

	return &contract.Response{
		Status: http.StatusOK,
		Body:   []byte("StopBlockChain success"),
	}, nil
}

func (p *paraChainContract) parseArgs(args map[string][]byte) (string, string, *Group, error) {
	// 检查bcName和bcData是否合法
	bcName := ""
	bcData := ""
	if args["name"] == nil {
		return bcName, bcData, nil, ErrBcNameEmpty
	}
	if args["data"] == nil {
		return bcName, bcData, nil, ErrBcDataEmpty
	}
	bcName = string(args["name"])
	bcData = string(args["data"])
	if bcName == "" {
		return bcName, bcData, nil, ErrBcNameEmpty
	}
	if bcName == p.ChainCtx.EngCtx.EngCfg.RootChain {
		return bcName, bcData, nil, utils.ErrBlockChainExist
	}
	if bcData == "" {
		return bcName, bcData, nil, ErrBcDataEmpty
	}
	// check data format, prevent panic
	bcCfg := &ledger.RootConfig{}
	err := json.Unmarshal(args["data"], bcCfg)
	if err != nil {
		return bcName, bcData, nil, fmt.Errorf("first block data error.err:%v", err)
	}
	if args["group"] == nil {
		return bcName, bcData, nil, nil
	}

	// 若群组存在检查群组是否合法
	var bcGroup Group
	err = json.Unmarshal(args["group"], &bcGroup)
	if err != nil {
		return bcName, bcData, nil, fmt.Errorf("group data error.err:%v", err)
	}
	if bcGroup.GroupID != bcName {
		return bcName, bcData, nil, fmt.Errorf("group name should be same with the parachain name")
	}
	if len(bcGroup.Admin) == 0 {
		return bcName, bcData, nil, ErrAdminEmpty
	}
	return bcName, bcData, &bcGroup, nil
}

// methodEditGroup 控制平行链对应的权限管理，被称为平行链群组or群组，旨在向外提供平行链权限信息
func (p *paraChainContract) editGroup(ctx contract.KContext) (*contract.Response, error) {
	group := &Group{}
	group, err := loadGroupArgs(ctx.Args(), group)
	if err != nil {
		return newContractErrResponse(http.StatusNotFound, err.Error()), err
	}
	// 1. 查看Group群组是否存在
	groupBytes, err := ctx.Get(ParaChainKernelContract, []byte(group.GroupID))
	if err != nil {
		return newContractErrResponse(http.StatusNotFound, ErrGroupNotFound.Error()), err
	}

	// 2. 查看发起者是否有权限修改
	chainGroup := Group{}
	err = json.Unmarshal(groupBytes, &chainGroup)
	if err != nil {
		return newContractErrResponse(http.StatusInternalServerError, err.Error()), err
	}
	if !chainGroup.hasAdminAuth(ctx.Initiator()) {
		return newContractErrResponse(http.StatusForbidden, ErrUnAuthorized.Error()), ErrUnAuthorized
	}

	// 3. 发起修改
	if group.Admin == nil { // 必须要有admin权限
		group.Admin = chainGroup.Admin
	}
	rawBytes, err := json.Marshal(group)
	if err != nil {
		return newContractErrResponse(http.StatusInternalServerError, err.Error()), err
	}
	if err := ctx.Put(ParaChainKernelContract, []byte(group.GroupID), rawBytes); err != nil {
		return newContractErrResponse(http.StatusInternalServerError, err.Error()), err
	}

	// 4. 通知event
	e := protos.ContractEvent{
		Name: paraChainEventName,
		Body: rawBytes,
	}
	ctx.AddEvent(&e)

	// 5. 发起另一个异步事件，旨在根据不同链的状况停掉链或者加载链
	genesisConfig, err := ctx.Get(ParaChainKernelContract, []byte(genesisConfigPrefix+group.GroupID))
	if err != nil {
		err = fmt.Errorf("get genesis config failed when edit the group, bcName = %s, err = %v",
			group.GroupID, err)
		return newContractErrResponse(http.StatusNotFound, ErrGroupNotFound.Error()), err
	}
	message := &refreshMessage{
		BcName:        group.GroupID,
		GenesisConfig: string(genesisConfig),
		Group:         *group,
	}
	err = ctx.EmitAsyncTask("RefreshBlockChain", message)
	if err != nil {
		return newContractErrResponse(http.StatusInternalServerError, err.Error()), err
	}

	delta := contract.Limits{
		XFee: p.MinNewChainAmount,
	}
	ctx.AddResourceUsed(delta)
	return &contract.Response{
		Status: http.StatusOK,
		Body:   []byte("Edit Group success"),
	}, nil
}

// methodGetGroup 平行链群组读方法
func (p *paraChainContract) getGroup(ctx contract.KContext) (*contract.Response, error) {
	group := &Group{}
	group, err := loadGroupArgs(ctx.Args(), group)
	if err != nil {
		return newContractErrResponse(http.StatusNotFound, err.Error()), err
	}
	groupBytes, err := ctx.Get(ParaChainKernelContract, []byte(group.GroupID))
	if err != nil {
		return newContractErrResponse(http.StatusNotFound, ErrGroupNotFound.Error()), err
	}
	err = json.Unmarshal(groupBytes, group)
	if err != nil {
		return newContractErrResponse(http.StatusInternalServerError, err.Error()), err
	}
	// 仅群组有权限的节点方可访问该key
	if !group.hasAccessAuth(ctx.Initiator()) {
		return newContractErrResponse(http.StatusForbidden, ErrUnAuthorized.Error()), nil
	}
	return &contract.Response{
		Status: http.StatusOK,
		Body:   groupBytes,
	}, nil
}

func loadGroupArgs(args map[string][]byte, group *Group) (*Group, error) {
	g := &Group{
		GroupID:    group.GroupID,
		Admin:      group.Admin,
		Identities: group.Identities,
	}
	bcNameBytes, ok := args["name"]
	if !ok {
		return nil, ErrBcNameEmpty
	}
	g.GroupID = string(bcNameBytes)
	if g.GroupID == "" {
		return nil, ErrBcNameEmpty
	}

	adminBytes, ok := args["admin"]
	if !ok {
		return g, nil
	}
	err := json.Unmarshal(adminBytes, &g.Admin)
	if err != nil {
		return nil, err
	}

	idsBytes, ok := args["identities"]
	if !ok {
		return g, nil
	}
	err = json.Unmarshal(idsBytes, &g.Identities)
	if err != nil {
		return nil, err
	}
	return g, nil
}

func newContractErrResponse(status int, msg string) *contract.Response {
	return &contract.Response{
		Status:  status,
		Message: msg,
	}
}

func GetParaChainGroup(reader kledger.XMSnapshotReader, bcName string) (Group, error) {
	group := Group{}
	val, err := reader.Get(ParaChainKernelContract, []byte(bcName))
	if err != nil {
		return group, err
	}
	err = json.Unmarshal(val, &group)
	return group, err
}

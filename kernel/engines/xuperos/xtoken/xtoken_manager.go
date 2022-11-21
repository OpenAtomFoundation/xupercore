package xtoken

import (
	"errors"
	"fmt"

	"github.com/spf13/viper"
	"github.com/xuperchain/xupercore/kernel/contract"
)

const (
	ConfigName = "engine.yaml"
)

type XTokenConfig struct {
	XTokenAdmins       map[string]bool
	XTokenFee          map[string]int64
	XTokenContractName string
}

type Manager struct {
	Ctx *Context
}

func NewManager(ctx *Context) (*Manager, error) {
	if ctx == nil || ctx.Contract == nil {
		return nil, errors.New("xtoken contract ctx set error")
	}

	admins := ctx.ChainCtx.Ledger.GenesisBlock.GetConfig().XTokenAdmins
	fee := ctx.ChainCtx.Ledger.GenesisBlock.GetConfig().XTokenFee
	if len(admins) == 0 && len(fee) == 0 { // 必须两个都没有设置时才能读取配置文件。
		// 创世中没有设置，此时以配置文件为主
		conf, err := loadConfig(ctx.ChainCtx.EngCtx.EnvCfg.GenConfFilePath(ConfigName))
		if err != nil {
			return nil, err
		}
		admins = conf.XTokenAdmins
		fee = conf.XTokenFee
	}

	x := NewContract(admins, fee, ctx)

	register := ctx.Contract.GetKernRegistry()
	kMethods := map[string]contract.KernMethod{
		// ERC20
		NewToken:        x.NewToken,
		TotalSupply:     x.TotalSupply,
		BalanceOf:       x.BalanceOf,
		Transfer:        x.Transfer,
		TransferFrom:    x.TransferFrom,
		Approve:         x.Approve,
		Allowance:       x.Allowance,
		AddSupply:       x.AddSupply,
		Burn:            x.Burn,
		QueryToken:      x.QueryToken,
		QueryTokenOwner: x.QueryTokenOwner,

		// Proposal & vote
		Propose:            x.Propose,
		Vote:               x.Vote,
		CheckVote:          x.CheckVote,
		QueryProposal:      x.QueryProposal,
		QueryProposalVotes: x.QueryProposalVotes,
		QueryTopic:         x.QueryTopic,

		// Admins
		AddAdmins:   x.AddAdmins,
		DelAdmins:   x.DelAdmins,
		QueryAdmins: x.QueryAdmins,
		SetFee:      x.SetFee,
		GetFee:      x.GetFee,
	}

	for method, f := range kMethods {
		if _, err := register.GetKernMethod(XTokenContract, method); err != nil {
			register.RegisterKernMethod(XTokenContract, method, f)
		}
	}
	return nil, nil
}

func loadConfig(fname string) (*XTokenConfig, error) {
	viperObj := viper.New()
	viperObj.SetConfigFile(fname)
	err := viperObj.ReadInConfig()
	if err != nil {
		return nil, fmt.Errorf("read config failed.path:%s,err:%v", fname, err)
	}

	cfg := &XTokenConfig{
		XTokenAdmins: map[string]bool{},
		XTokenFee:    map[string]int64{},
	}
	if err = viperObj.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("unmatshal config failed.path:%s,err:%v", fname, err)
	}
	return cfg, nil
}

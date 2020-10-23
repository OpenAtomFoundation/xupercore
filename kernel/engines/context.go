// 统一管理系统全局上下文
package engines

import (
	"github.com/xuperchain/xupercore/kernel/common/xcontext"
)

// 区块链执行引擎级context，维护区块链运行环境
type BCEngineCtx interface {
	xcontext.BaseCtx
	GetEnvConf() *EnvConfig
	GetNetHD() XNetwork
	GetCryptoHD() XCrypto
}

type BCEngineCtxImpl struct {
	// 基础上下文
	xcontext.BaseCtxImpl
	// 内核运行环境配置
	envCfg *EnvConfig
	// 网络组件句柄
	netHD XNetwork
	// 加密组件句柄
	cryptoHD XCrypto
}

func (t *BCEngineCtxImpl) GetEnvConf() *EnvConfig {
	if t == nil || t.envCfg == nil {
		return nil
	}

	return t.envCfg
}

func (t *BCEngineCtxImpl) GetNetHD() XNetwork {
	if t == nil || t.netHD == nil {
		return nil
	}

	return t.netHD
}

func (t *BCEngineCtxImpl) GetCryptoHD() XCrypto {
	if t == nil || t.cryptoHD == nil {
		return nil
	}

	return t.cryptoHD
}

// 链级别上下文，维护链级别上下文，每条平行链各有一个
type ChainCtx interface {
	xcontext.BaseCtx
	GetLedgerHD() XLedger
	GetConsHD() XConsensus
	GetContractHD() XContract
	GetPermHD() XPermission
}

type ChainCtxImpl struct {
	// 基础上下文
	xcontext.BaseCtxImpl
	// 账本
	ledgerHD XLedger
	// 共识
	consHD XConsensus
	// 合约
	contractHD XContract
	// 权限
	permHD XPermission
}

func (t *ChainCtxImpl) GetLedgerHD() XLedger {
	if t == nil || t.ledgerHD == nil {
		return nil
	}

	return t.ledgerHD
}

func (t *ChainCtxImpl) GetConsHD() XConsensus {
	if t == nil || t.consHD == nil {
		return nil
	}

	return t.consHD
}

func (t *ChainCtxImpl) GetContractHD() XContract {
	if t == nil || t.contractHD == nil {
		return nil
	}

	return t.contractHD
}

func (t *ChainCtxImpl) GetPermHD() XPermission {
	if t == nil || t.permHD == nil {
		return nil
	}

	return t.permHD
}

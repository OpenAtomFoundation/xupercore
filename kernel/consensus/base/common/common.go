package utils

import (
	"encoding/json"
	"errors"
	"strings"

	chainedBft "github.com/xuperchain/xupercore/kernel/consensus/base/driver/chained-bft"
	cctx "github.com/xuperchain/xupercore/kernel/consensus/context"
)

var (
	EmptyValidors    = errors.New("Current validators is empty.")
	NotValidContract = errors.New("Cannot get valid res with contract.")
)

// AddressEqual 判断两个validators地址是否相等
func AddressEqual(a []string, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i, _ := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// initQCTree 创建了smr需要的QC树存储，该Tree存储了目前待commit的QC信息
func InitQCTree(startHeight int64, ledger cctx.LedgerRely) *chainedBft.QCPendingTree {
	// 初始状态，应该是start高度的前一个区块为genesisQC
	b, err := ledger.QueryBlockByHeight(startHeight - 1)
	if err != nil {
		return nil
	}
	initQC := &chainedBft.QuorumCert{
		VoteInfo: &chainedBft.VoteInfo{
			ProposalId:   b.GetBlockid(),
			ProposalView: startHeight - 1,
		},
		LedgerCommitInfo: &chainedBft.LedgerCommitInfo{
			CommitStateId: b.GetBlockid(),
		},
	}
	rootNode := &chainedBft.ProposalNode{
		In: initQC,
	}
	return &chainedBft.QCPendingTree{
		Genesis:  rootNode,
		Root:     rootNode,
		HighQC:   rootNode,
		CommitQC: rootNode,
	}
}

func LoadValidatorsMultiInfo(res []byte, addrToNet *map[string]string) ([]string, error) {
	if res == nil {
		return nil, NotValidContract
	}
	// 读取最新的validators值
	contractInfo := chainedBft.ProposerInfo{}
	if err := json.Unmarshal(res, &contractInfo); err != nil {
		return nil, err
	}
	validators := strings.Split(contractInfo.Address, ";") // validators由分号隔开
	if len(validators) == 0 {
		return nil, EmptyValidors
	}
	neturls := strings.Split(contractInfo.Neturl, ";") // neturls由分号隔开
	if len(neturls) != len(validators) {
		return nil, EmptyValidors
	}
	for i, v := range validators {
		(*addrToNet)[v] = neturls[i]
	}
	return validators, nil
}

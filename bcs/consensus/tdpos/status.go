package tdpos

import (
	"encoding/json"
)

// tdposStatus 实现了ConsensusStatus接口
type TdposStatus struct {
	Version     int64 `json:"version"`
	StartHeight int64 `json:"startHeight"`
	Index       int   `json:"index"`
	election    *tdposSchedule
}

// 获取共识版本号
func (t *TdposStatus) GetVersion() int64 {
	return t.Version
}

// 共识起始高度
func (t *TdposStatus) GetConsensusBeginInfo() int64 {
	return t.StartHeight
}

// 获取共识item所在consensus slice中的index
func (t *TdposStatus) GetStepConsensusIndex() int {
	return t.Index
}

// 获取共识类型
func (t *TdposStatus) GetConsensusName() string {
	return "tdpos"
}

// 获取当前状态机term
func (t *TdposStatus) GetCurrentTerm() int64 {
	return t.election.curTerm
}

// 获取当前矿工信息
func (t *TdposStatus) GetCurrentValidatorsInfo() []byte {
	var v []*ProposerInfo
	for _, a := range t.election.proposers {
		v = append(v, &ProposerInfo{
			Address: a,
			Neturl:  t.election.netUrlMap[a],
		})
	}
	i := ValidatorsInfo{
		Validators: v,
	}
	b, _ := json.Marshal(i)
	return b
}

type ValidatorsInfo struct {
	Validators []*ProposerInfo `json:"validators"`
}

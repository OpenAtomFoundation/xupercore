package tdpos

import (
	"encoding/json"
)

type ValidatorsInfo struct {
	Validators   []string `json:"validators"`
	Miner        string   `json:"miner"`
	Curterm      int64    `json:"curterm"`
	ContractInfo string   `json:"contract"`
}

// tdposStatus 实现了ConsensusStatus接口
type TdposStatus struct {
	Name        string
	Version     int64 `json:"version"`
	StartHeight int64 `json:"startHeight"`
	Index       int   `json:"index"`
	election    *tdposSchedule
}

// 获取共识版本号
func (t *TdposStatus) GetVersion() int64 {
	return t.Version
}

func (t *TdposStatus) GetConsensusBeginInfo() int64 {
	return t.StartHeight
}

// 获取共识item所在consensus slice中的index
func (t *TdposStatus) GetStepConsensusIndex() int {
	return t.Index
}

// 获取共识类型
func (t *TdposStatus) GetConsensusName() string {
	return t.Name
}

// 获取当前状态机term
func (t *TdposStatus) GetCurrentTerm() int64 {
	return t.election.curTerm
}

// 获取当前矿工信息
func (t *TdposStatus) GetCurrentValidatorsInfo() []byte {
	var validators []string
	for _, a := range t.election.validators {
		validators = append(validators, a)
	}
	v := ValidatorsInfo{
		Validators:   validators,
		Curterm:      t.election.curTerm,
		Miner:        t.election.miner,
		ContractInfo: "pls invoke getTdposInfos",
	}
	b, _ := json.Marshal(&v)
	return b
}

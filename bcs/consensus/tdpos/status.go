package tdpos

import (
	"encoding/json"
	"fmt"
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
		ContractInfo: t.getTdposInfos(),
	}
	b, _ := json.Marshal(&v)
	return b
}

// getTdposInfos 替代原来getTdposInfos合约
func (t *TdposStatus) getTdposInfos() string {
	height := t.election.ledger.GetTipBlock().GetHeight()
	// nominate信息
	res, err := t.election.getSnapshotKey(height, t.election.bindContractBucket, []byte(fmt.Sprintf("%s_%d_%s", t.Name, t.Version, nominateKey)))
	if res == nil || err != nil {
		return ""
	}
	if err != nil {
		t.election.log.Error("TdposStatus::getTdposInfos::getSnapshotKey err.", "err", err)
		return ""
	}
	nominateValue := NewNominateValue()
	if err := json.Unmarshal(res, &nominateValue); err != nil {
		t.election.log.Error("TdposStatus::getTdposInfos::Unmarshal nominate err.", "err", err)
		return ""
	}

	// vote信息
	voteMap := make(map[string]voteValue)
	for candidate, _ := range nominateValue {
		// 读取投票存储
		voteKey := fmt.Sprintf("%s_%d_%s%s", t.Name, t.Version, voteKeyPrefix, candidate)
		res, err = t.election.getSnapshotKey(height, t.election.bindContractBucket, []byte(voteKey))
		if err != nil {
			t.election.log.Error("TdposStatus::getTdposInfos::load vote read set err when get key.", "key", voteKey)
			continue
		}
		voteValue := NewvoteValue()
		if res == nil {
			continue
		}
		if err := json.Unmarshal(res, &voteValue); err != nil {
			t.election.log.Error("TdposStatus::getTdposInfos::load vote read set err.", "res", res, "err", err)
			continue
		}
		voteMap[candidate] = voteValue
	}

	// revoke信息
	res, err = t.election.getSnapshotKey(height, t.election.bindContractBucket, []byte(fmt.Sprintf("%s_%d_%s", t.Name, t.Version, revokeKey)))
	if err != nil {
		t.election.log.Error("TdposStatus::getTdposInfos::load revoke read set err when get key.")
		return ""
	}
	revokeValue := NewRevokeValue()
	if res != nil {
		if err := json.Unmarshal(res, &revokeValue); err != nil {
			t.election.log.Error("TdposStatus::getTdposInfos::load revoke read set err.", "res", res, "err", err)
			return ""
		}
	}
	r := `{"nominate":` + fmt.Sprintf("%v", nominateValue) + `,"vote":` + fmt.Sprintf("%v", voteMap) + `,"revoke":` + fmt.Sprintf("%v", revokeValue) + `}`
	return r
}

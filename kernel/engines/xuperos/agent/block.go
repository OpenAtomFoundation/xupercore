package agent

import (
	"encoding/json"
	"fmt"

	"github.com/xuperchain/xupercore/bcs/ledger/xledger/ledger"
	lpb "github.com/xuperchain/xupercore/bcs/ledger/xledger/xldgpb"
)

type BlockAgent struct {
	blk *lpb.InternalBlock
}

// 兼容xledger账本历史原因共识部分字段分开存储在区块中
type ConsensusStorage struct {
	TargetBits  int32           `json:"targetBits,omitempty"`
	Justify     *lpb.QuorumCert `json:"justify,omitempty"`
	CurTerm     int64           `json:"curTerm,omitempty"`
	CurBlockNum int64           `json:"curBlockNum,omitempty"`
}

func NewBlockAgent(blk *lpb.InternalBlock) *BlockAgent {
	return &BlockAgent{
		blk: blk,
	}
}

func (t *BlockAgent) GetProposer() []byte {
	return t.blk.GetProposer()
}

func (t *BlockAgent) GetHeight() int64 {
	return t.blk.GetHeight()
}

func (t *BlockAgent) GetBlockid() []byte {
	return t.blk.GetBlockid()
}

// 共识记录信息，xledger账本由于历史原因需要做下转换
func (t *BlockAgent) GetConsensusStorage() ([]byte, error) {
	strg := &ConsensusStorage{
		TargetBits: t.blk.GetTargetBits(),
		Justify:    t.blk.GetJustify(),
		CurTerm:    t.blk.GetCurTerm(),
	}

	js, err := json.Marshal(strg)
	if err != nil {
		return nil, fmt.Errorf("json marshal failed.err:%v", err)
	}

	return js, nil
}

func (t *BlockAgent) GetTimestamp() int64 {
	return t.blk.GetTimestamp()
}

// 用于pow挖矿时需更新nonce
func (t *BlockAgent) SetItem(item string, value interface{}) error {
	switch item {
	case "nonce":
		nonce, ok := value.(int32)
		if !ok {
			return fmt.Errorf("nonce type not match")
		}
		t.blk.Nonce = nonce
	case "blockid":
		blockId, ok := value.([]byte)
		if !ok {
			return fmt.Errorf("blockid type not match")
		}
		t.blk.Blockid = blockId
	case "sign":
		sign, ok := value.([]byte)
		if !ok {
			return fmt.Errorf("sign type not match")
		}
		t.blk.Sign = sign
	default:
		return fmt.Errorf("item not support set")
	}

	return nil
}

// 计算BlockId
func (t *BlockAgent) MakeBlockId() ([]byte, error) {
	blkId, err := ledger.MakeBlockID(t.blk)
	if err != nil {
		return nil, err
	}
	t.blk.Blockid = blkId
	return blkId, nil
}

func (t *BlockAgent) GetPreHash() []byte {
	return t.blk.GetPreHash()
}

func (t *BlockAgent) GetPublicKey() string {
	return string(t.blk.GetPubkey())
}

func (t *BlockAgent) GetSign() []byte {
	return t.blk.GetSign()
}

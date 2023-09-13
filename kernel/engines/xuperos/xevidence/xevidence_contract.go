package xevidence

import (
	"encoding/json"
	"fmt"

	"github.com/pkg/errors"
	"github.com/xuperchain/xupercore/kernel/contract"
	"github.com/xuperchain/xupercore/kernel/contract/sandbox"
	"github.com/xuperchain/xupercore/lib/storage/kvdb"
)

// Contract struct represents the Xuperos contract interface.
type Contract struct {
	contractCtx *Context
	cfg         *XEvidenceConfig
}

type Evidence struct {
	Hash    string `json:"hash,omitempty"`
	Content string `json:"content,omitempty"`
	Desc    string `json:"desc,omitempty"`
	Sender  string `json:"sender,omitempty"`
}

// NewContract creates a new instance of the Xuperos contract.
func NewContract(ctx *Context, cfg *XEvidenceConfig) *Contract {
	return &Contract{contractCtx: ctx, cfg: cfg}
}

// Save check args and save evidence.
func (c *Contract) Save(ctx contract.KContext) (*contract.Response, error) {
	c.contractCtx.XLog.Debug("XEvidence save method")
	hash := string(ctx.Args()["hash"])
	if len(hash) == 0 {
		return nil, fmt.Errorf("hash is required")
	}
	if len(hash) > 256 { // 这里256为字符串长度
		return nil, fmt.Errorf("hash length too long")
	}

	content := string(ctx.Args()["content"])
	desc := string(ctx.Args()["desc"])
	if len(content) == 0 && len(desc) == 0 {
		// 不能同时为空
		// 如果只有 hash，没有 content 和 desc，后期难以说明此存证的内容
		return nil, fmt.Errorf("content and desc is required")
	}

	if len(desc) > 256 {
		return nil, fmt.Errorf("desc length too long")
	}

	evidenceExit, err := getEvidenceByHash(ctx, hash)
	if err != nil {
		return nil, err
	}
	if len(evidenceExit) != 0 {
		return nil, fmt.Errorf("evidence already exists hash: %s", hash)
	}

	evidence := &Evidence{
		Hash:    hash,
		Content: content,
		Desc:    desc,
		Sender:  ctx.Initiator(),
	}

	err = saveEvidence(ctx, evidence)
	if err != nil {
		return nil, fmt.Errorf("save evidence error: %s", err)
	}
	c.contractCtx.XLog.Debug("XEvidence", "save evidence succ, hash", hash)
	err = addSaveFee(ctx, c.cfg.XEvidenceSaveMethodFeeConfig, content)
	if err != nil {
		return nil, err
	}
	return &contract.Response{
		Status: Success,
	}, nil
}

// Get get evidence by hash
func (c *Contract) Get(ctx contract.KContext) (*contract.Response, error) {
	hash := string(ctx.Args()["hash"])
	if len(hash) == 0 {
		return nil, fmt.Errorf("hash is required")
	}
	evidence, err := getEvidenceByHash(ctx, hash)
	if err != nil {
		return nil, err
	}
	if len(evidence) == 0 {
		return nil, fmt.Errorf("evidence not found hash: %s", hash)
	}

	return &contract.Response{
		Status: Success,
		Body:   evidence,
	}, nil
}

func KeyOfAdmins() string {
	return "admins"
}

func KeyOfFee(method string) string {
	return "Fee_" + method
}

func keyOfEvidence(hash string) string {
	return "XEVIDENCE_" + hash
}

// return nil and nil if no evidence exist
func getEvidenceByHash(ctx contract.KContext, hash string) ([]byte, error) {
	evidence, err := ctx.Get(XEvidence, []byte(keyOfEvidence(hash)))
	if err != nil && !kvdb.ErrNotFound(err) && !errors.Is(err, sandbox.ErrHasDel) {
		return nil, errors.Wrap(err, "query evidence error")
	}
	return evidence, nil
}

func saveEvidence(ctx contract.KContext, e *Evidence) error {
	key := keyOfEvidence(e.Hash)
	value, err := json.Marshal(e)
	if err != nil {
		return errors.Wrap(err, "save evidence marshal evidence failed")
	}
	err = ctx.Put(XEvidence, []byte(key), value)
	if err != nil {
		return errors.Wrap(err, "save evidence failed")
	}
	return nil
}

func addSaveFee(ctx contract.KContext, cfg *SaveMethodFeeConfig, content string) error {
	key := []byte(KeyOfFee(Save))
	value, err := ctx.Get(XEvidence, key)
	if err != nil && !kvdb.ErrNotFound(err) && !errors.Is(err, sandbox.ErrHasDel) {
		return errors.Wrap(err, "get fee failed")
	}
	saveCfg := &SaveMethodFeeConfig{}
	if len(value) > 0 {
		// 如果数据库中有则以数据库中为主
		err = json.Unmarshal(value, saveCfg)
		if err != nil {
			return err
		}
	} else {
		saveCfg = cfg
	}

	if len(content) > int(saveCfg.MaxLength) {
		return fmt.Errorf("content length too long")
	}

	fee := saveCfg.FeeForLengthThreshold
	if int64(len(content)) > saveCfg.LengthThreshold {
		fee += ((int64(len(content))-saveCfg.LengthThreshold)/saveCfg.LengthIncrement + 1) *
			saveCfg.FeeIncrement
	}

	delta := contract.Limits{
		XFee: fee,
	}
	ctx.AddResourceUsed(delta)
	return nil
}

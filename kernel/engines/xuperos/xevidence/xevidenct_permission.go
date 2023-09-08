package xevidence

import (
	"encoding/json"
	"math/big"

	"github.com/pkg/errors"

	"github.com/xuperchain/xupercore/kernel/contract"
	"github.com/xuperchain/xupercore/kernel/contract/sandbox"
	"github.com/xuperchain/xupercore/lib/storage/kvdb"
)

func (c *Contract) AddAdmins(ctx contract.KContext) (*contract.Response, error) {
	// 如果想添加admin，前提时创世文件或者配置文件中设置了admin。
	ok, err := c.checkPermissionWithMustHasAdmin(ctx, ctx.Initiator())
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, errors.New("check permission failed")
	}

	value := ctx.Args()["addrs"]
	if len(value) == 0 {
		return nil, errors.New("addrs param empty")
	}

	addrs := []string{}
	if err := json.Unmarshal(value, &addrs); err != nil {
		return nil, errors.Wrap(err, "addrs param unmarshal failed")
	}
	if len(addrs) == 0 {
		return nil, errors.New("addrs param empty")
	}

	admins, err := c.getAdmins(ctx)
	if err != nil {
		return nil, err
	}
	if admins == nil {
		admins = map[string]bool{}
	}
	for _, addr := range addrs {
		admins[addr] = true
	}
	// 如果参数中的地址都已经存在，此交易也会成功，但是没有修改任何数据。
	err = c.setAdmins(ctx, admins)
	if err != nil {
		return nil, err
	}

	err = c.addFee(ctx, AddAdmins)
	if err != nil {
		return nil, err
	}
	return &contract.Response{
		Status: Success,
	}, nil
}

func (c *Contract) DelAdmins(ctx contract.KContext) (*contract.Response, error) {
	// 如果想删除admin，前提时创世文件或者配置文件中设置了admin。
	ok, err := c.checkPermissionWithMustHasAdmin(ctx, ctx.Initiator())
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, errors.New("check permission failed")
	}

	value := ctx.Args()["addrs"]
	if len(value) == 0 {
		return nil, errors.New("addrs param empty")
	}

	addrs := []string{}
	if err := json.Unmarshal(value, &addrs); err != nil {
		return nil, errors.Wrap(err, "addrs param unmarshal failed")
	}
	if len(addrs) == 0 {
		return nil, errors.New("addrs param empty")
	}

	admins, err := c.getAdmins(ctx)
	if err != nil {
		return nil, err
	}
	for _, addr := range addrs {
		delete(admins, addr)
	}
	// 如果参数中的地址都不存在，此交易也会成功，但是没有修改任何数据。
	err = c.setAdmins(ctx, admins)
	if err != nil {
		return nil, err
	}

	err = c.addFee(ctx, DelAdmins)
	if err != nil {
		return nil, err
	}
	return &contract.Response{
		Status: Success,
	}, nil
}

func (c *Contract) QueryAdmins(ctx contract.KContext) (*contract.Response, error) {
	admins, err := c.getAdmins(ctx)
	if err != nil {
		return nil, err
	}
	result := []byte{}
	if len(admins) > 0 {
		value, err := json.Marshal(admins)
		if err != nil {
			return nil, err
		}
		result = value
	} else {
		if len(c.cfg.XEvidenceAdmins) > 0 {
			value, err := json.Marshal(c.cfg.XEvidenceAdmins)
			if err != nil {
				return nil, err
			}
			result = value
		}
	}

	err = c.addFee(ctx, QueryAdmins)
	if err != nil {
		return nil, err
	}
	return &contract.Response{
		Status: Success,
		Body:   result,
	}, nil
}

func (c *Contract) SetFee(ctx contract.KContext) (*contract.Response, error) {
	// 如果想修改手续费，前提时创世文件或者配置文件中设置了admin。
	ok, err := c.checkPermissionWithMustHasAdmin(ctx, ctx.Initiator())
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, errors.New("check permission failed")
	}

	method := string(ctx.Args()["method"])
	feeStr := string(ctx.Args()["fee"])
	if len(method) == 0 || len(feeStr) == 0 {
		return nil, errors.New("fee and method param can not be empty")
	}
	if method == Save {
		// Save 接口手续费配置结构特殊，这里需要单独处理。
		return c.setSaveMethodFee(ctx)
	}

	fee, ok := big.NewInt(0).SetString(feeStr, 10)
	if !ok {
		return nil, errors.New("invalid fee")
	}
	err = c.setFee(ctx, method, fee)
	if err != nil {
		return nil, err
	}
	err = c.addFee(ctx, SetFee)
	if err != nil {
		return nil, err
	}
	return &contract.Response{
		Status: Success,
	}, nil
}

func (c *Contract) setSaveMethodFee(ctx contract.KContext) (*contract.Response, error) {
	feeStr := string(ctx.Args()["fee"])
	saveCfg := &SaveMethodFeeConfig{}
	err := json.Unmarshal([]byte(feeStr), saveCfg)
	if err != nil {
		return nil, err
	}

	// 参数中和 save method cfg 无关的字段不需要入库，因此这里序列化。
	value, err := json.Marshal(saveCfg)
	if err != nil {
		return nil, err
	}
	key := []byte(KeyOfFee(Save))
	err = ctx.Put(XEvidence, key, value)
	if err != nil {
		return nil, errors.Wrap(err, "set save method fee failed")
	}
	err = c.addFee(ctx, SetFee)
	if err != nil {
		return nil, err
	}
	return &contract.Response{
		Status: Success,
	}, nil
}

func (c *Contract) GetFee(ctx contract.KContext) (*contract.Response, error) {
	method := ctx.Args()["method"]
	if len(method) == 0 {
		return nil, errors.New("method param can not be empty")
	}
	if string(method) == Save {
		return c.getSaveMethodFee(ctx)
	}
	fee, err := c.getFee(ctx, string(method))
	if err != nil {
		return nil, err
	}
	err = c.addFee(ctx, GetFee)
	if err != nil {
		return nil, err
	}
	return &contract.Response{
		Status: Success,
		Body:   []byte(fee.String()),
	}, nil
}

func (c *Contract) getSaveMethodFee(ctx contract.KContext) (*contract.Response, error) {
	key := []byte(KeyOfFee(Save))
	value, err := ctx.Get(XEvidence, key)
	if err != nil && !kvdb.ErrNotFound(err) && !errors.Is(err, sandbox.ErrHasDel) {
		return nil, errors.Wrap(err, "get save method fee failed")
	}
	err = c.addFee(ctx, GetFee)
	if err != nil {
		return nil, err
	}
	return &contract.Response{
		Status: Success,
		Body:   value,
	}, nil
}

func (c *Contract) checkPermissionWithMustHasAdmin(ctx contract.KContext, address string) (bool, error) {
	admins, err := c.getAdmins(ctx)
	if err != nil {
		return false, err
	}
	if len(admins) == 0 {
		if len(c.cfg.XEvidenceAdmins) == 0 {
			// 如果账本中没有设置admins，同时配置文件也没有设置，那么返回false。
			return false, nil
		}
		return c.cfg.XEvidenceAdmins[address], nil
	}
	return admins[address], nil
}

func (c *Contract) getFee(ctx contract.KContext, method string) (*big.Int, error) {
	key := []byte(KeyOfFee(method))
	value, err := ctx.Get(XEvidence, key)
	if err != nil && !kvdb.ErrNotFound(err) && !errors.Is(err, sandbox.ErrHasDel) {
		return nil, errors.Wrap(err, "get fee failed")
	}
	if len(value) == 0 {
		return big.NewInt(0), nil
	}
	feeBig, ok := big.NewInt(0).SetString(string(value), 10)
	if !ok {
		return nil, errors.New("getFee bigInt set string failed")
	}
	return feeBig, nil
}

func (c *Contract) setFee(ctx contract.KContext, method string, fee *big.Int) error {
	key := []byte(KeyOfFee(method))
	err := ctx.Put(XEvidence, key, []byte(fee.String()))
	if err != nil {
		return errors.Wrap(err, "set fee failed")
	}
	return nil
}

func (c *Contract) getAdmins(ctx contract.KContext) (map[string]bool, error) {
	key := []byte(KeyOfAdmins())
	value, err := ctx.Get(XEvidence, key)
	if err != nil && !kvdb.ErrNotFound(err) && !errors.Is(err, sandbox.ErrHasDel) {
		return nil, errors.Wrap(err, "get admins failed")
	}
	if len(value) == 0 {
		return nil, nil
	}
	addrs := new(map[string]bool)
	if err := json.Unmarshal(value, addrs); err != nil {
		return nil, err
	}
	return *addrs, nil
}

func (c *Contract) setAdmins(ctx contract.KContext, addrs map[string]bool) error {
	value, err := json.Marshal(addrs)
	if err != nil {
		return err
	}
	key := []byte(KeyOfAdmins())
	err = ctx.Put(XEvidence, key, value)
	if err != nil {
		return errors.Wrap(err, "set admins failed")
	}
	return nil
}

func (c *Contract) addFee(ctx contract.KContext, method string) error {
	if method == Save {
		return errors.New("Save method please call addSaveFee method")
	}

	key := []byte(KeyOfFee(method))
	value, err := ctx.Get(XEvidence, key)
	if err != nil && !kvdb.ErrNotFound(err) && !errors.Is(err, sandbox.ErrHasDel) {
		return errors.Wrap(err, "get fee failed")
	}
	fee := int64(0)
	if len(value) == 0 {
		// 如果数据库中没有，则从配置中读取。
		// 配置中没有则为0。
		fee = c.cfg.XEvidenceMethodFee[method]
	} else {
		// 如果数据库中有，则以数据库中为主。

		feeBig, ok := big.NewInt(0).SetString(string(value), 10)
		if !ok {
			return errors.New("get fee bigInt set string failed")
		}
		fee = feeBig.Int64()
	}

	delta := contract.Limits{
		XFee: fee,
	}
	ctx.AddResourceUsed(delta)
	return nil
}

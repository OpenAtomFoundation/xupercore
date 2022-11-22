package xtoken

import (
	"encoding/json"
	"math/big"

	"github.com/pkg/errors"

	"github.com/xuperchain/xupercore/kernel/contract"
	"github.com/xuperchain/xupercore/kernel/contract/sandbox"
	"github.com/xuperchain/xupercore/lib/storage/kvdb"
)

func (x *Contract) AddAdmins(ctx contract.KContext) (*contract.Response, error) {
	// 如果想添加admin，前提时创世文件或者配置文件中设置了admin。
	ok, err := x.checkPermissionWithMustHasAdmin(ctx, ctx.Initiator())
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

	admins, err := x.getAdmins(ctx)
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
	err = x.setAdmins(ctx, admins)
	if err != nil {
		return nil, err
	}

	err = x.addFee(ctx, AddAdmins)
	if err != nil {
		return nil, err
	}
	return &contract.Response{
		Status: Success,
	}, nil
}

func (x *Contract) DelAdmins(ctx contract.KContext) (*contract.Response, error) {
	// 如果想删除admin，前提时创世文件或者配置文件中设置了admin。
	ok, err := x.checkPermissionWithMustHasAdmin(ctx, ctx.Initiator())
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

	admins, err := x.getAdmins(ctx)
	if err != nil {
		return nil, err
	}
	for _, addr := range addrs {
		delete(admins, addr)
	}
	// 如果参数中的地址都不存在，此交易也会成功，但是没有修改任何数据。
	err = x.setAdmins(ctx, admins)
	if err != nil {
		return nil, err
	}

	err = x.addFee(ctx, DelAdmins)
	if err != nil {
		return nil, err
	}
	return &contract.Response{
		Status: Success,
	}, nil
}

func (x *Contract) QueryAdmins(ctx contract.KContext) (*contract.Response, error) {
	admins, err := x.getAdmins(ctx)
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
		if len(x.Admins) > 0 {
			value, err := json.Marshal(x.Admins)
			if err != nil {
				return nil, err
			}
			result = value
		}
	}

	err = x.addFee(ctx, QueryAdmins)
	if err != nil {
		return nil, err
	}
	return &contract.Response{
		Status: Success,
		Body:   result,
	}, nil
}

func (x *Contract) SetFee(ctx contract.KContext) (*contract.Response, error) {
	// 如果想修改手续费，前提时创世文件或者配置文件中设置了admin。
	ok, err := x.checkPermissionWithMustHasAdmin(ctx, ctx.Initiator())
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

	fee, ok := big.NewInt(0).SetString(feeStr, 10)
	if !ok {
		return nil, errors.New("invalid fee")
	}
	err = x.setFee(ctx, method, fee)
	if err != nil {
		return nil, err
	}
	err = x.addFee(ctx, SetFee)
	if err != nil {
		return nil, err
	}
	return &contract.Response{
		Status: Success,
	}, nil
}

func (x *Contract) GetFee(ctx contract.KContext) (*contract.Response, error) {
	method := ctx.Args()["method"]
	if len(method) == 0 {
		return nil, errors.New("method param can not be empty")
	}
	fee, err := x.getFee(ctx, string(method))
	if err != nil {
		return nil, err
	}
	err = x.addFee(ctx, GetFee)
	if err != nil {
		return nil, err
	}
	return &contract.Response{
		Status: Success,
		Body:   []byte(fee.String()),
	}, nil
}

func (x *Contract) checkPermissionWithMustHasAdmin(ctx contract.KContext, address string) (bool, error) {
	admins, err := x.getAdmins(ctx)
	if err != nil {
		return false, err
	}
	if len(admins) == 0 {
		if len(x.Admins) == 0 {
			// 如果账本中没有设置admins，同时配置文件也没有设置，那么返回false。
			return false, nil
		}
		return x.Admins[address], nil
	}
	return admins[address], nil
}

// 如果address在admins列表则返回true。
func (x *Contract) checkPermission(ctx contract.KContext, address string) (bool, error) {
	admins, err := x.getAdmins(ctx)
	if err != nil {
		return false, err
	}
	if len(admins) == 0 {
		// 如果没有通过交易设置admin，则根据配置文件获取。
		if len(x.Admins) == 0 {
			// 如果配置文件也没设置，则说明不加权限设置。
			return true, nil
		}
		return x.Admins[address], nil
	}
	return admins[address], nil
}

func (x *Contract) getFee(ctx contract.KContext, method string) (*big.Int, error) {
	key := []byte(KeyOfFee(method))
	value, err := ctx.Get(XTokenContract, key)
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

func (x *Contract) setFee(ctx contract.KContext, method string, fee *big.Int) error {
	key := []byte(KeyOfFee(method))
	err := ctx.Put(XTokenContract, key, []byte(fee.String()))
	if err != nil {
		return errors.Wrap(err, "set fee failed")
	}
	return nil
}

func (x *Contract) getAdmins(ctx contract.KContext) (map[string]bool, error) {
	key := []byte(KeyOfAdmins())
	value, err := ctx.Get(XTokenContract, key)
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

func (x *Contract) setAdmins(ctx contract.KContext, addrs map[string]bool) error {
	value, err := json.Marshal(addrs)
	if err != nil {
		return err
	}
	key := []byte(KeyOfAdmins())
	err = ctx.Put(XTokenContract, key, value)
	if err != nil {
		return errors.Wrap(err, "set admins failed")
	}
	return nil
}

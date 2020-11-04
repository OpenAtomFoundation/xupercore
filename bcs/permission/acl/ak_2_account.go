package acl

import (
	"encoding/json"
	"errors"

	"github.com/xuperchain/xupercore/kernel/contract/kernel"
	"github.com/xuperchain/xupercore/kernel/permission/acl/utils"
	"github.com/xuperchain/xupercore/kernel/permission/pb"
)

func updateThresholdWithDel(ctx kernel.KContext, aksWeight map[string]float64, accountName string) error {
	for address := range aksWeight {
		key := utils.MakeAK2AccountKey(address, accountName)
		err := ctx.DeleteObject(utils.GetAK2AccountBucket(), []byte(key))
		if err != nil {
			return err
		}
	}
	return nil
}

func updateThresholdWithPut(ctx kernel.KContext, aksWeight map[string]float64, accountName string) error {
	for address := range aksWeight {
		key := utils.MakeAK2AccountKey(address, accountName)
		err := ctx.PutObject(utils.GetAK2AccountBucket(), []byte(key), []byte("true"))
		if err != nil {
			return err
		}
	}
	return nil
}

func updateAkSetWithDel(ctx kernel.KContext, sets map[string]*pb.AkSet, accountName string) error {
	for _, akSets := range sets {
		for _, ak := range akSets.GetAks() {
			key := utils.MakeAK2AccountKey(ak, accountName)
			err := ctx.DeleteObject(utils.GetAK2AccountBucket(), []byte(key))
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func updateAkSetWithPut(ctx kernel.KContext, sets map[string]*pb.AkSet, accountName string) error {
	for _, akSets := range sets {
		for _, ak := range akSets.GetAks() {
			key := utils.MakeAK2AccountKey(ak, accountName)
			err := ctx.PutObject(utils.GetAK2AccountBucket(), []byte(key), []byte("true"))
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func updateForThreshold(ctx kernel.KContext, aksWeight map[string]float64, accountName string, method string) error {
	switch method {
	case "Del":
		return updateThresholdWithDel(ctx, aksWeight, accountName)
	case "Put":
		return updateThresholdWithPut(ctx, aksWeight, accountName)
	default:
		return errors.New("unexpected error, method only for Del or Put")
	}
}

func updateForAKSet(ctx kernel.KContext, akSets *pb.AkSets, accountName string, method string) error {
	sets := akSets.GetSets()
	switch method {
	case "Del":
		return updateAkSetWithDel(ctx, sets, accountName)
	case "Put":
		return updateAkSetWithPut(ctx, sets, accountName)
	default:
		return errors.New("unexpected error, method only for Del or Put")
	}
}

func update(ctx kernel.KContext, aclJSON []byte, accountName string, method string) error {
	if aclJSON == nil {
		return nil
	}
	acl := &pb.Acl{}
	json.Unmarshal(aclJSON, acl)
	akSets := acl.GetAkSets()
	aksWeight := acl.GetAksWeight()
	permissionRule := acl.GetPm().GetRule()

	switch permissionRule {
	case pb.PermissionRule_SIGN_THRESHOLD:
		return updateForThreshold(ctx, aksWeight, accountName, method)
	case pb.PermissionRule_SIGN_AKSET:
		return updateForAKSet(ctx, akSets, accountName, method)
	default:
		return errors.New("update ak to account reflection failed, permission model is not found")
	}
	return nil
}

func UpdateAK2AccountReflection(ctx kernel.KContext, aclOldJSON []byte, aclNewJSON []byte, accountName string) error {
	if err := update(ctx, aclOldJSON, accountName, "Del"); err != nil {
		return err
	}
	if err := update(ctx, aclNewJSON, accountName, "Put"); err != nil {
		return err
	}
	return nil
}

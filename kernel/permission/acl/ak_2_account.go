package acl

import (
	"encoding/json"
	"errors"

	"github.com/xuperchain/xupercore/kernel/contract"
	"github.com/xuperchain/xupercore/kernel/permission/acl/utils"
	pb "github.com/xuperchain/xupercore/protos"
)

func updateThresholdWithDel(ctx contract.KContext, aksWeight map[string]float64, accountName string) error {
	for address := range aksWeight {
		key := utils.MakeAK2AccountKey(address, accountName)
		err := ctx.Del(utils.GetAK2AccountBucket(), []byte(key))
		if err != nil {
			return err
		}
	}
	return nil
}

func updateThresholdWithPut(ctx contract.KContext, aksWeight map[string]float64, accountName string) error {
	for address := range aksWeight {
		key := utils.MakeAK2AccountKey(address, accountName)
		err := ctx.Put(utils.GetAK2AccountBucket(), []byte(key), []byte("true"))
		if err != nil {
			return err
		}
	}
	return nil
}

func updateAkSetWithDel(ctx contract.KContext, sets map[string]*pb.AkSet, accountName string) error {
	for _, akSets := range sets {
		for _, ak := range akSets.GetAks() {
			key := utils.MakeAK2AccountKey(ak, accountName)
			err := ctx.Del(utils.GetAK2AccountBucket(), []byte(key))
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func updateAkSetWithPut(ctx contract.KContext, sets map[string]*pb.AkSet, accountName string) error {
	for _, akSets := range sets {
		for _, ak := range akSets.GetAks() {
			key := utils.MakeAK2AccountKey(ak, accountName)
			err := ctx.Put(utils.GetAK2AccountBucket(), []byte(key), []byte("true"))
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func updateForThreshold(ctx contract.KContext, aksWeight map[string]float64, accountName string, method string) error {
	switch method {
	case "Del":
		return updateThresholdWithDel(ctx, aksWeight, accountName)
	case "Put":
		return updateThresholdWithPut(ctx, aksWeight, accountName)
	default:
		return errors.New("unexpected error, method only for Del or Put")
	}
}

func updateForAKSet(ctx contract.KContext, akSets *pb.AkSets, accountName string, method string) error {
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

func update(ctx contract.KContext, aclJSON []byte, accountName string, method string) error {
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

func UpdateAK2AccountReflection(ctx contract.KContext, aclOldJSON []byte, aclNewJSON []byte, accountName string) error {
	if err := update(ctx, aclOldJSON, accountName, "Del"); err != nil {
		return err
	}
	if err := update(ctx, aclNewJSON, accountName, "Put"); err != nil {
		return err
	}
	return nil
}

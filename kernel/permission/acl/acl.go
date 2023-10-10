package acl

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/xuperchain/xupercore/kernel/permission/acl/utils"
	pb "github.com/xuperchain/xupercore/protos"
)

// ACL is a wrapper for pb.ACL, which helps to deal with it
type ACL struct {
	pb.Acl
}

func newACL(data []byte) (*ACL, error) {
	acl := new(ACL)

	err := json.Unmarshal(data, &acl.Acl)
	if err != nil {
		return nil, fmt.Errorf("unmarshal args acl error: %v", err)
	}

	if err := acl.check(); err != nil {
		return nil, err
	}
	return acl, nil
}

// mustGetAKs get AK list when caller make sure no error occure
// usually called after checked ACL
func (l *ACL) mustGetAKs() []string {
	aks, _ := l.getAKs()
	return aks
}

// getAKs gets AK list under different permission rule,
// which works for AK2Account bucket
func (l *ACL) getAKs() ([]string, error) {
	rule := l.GetPm().GetRule()
	switch rule {
	case pb.PermissionRule_SIGN_THRESHOLD:
		aks := make([]string, 0, len(l.GetAksWeight()))
		for ak := range l.GetAksWeight() {
			aks = append(aks, ak)
		}
		return aks, nil
	case pb.PermissionRule_SIGN_AKSET:
		aks := make([]string, 0)
		for _, akSets := range l.GetAkSets().GetSets() {
			aks = append(aks, akSets.GetAks()...)
		}
		return aks, nil
	default:
		return nil, errors.New("permission rule is invalid")
	}
}

func (l *ACL) check() error {

	// check permission model
	if l.GetPm() == nil {
		return fmt.Errorf("valid acl failed, lack of argument of permission model")
	}

	// check AK limitation
	switch l.GetPm().GetRule() {
	case pb.PermissionRule_SIGN_THRESHOLD:
		aksWeight := l.GetAksWeight()
		if aksWeight == nil || len(aksWeight) > utils.GetAkLimit() {
			return fmt.Errorf("valid acl failed, aksWeight is nil or size of aksWeight is very big")
		}
	case pb.PermissionRule_SIGN_AKSET:
		akSets := l.GetAkSets()
		if akSets == nil {
			return fmt.Errorf("valid acl failed, akSets is nil")
		}
		sets := akSets.GetSets()
		if sets == nil || len(sets) > utils.GetAkLimit() {
			return fmt.Errorf("valid acl failed, Sets is nil or size of Sets is very big")
		}
	default:
		return fmt.Errorf("valid acl failed, permission model is not found")
	}

	return nil
}

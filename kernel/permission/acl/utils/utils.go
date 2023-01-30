package utils

import (
	"errors"
	"fmt"
	"strings"

	"github.com/xuperchain/xupercore/kernel/permission/acl/base"
	"github.com/xuperchain/xupercore/kernel/permission/acl/ptree"
	"github.com/xuperchain/xupercore/kernel/permission/acl/rule"
	"github.com/xuperchain/xupercore/lib/crypto/client"
	pb "github.com/xuperchain/xupercore/protos"
)

func IdentifyAK(akURI string, sign *pb.SignatureInfo, msg []byte) (bool, error) {
	if sign == nil {
		return false, errors.New("sign is nil")
	}
	ak := ExtractAddrFromAkURI(akURI)
	return VerifySign(ak, sign, msg)
}

func IdentifyAccount(aclMgr base.AclManager, account string, akURIs []string) (bool, error) {
	// aks and signs could have zero length for permission rule Null
	if aclMgr == nil {
		return false, fmt.Errorf("Invalid Param, aclMgr=%v", aclMgr)
	}

	// build perm tree
	tree, err := ptree.BuildAccountPermTree(aclMgr, account, akURIs)
	if err != nil {
		return false, err
	}

	return validatePermTree(tree, true)
}

func CheckContractMethodPerm(aclMgr base.AclManager, akURIs []string,
	contractName, methodName string) (bool, error) {

	// aks and signs could have zero length for permission rule Null
	if aclMgr == nil {
		return false, fmt.Errorf("Invalid Param, aclMgr=%v", aclMgr)
	}

	// build perm tree
	tree, err := ptree.BuildMethodPermTree(aclMgr, contractName, methodName, akURIs)
	if err != nil {
		return false, err
	}

	// validate perm tree
	return validatePermTree(tree, false)
}

func validatePermTree(root *ptree.PermNode, isAccount bool) (bool, error) {
	if root == nil {
		return false, errors.New("Root is null")
	}

	// get BFS list of perm tree
	plist, err := ptree.GetPermTreeList(root)
	if err != nil {
		return false, err
	}
	size := len(plist)
	vf := &rule.ACLValidatorFactory{}

	// reverse travel the perm tree
	for i := size - 1; i >= 0; i-- {
		node := plist[i]
		t, isValid := ParseAddressType(node.Name)
		// 0 means AK, 1 means Account, otherwise invalid
		if !isValid {
			return false, errors.New("Invalid account/ak name")
		}

		// for non-account perm tree, the root node is not account name
		if i == 0 && !isAccount {
			t = AddressAccount
		}

		checkResult := false
		if t == AddressAK {
			// current node is AK, signature should be validated before
			checkResult = true
		} else if t == AddressAccount {
			// current node is Account, so validation using ACLValidator
			if node.ACL == nil {
				// empty ACL means everyone could pass ACL validation
				checkResult = true
			} else {
				if node.ACL.Pm == nil {
					return false, errors.New("Acl has empty Pm field")
				}

				// get ACLValidator by ACL type
				validator, err := vf.GetACLValidator(node.ACL.Pm.Rule)
				if err != nil {
					return false, err
				}
				checkResult, err = validator.Validate(node)
				if err != nil {
					return false, err
				}
			}
		}

		// set validation status
		if checkResult {
			node.Status = ptree.Success
		} else {
			node.Status = ptree.Failed
		}
	}
	return root.Status == ptree.Success, nil
}

// ExtractAkFromAuthRequire extracts required AK from auth requirement
// return AK in `Account/AK`
func ExtractAkFromAuthRequire(authRequire string) string {
	return ExtractAddrFromAkURI(authRequire)
}

// ExtractAddrFromAkURI extracts target address from input
// for AK, return AK itself
// for Account, return Account itself
// for auth requirement `Account/AK`， return AK
func ExtractAddrFromAkURI(akURI string) string {
	ids := strings.Split(akURI, "/") // len(ids) must be > 1, see strings.Split()
	return ids[len(ids)-1]
}

func VerifySign(ak string, si *pb.SignatureInfo, data []byte) (bool, error) {
	pk := []byte(si.PublicKey)
	xcc, err := client.CreateCryptoClientFromJSONPublicKey(pk)
	if err != nil {
		return false, err
	}

	ecdsaKey, err := xcc.GetEcdsaPublicKeyFromJsonStr(string(pk))
	if err != nil {
		return false, err
	}

	isMatch, _ := xcc.VerifyAddressUsingPublicKey(ak, ecdsaKey)
	if !isMatch {
		return false, errors.New("address and public key not match")
	}

	return xcc.VerifyECDSA(ecdsaKey, si.Sign, data)
}

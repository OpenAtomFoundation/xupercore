package utils

import (
	"errors"
	"fmt"
	"strings"

	"github.com/xuperchain/xupercore/kernel/permission/acl/ptree"
	"github.com/xuperchain/xupercore/kernel/permission/acl/rule"
	crypto_client "github.com/xuperchain/xupercore/lib/crypto/client"
	pb "github.com/xuperchain/xupercore/protos"

	"github.com/xuperchain/xupercore/kernel/permission/acl/base"
)

func IdentifyAK(akuri string, sign *pb.SignatureInfo, msg []byte) (bool, error) {
	if sign == nil {
		return false, errors.New("sign is nil")
	}
	akpath := SplitAccountURI(akuri)
	if len(akpath) < 1 {
		return false, errors.New("Invalid address")
	}
	ak := akpath[len(akpath)-1]
	return VerifySign(ak, sign, msg)
}

func IdentifyAccount(aclMgr base.AclManager, account string, aksuri []string) (bool, error) {
	// aks and signs could have zero length for permission rule Null
	if aclMgr == nil {
		return false, fmt.Errorf("Invalid Param, aclMgr=%v", aclMgr)
	}

	// build perm tree
	pnode, err := ptree.BuildAccountPermTree(aclMgr, account, aksuri)
	if err != nil {
		return false, err
	}

	return validatePermTree(pnode, true)
}

func CheckContractMethodPerm(aclMgr base.AclManager, aksuri []string,
	contractName, methodName string) (bool, error) {

	// aks and signs could have zero length for permission rule Null
	if aclMgr == nil {
		return false, fmt.Errorf("Invalid Param, aclMgr=%v", aclMgr)
	}

	// build perm tree
	pnode, err := ptree.BuildMethodPermTree(aclMgr, contractName, methodName, aksuri)
	if err != nil {
		return false, err
	}

	// validate perm tree
	return validatePermTree(pnode, false)
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
	listlen := len(plist)
	vf := &rule.ACLValidatorFactory{}

	// reverse travel the perm tree
	for i := listlen - 1; i >= 0; i-- {
		pnode := plist[i]
		nameCheck := IsAccount(pnode.Name)
		// 0 means AK, 1 means Account, otherwise invalid
		if nameCheck < 0 || nameCheck > 1 {
			return false, errors.New("Invalid account/ak name")
		}

		// for non-account perm tree, the root node is not account name
		if i == 0 && !isAccount {
			nameCheck = 1
		}

		checkResult := false
		if nameCheck == 0 {
			// current node is AK, signature should be validated before
			checkResult = true
		} else if nameCheck == 1 {
			// current node is Account, so validation using ACLValidator
			if pnode.ACL == nil {
				// empty ACL means everyone could pass ACL validation
				checkResult = true
			} else {
				if pnode.ACL.Pm == nil {
					return false, errors.New("Acl has empty Pm field")
				}

				// get ACLValidator by ACL type
				validator, err := vf.GetACLValidator(pnode.ACL.Pm.Rule)
				if err != nil {
					return false, err
				}
				checkResult, err = validator.Validate(pnode)
				if err != nil {
					return false, err
				}
			}
		}

		// set validation status
		if checkResult {
			pnode.Status = ptree.Success
		} else {
			pnode.Status = ptree.Failed
		}
	}
	return (root.Status == ptree.Success), nil
}

func SplitAccountURI(akuri string) []string {
	ids := strings.Split(akuri, "/")
	return ids
}

// GetAccountACL return account acl
func GetAccountACL(aclMgr base.AclManager, account string) (*pb.Acl, error) {
	return aclMgr.GetAccountACL(account)
}

// GetContractMethodACL return contract method acl
func GetContractMethodACL(aclMgr base.AclManager, contractName, methodName string) (*pb.Acl, error) {
	return aclMgr.GetContractMethodACL(contractName, methodName)
}

func VerifySign(ak string, si *pb.SignatureInfo, data []byte) (bool, error) {
	bytespk := []byte(si.PublicKey)
	xcc, err := crypto_client.CreateCryptoClientFromJSONPublicKey(bytespk)
	if err != nil {
		return false, err
	}

	ecdsaKey, err := xcc.GetEcdsaPublicKeyFromJsonStr(string(bytespk[:]))
	if err != nil {
		return false, err
	}

	isMatch, _ := xcc.VerifyAddressUsingPublicKey(ak, ecdsaKey)
	if !isMatch {
		return false, errors.New("address and public key not match")
	}

	return xcc.VerifyECDSA(ecdsaKey, si.Sign, data)
}

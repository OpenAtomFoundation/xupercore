package state

import (
	"bytes"
	"crypto/ecdsa"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"strings"
	"sync"

	"github.com/xuperchain/xupercore/bcs/ledger/xledger/state/utxo"
	"github.com/xuperchain/xupercore/bcs/ledger/xledger/state/utxo/txhash"
	"github.com/xuperchain/xupercore/bcs/ledger/xledger/state/xmodel"
	txn "github.com/xuperchain/xupercore/bcs/ledger/xledger/tx"
	pb "github.com/xuperchain/xupercore/bcs/ledger/xledger/xldgpb"
	"github.com/xuperchain/xupercore/kernel/contract"
	"github.com/xuperchain/xupercore/kernel/contract/sandbox"
	kledger "github.com/xuperchain/xupercore/kernel/ledger"
	aclu "github.com/xuperchain/xupercore/kernel/permission/acl/utils"
	"github.com/xuperchain/xupercore/lib/crypto/client"
	"github.com/xuperchain/xupercore/protos"

	"github.com/golang/protobuf/proto"
)

// ImmediateVerifyTx verify tx Immediately
// Transaction verification workflow:
//   1. verify transaction ID is the same with data hash
//   2. verify all signatures of initiator and auth requires
//   3. verify the utxo input, there are three kinds of input validation
//		1). PKI technology for transferring from address
//		2). Account ACL for transferring from account
//		3). Contract logic transferring from contract
//   4. verify the contract requests' permission
//   5. verify the permission of contract RWSet (WriteSet could including unauthorized data change)
//   6. run contract requests and verify if the RWSet result is the same with preExed RWSet (heavy
//      operation, keep it at last)
func (t *State) ImmediateVerifyTx(tx *pb.Transaction, isRootTx bool) (bool, error) {
	// Pre processing of tx data
	if !isRootTx && tx.Version == RootTxVersion {
		return false, ErrVersionInvalid
	}
	if tx.Version > BetaTxVersion || tx.Version < RootTxVersion {
		return false, ErrVersionInvalid
	}
	// autogen tx should not run ImmediateVerifyTx, this could be a fake tx
	if tx.Autogen {
		return false, ErrInvalidAutogenTx
	}
	MaxTxSizePerBlock, MaxTxSizePerBlockErr := t.MaxTxSizePerBlock()
	if MaxTxSizePerBlockErr != nil {
		return false, MaxTxSizePerBlockErr
	}
	if proto.Size(tx) > MaxTxSizePerBlock {
		t.log.Warn("tx too large, should not be greater than half of max blocksize", "size", proto.Size(tx))
		return false, ErrTxTooLarge
	}

	// Start transaction verification workflow
	if tx.Version > RootTxVersion {
		// verify txid
		txid, err := txhash.MakeTransactionID(tx)
		if err != nil {
			t.log.Warn("ImmediateVerifyTx: call MakeTransactionID failed", "error", err)
			return false, err
		}
		if bytes.Compare(tx.Txid, txid) != 0 {
			t.log.Warn("ImmediateVerifyTx: txid not match", "tx.Txid", tx.Txid, "txid", txid)
			return false, fmt.Errorf("Txid verify failed")
		}

		// get digestHash
		digestHash, err := txhash.MakeTxDigestHash(tx)
		if err != nil {
			t.log.Warn("ImmediateVerifyTx: call MakeTxDigestHash failed", "error", err)
			return false, err
		}

		// verify signatures
		ok, verifiedID, err := t.verifySignatures(tx, digestHash)
		if !ok {
			t.log.Warn("ImmediateVerifyTx: verifySignatures failed", "error", err)
			return ok, ErrInvalidSignature
		}

		// get all authenticated users
		authUsers := t.removeDuplicateUser(tx.GetInitiator(), tx.GetAuthRequire())

		// veify tx UTXO input permission (Account ACL)
		ok, err = t.verifyUTXOPermission(tx, verifiedID)
		if !ok {
			t.log.Warn("ImmediateVerifyTx: verifyUTXOPermission failed", "error", err)
			return ok, ErrACLNotEnough
		}

		// verify contract requests' permission using ACL
		ok, err = t.verifyContractPermission(tx, authUsers)
		if !ok {
			t.log.Warn("ImmediateVerifyTx: verifyContractPermission failed", "error", err)
			return ok, ErrACLNotEnough
		}

		// verify amount of transfer within contract
		ok, err = t.verifyContractTxAmount(tx)
		if !ok {
			t.log.Warn("ImmediateVerifyTx: verifyContractTxAmount failed", "error", err)
			return ok, ErrContractTxAmout
		}

		// verify the permission of RWSet using ACL
		ok, err = t.verifyRWSetPermission(tx, verifiedID)
		if !ok {
			t.log.Warn("ImmediateVerifyTx: verifyRWSetPermission failed", "error", err)
			return ok, ErrACLNotEnough
		}
		// verify RWSet(run contracts and compare RWSet)
		ok, err = t.verifyTxRWSets(tx)
		if err != nil {
			t.log.Warn("ImmediateVerifyTx: verifyTxRWSets failed", "error", err)
			// reset error message
			if strings.HasPrefix(err.Error(), "Gas not enough") {
				err = ErrGasNotEnough
			} else {
				err = ErrRWSetInvalid
			}
			return ok, err
		}
		if !ok {
			// always return RWSet Invalid Error if verification not passed
			return ok, ErrRWSetInvalid
		}
	}
	return true, nil
}

// ImmediateVerifyTx verify auto tx Immediately
// Transaction verification workflow:
//	 0. 其实可以直接判断二者的txid，相同，则包括读写集在内的内容都相同
//   1. verify transaction ID is the same with data hash
//   2. run contract requests and verify if the RWSet result is the same with preExed RWSet (heavy
//      operation, keep it at last)
func (t *State) ImmediateVerifyAutoTx(blockHeight int64, tx *pb.Transaction, isRootTx bool) (bool, error) {
	// 获取该区块触发的定时交易
	autoTx, genErr := t.GetTimerTx(blockHeight)
	if genErr != nil || autoTx == nil {
		t.log.Warn("get timer tasks failed", "err", genErr)
		return false, genErr
	}
	if len(autoTx.TxOutputsExt) == 0 {
		return false, fmt.Errorf("get timer tasks failed, no tx outputs ext")
	}

	// Pre processing of tx data
	if !isRootTx && tx.Version == RootTxVersion {
		return false, ErrVersionInvalid
	}
	if tx.Version > BetaTxVersion || tx.Version < RootTxVersion {
		return false, ErrVersionInvalid
	}
	MaxTxSizePerBlock, MaxTxSizePerBlockErr := t.MaxTxSizePerBlock()
	if MaxTxSizePerBlockErr != nil {
		return false, MaxTxSizePerBlockErr
	}
	if proto.Size(tx) > MaxTxSizePerBlock {
		t.log.Warn("tx too large, should not be greater than half of max blocksize", "size", proto.Size(tx))
		return false, ErrTxTooLarge
	}

	// Start transaction verification workflow
	if tx.Version > RootTxVersion {
		// verify txid
		txid, err := txhash.MakeTransactionID(tx)
		if err != nil {
			t.log.Warn("ImmediateVerifyTx: call MakeTransactionID failed", "error", err)
			return false, err
		}
		if bytes.Compare(tx.Txid, txid) != 0 {
			t.log.Warn("ImmediateVerifyTx: txid not match", "tx.Txid", tx.Txid, "txid", txid)
			return false, fmt.Errorf("Txid verify failed")
		}

		// verify RWSet(compare RWSet)
		ok, err := t.verifyAutoTxRWSets(tx, autoTx)
		if err != nil {
			t.log.Warn("ImmediateVerifyTx: verifyTxRWSets failed", "error", err)
			// reset error message
			if strings.HasPrefix(err.Error(), "Gas not enough") {
				err = ErrGasNotEnough
			} else {
				err = ErrRWSetInvalid
			}
			return ok, err
		}
		if !ok {
			// always return RWSet Invalid Error if verification not passed
			return ok, ErrRWSetInvalid
		}
	}
	return true, nil
}

// Note that if tx.XuperSign is not nil, the signature verification use XuperSign process
func (t *State) verifySignatures(tx *pb.Transaction, digestHash []byte) (bool, map[string]bool, error) {
	// XuperSign is not empty, use XuperSign verify
	if tx.GetXuperSign() != nil {
		return t.verifyXuperSign(tx, digestHash)
	}

	// Not XuperSign(multisig/rignsign etc.), use old signature process
	verifiedAddr := make(map[string]bool)
	if len(tx.InitiatorSigns) < 1 || len(tx.AuthRequire) != len(tx.AuthRequireSigns) {
		return false, nil, errors.New("invalid signature param")
	}

	// verify initiator
	akType := aclu.IsAccount(tx.Initiator)
	if akType == 0 {
		// check initiator address signature
		ok, err := aclu.IdentifyAK(tx.Initiator, tx.InitiatorSigns[0], digestHash)
		if err != nil || !ok {
			t.log.Warn("verifySignatures failed", "address", tx.Initiator, "error", err)
			return false, nil, err
		}
		verifiedAddr[tx.Initiator] = true
	} else if akType == 1 {
		initiatorAddr := make([]string, 0)
		// check initiator account signatures
		for _, sign := range tx.InitiatorSigns {
			ak, err := t.sctx.Crypt.GetEcdsaPublicKeyFromJsonStr(sign.PublicKey)
			if err != nil {
				t.log.Warn("verifySignatures failed", "address", tx.Initiator, "error", err)
				return false, nil, err
			}
			addr, err := t.sctx.Crypt.GetAddressFromPublicKey(ak)
			if err != nil {
				t.log.Warn("verifySignatures failed", "address", tx.Initiator, "error", err)
				return false, nil, err
			}
			ok, err := aclu.IdentifyAK(addr, sign, digestHash)
			if !ok {
				t.log.Warn("verifySignatures failed", "address", tx.Initiator, "error", err)
				return ok, nil, err
			}
			verifiedAddr[addr] = true
			initiatorAddr = append(initiatorAddr, tx.Initiator+"/"+addr)
		}
		ok, err := aclu.IdentifyAccount(t.sctx.AclMgr, tx.Initiator, initiatorAddr)
		if !ok {
			t.log.Warn("verifySignatures initiator permission check failed",
				"account", tx.Initiator, "error", err)
			return false, nil, err
		}
	} else {
		t.log.Warn("verifySignatures failed, invalid address", "address", tx.Initiator)
		return false, nil, ErrInvalidSignature
	}

	// verify authRequire
	for idx, authReq := range tx.AuthRequire {
		splitRes := strings.Split(authReq, "/")
		addr := splitRes[len(splitRes)-1]
		signInfo := tx.AuthRequireSigns[idx]
		if _, has := verifiedAddr[addr]; has {
			continue
		}
		ok, err := aclu.IdentifyAK(addr, signInfo, digestHash)
		if err != nil || !ok {
			t.log.Warn("verifySignatures failed", "address", addr, "error", err)
			return false, nil, err
		}
		verifiedAddr[addr] = true
	}
	return true, verifiedAddr, nil
}

func (t *State) verifyXuperSign(tx *pb.Transaction, digestHash []byte) (bool, map[string]bool, error) {
	uniqueAddrs := make(map[string]bool)
	// get all addresses
	uniqueAddrs[tx.Initiator] = true
	addrList := make([]string, 0)
	addrList = append(addrList, tx.Initiator)
	for _, authReq := range tx.AuthRequire {
		splitRes := strings.Split(authReq, "/")
		addr := splitRes[len(splitRes)-1]
		if uniqueAddrs[addr] {
			continue
		}
		uniqueAddrs[addr] = true
		addrList = append(addrList, addr)
	}

	// check addresses and public keys
	if len(addrList) != len(tx.GetXuperSign().GetPublicKeys()) {
		return false, nil, errors.New("XuperSign: number of address and public key not match")
	}
	pubkeys := make([]*ecdsa.PublicKey, 0)
	for _, pubJSON := range tx.GetXuperSign().GetPublicKeys() {
		pubkey, err := t.sctx.Crypt.GetEcdsaPublicKeyFromJsonStr(string(pubJSON))
		if err != nil {
			return false, nil, errors.New("XuperSign: found invalid public key")
		}
		pubkeys = append(pubkeys, pubkey)
	}
	for idx, addr := range addrList {
		ok, _ := t.sctx.Crypt.VerifyAddressUsingPublicKey(addr, pubkeys[idx])
		if !ok {
			t.log.Warn("XuperSign: address and public key not match", "addr", addr, "pubkey", pubkeys[idx])
			return false, nil, errors.New("XuperSign: address and public key not match")
		}
	}
	ok, err := t.sctx.Crypt.VerifyXuperSignature(pubkeys, tx.GetXuperSign().GetSignature(), digestHash)
	if err != nil || !ok {
		t.log.Warn("XuperSign: signature verify failed", "error", err)
		return false, nil, errors.New("XuperSign: address and public key not match")
	}
	return true, uniqueAddrs, nil
}

// verify utxo inputs, there are three kinds of input validation
//	1). PKI technology for transferring from address
//	2). Account ACL for transferring from account
//	3). Contract logic transferring from contract
func (t *State) verifyUTXOPermission(tx *pb.Transaction, verifiedID map[string]bool) (bool, error) {
	// verify tx input
	conUtxoInputs, err := xmodel.ParseContractUtxoInputs(tx)
	if err != nil {
		t.log.Warn("verifyUTXOPermission error, parseContractUtxo ")
		return false, ErrParseContractUtxos
	}
	conUtxoInputsMap := map[string]bool{}
	for _, conUtxoInput := range conUtxoInputs {
		addr := conUtxoInput.GetFromAddr()
		txid := conUtxoInput.GetRefTxid()
		offset := conUtxoInput.GetRefOffset()
		utxoKey := utxo.GenUtxoKey(addr, txid, offset)
		conUtxoInputsMap[utxoKey] = true
	}

	for _, txInput := range tx.TxInputs {
		// if transfer from contract
		addr := txInput.GetFromAddr()
		txid := txInput.GetRefTxid()
		offset := txInput.GetRefOffset()
		utxoKey := utxo.GenUtxoKey(addr, txid, offset)
		if conUtxoInputsMap[utxoKey] {
			// this utxo transfer from contract, will verify in rwset verify
			continue
		}

		name := string(txInput.FromAddr)
		if verifiedID[name] {
			// this ID(either AK or Account) is verified before
			continue
		}
		akType := aclu.IsAccount(name)
		if akType == 1 {
			// Identify account
			acl, err := t.queryAccountACL(name)
			if err != nil || acl == nil {
				// valid account should have ACL info, so this account might not exsit
				t.log.Warn("verifyUTXOPermission error, account might not exist", "account", name, "error", err)
				return false, ErrInvalidAccount
			}
			if ok, err := aclu.IdentifyAccount(t.sctx.AclMgr, string(name), tx.AuthRequire); !ok {
				t.log.Warn("verifyUTXOPermission error, failed to IdentifyAccount", "error", err)
				return false, ErrACLNotEnough
			}
		} else if akType == 0 {
			// Identify address failed, if address not in verifiedID then it must have no signature
			t.log.Warn("verifyUTXOPermission error, address has no signature", "address", name)
			return false, ErrInvalidSignature
		} else {
			t.log.Warn("verifyUTXOPermission error, Invalid account/address name", "name", name)
			return false, ErrInvalidAccount
		}
		verifiedID[name] = true
	}

	return true, nil
}

// verifyContractOwnerPermission check if the transaction has the permission of a contract owner.
// this usually happens in account management operations.
func (t *State) verifyContractOwnerPermission(contractName string, tx *pb.Transaction,
	verifiedID map[string]bool) (bool, error) {
	versionData, confirmed, err := t.xmodel.GetWithTxStatus(aclu.GetContract2AccountBucket(), []byte(contractName))
	if err != nil || versionData == nil {
		return false, err
	}
	pureData := versionData.GetPureData()
	if pureData == nil || confirmed == false {
		return false, errors.New("pure data is nil or unconfirmed")
	}
	accountName := string(pureData.GetValue())
	if verifiedID[accountName] {
		return true, nil
	}
	ok, err := aclu.IdentifyAccount(t.sctx.AclMgr, accountName, tx.AuthRequire)
	if err == nil && ok {
		verifiedID[accountName] = true
	}
	return ok, err
}

// verifyRWSetPermission verify the permission of RWSet using ACL
func (t *State) verifyRWSetPermission(tx *pb.Transaction, verifiedID map[string]bool) (bool, error) {
	req := tx.GetContractRequests()
	// if not contract, pass directly
	if req == nil {
		return true, nil
	}
	writeSet := []*kledger.PureData{}
	for _, txOut := range tx.TxOutputsExt {
		writeSet = append(writeSet, &kledger.PureData{Bucket: txOut.Bucket, Key: txOut.Key, Value: txOut.Value})
	}
	for _, ele := range writeSet {
		bucket := ele.GetBucket()
		key := ele.GetKey()
		switch bucket {
		case aclu.GetAccountBucket():
			// modified account data, need to check if the tx has the permission of account
			accountName := string(key)
			if verifiedID[accountName] {
				continue
			}
			ok, err := aclu.IdentifyAccount(t.sctx.AclMgr, accountName, tx.AuthRequire)
			if !ok {
				t.log.Warn("verifyRWSetPermission check account bucket failed",
					"account", accountName, "AuthRequire ", tx.AuthRequire, "error", err)
				return ok, err
			}
			verifiedID[accountName] = true
		case aclu.GetContractBucket():
			// modified contact data, need to check if the tx has the permission of contract owner
			separator := aclu.GetACLSeparator()
			idx := bytes.Index(key, []byte(separator))
			if idx < 0 {
				return false, errors.New("invalid raw key")
			}
			contractName := string(key[:idx])
			ok, contractErr := t.verifyContractOwnerPermission(contractName, tx, verifiedID)
			if !ok {
				t.log.Warn("verifyRWSetPermission check contract bucket failed",
					"contract", contractName, "AuthRequire ", tx.AuthRequire, "error", contractErr)
				return ok, contractErr
			}
		case aclu.GetContract2AccountBucket():
			// modified contract/account mapping
			// need to check if the tx has the permission of target account
			accountValue := ele.GetValue()
			if accountValue == nil {
				return false, errors.New("account name is empty")
			}
			accountName := string(accountValue)
			if verifiedID[accountName] {
				continue
			}
			ok, accountErr := aclu.IdentifyAccount(t.sctx.AclMgr, accountName, tx.AuthRequire)
			if !ok {
				t.log.Warn("verifyRWSetPermission check contract2account bucket failed",
					"account", accountName, "AuthRequire ", tx.AuthRequire, "error", accountErr)
				return ok, accountErr
			}
			verifiedID[accountName] = true
		}
	}
	return true, nil
}

// verifyContractValid verify the permission of contract requests using ACL
func (t *State) verifyContractPermission(tx *pb.Transaction, allUsers []string) (bool, error) {
	req := tx.GetContractRequests()
	if req == nil {
		// if no contract requests, no need to verify
		return true, nil
	}

	for i := 0; i < len(req); i++ {
		tmpReq := req[i]
		contractName := tmpReq.GetContractName()
		methodName := tmpReq.GetMethodName()

		ok, err := aclu.CheckContractMethodPerm(t.sctx.AclMgr, allUsers, contractName, methodName)
		if err != nil || !ok {
			t.log.Warn("verify contract method ACL failed ", "contract", contractName, "method",
				methodName, "error", err)
			return ok, ErrACLNotEnough
		}
	}
	return true, nil
}

func getGasLimitFromTx(tx *pb.Transaction) (int64, error) {
	for _, output := range tx.GetTxOutputs() {
		if string(output.GetToAddr()) != "$" {
			continue
		}
		gasLimit := big.NewInt(0).SetBytes(output.GetAmount()).Int64()
		if gasLimit <= 0 {
			return 0, fmt.Errorf("bad gas limit %d", gasLimit)
		}
		return gasLimit, nil
	}
	return 0, nil
}

// verifyContractTxAmount verify
func (t *State) verifyContractTxAmount(tx *pb.Transaction) (bool, error) {
	amountOut := new(big.Int).SetInt64(0)
	req := tx.GetContractRequests()
	contractName, amountCon, err := txn.ParseContractTransferRequest(req)
	if err != nil {
		return false, err
	}
	for _, txOutput := range tx.GetTxOutputs() {
		if string(txOutput.GetToAddr()) == contractName {
			tmpAmount := new(big.Int).SetBytes(txOutput.GetAmount())
			amountOut.Add(tmpAmount, amountOut)
		}
	}

	if amountOut.Cmp(amountCon) != 0 {
		return false, ErrContractTxAmout
	}
	return true, nil
}

// verifyTxRWSets verify tx read sets and write sets
func (t *State) verifyTxRWSets(tx *pb.Transaction) (bool, error) {
	if t.VerifyReservedWhitelist(tx) {
		t.log.Info("verifyReservedWhitelist true", "txid", fmt.Sprintf("%x", tx.GetTxid()))
		return true, nil
	}

	req := tx.GetContractRequests()
	reservedRequests, err := t.GetReservedContractRequests(tx.GetContractRequests(), false)
	if err != nil {
		t.log.Error("getReservedContractRequests error", "error", err.Error())
		return false, err
	}

	if !t.VerifyReservedContractRequests(reservedRequests, req) {
		t.log.Error("verifyReservedContractRequests error", "reservedRequests", reservedRequests, "req", req)
		return false, fmt.Errorf("verify reservedContracts error")
	}

	if req == nil {
		if tx.GetTxInputsExt() != nil || tx.GetTxOutputsExt() != nil {
			t.log.Error("verifyTxRWSets error", "error", ErrInvalidTxExt.Error())
			return false, ErrInvalidTxExt
		}
		return true, nil
	}

	rset, wset, err := t.GenRWSetFromTx(tx)
	if err != nil {
		return false, nil
	}
	rwSet := &contract.RWSet{
		RSet: rset,
		WSet: wset,
	}

	reader := sandbox.XMReaderFromRWSet(rwSet)
	sandBoxConfig := &contract.SandboxConfig{
		XMReader: reader,
	}
	sandBox, err := t.sctx.ContractMgr.NewStateSandbox(sandBoxConfig)
	if err != nil {
		return false, nil
	}

	transContractName, transAmount, err := txn.ParseContractTransferRequest(req)
	if err != nil {
		return false, err
	}

	contextConfig := &contract.ContextConfig{
		State:       sandBox,
		Initiator:   tx.GetInitiator(),
		AuthRequire: tx.GetAuthRequire(),
	}
	gasLimit, err := getGasLimitFromTx(tx)
	if err != nil {
		return false, err
	}
	t.log.Trace("get gas limit from tx", "gasLimit", gasLimit, "txid", hex.EncodeToString(tx.Txid))

	// get gas rate to utxo
	gasPrice := t.meta.Meta.GetGasPrice()

	for i, tmpReq := range tx.GetContractRequests() {
		limits := contract.FromPbLimits(tmpReq.GetResourceLimits())
		if i >= len(reservedRequests) {
			gasLimit -= limits.TotalGas(gasPrice)
		}
		if gasLimit < 0 {
			t.log.Error("virifyTxRWSets error:out of gas", "contractName", tmpReq.GetContractName(),
				"txid", hex.EncodeToString(tx.Txid))
			return false, errors.New("out of gas")
		}
		contextConfig.ResourceLimits = limits
		contextConfig.Module = tmpReq.ModuleName
		contextConfig.ContractName = tmpReq.GetContractName()
		contextConfig.Module = tmpReq.GetModuleName()
		if transContractName == tmpReq.GetContractName() {
			contextConfig.TransferAmount = transAmount.String()
		} else {
			contextConfig.TransferAmount = ""
		}

		ctx, err := t.sctx.ContractMgr.NewContext(contextConfig)
		if err != nil {
			t.log.Error("verifyTxRWSets NewContext error", "err", err, "contractName", tmpReq.GetContractName())
			if i < len(reservedRequests) && (err.Error() == "leveldb: not found" || strings.HasSuffix(err.Error(), "not found")) {
				continue
			}
			return false, err
		}

		ctxResponse, ctxErr := ctx.Invoke(tmpReq.MethodName, tmpReq.Args)
		if ctxErr != nil {
			ctx.Release()
			t.log.Error("verifyTxRWSets Invoke error", "error", ctxErr, "contractName", tmpReq.GetContractName())
			return false, ctxErr
		}
		// 判断合约调用的返回码
		if ctxResponse.Status >= 400 && i < len(reservedRequests) {
			ctx.Release()
			t.log.Error("verifyTxRWSets Invoke error", "status", ctxResponse.Status, "contractName", tmpReq.GetContractName())
			return false, errors.New(ctxResponse.Message)
		}

		ctx.Release()
	}

	err = sandBox.Flush()
	if err != nil {
		return false, err
	}

	RWSet := sandBox.RWSet()
	t.log.Trace("verifyTxRWSets", "env.output", wset, "writeSet", RWSet.WSet)
	ok := xmodel.Equal(wset, RWSet.WSet)
	if !ok {
		return false, fmt.Errorf("write set not equal")
	}

	return true, nil
}

// verifyAutoTxRWSets verify auto tx read sets and write sets
func (t *State) verifyAutoTxRWSets(tx, autoTx *pb.Transaction) (bool, error) {
	txRsets := tx.GetTxInputsExt()
	autoRsets := autoTx.GetTxInputsExt()

	txWsets := tx.GetTxOutputsExt()
	autoWsets := autoTx.GetTxOutputsExt()

	t.log.Trace("verifyAutoTxRWSets", "tx", txRsets, "auto", autoRsets)
	t.log.Trace("verifyAutoTxRWSets", "tx", txWsets, "auto", autoWsets)
	// 判断读写集是否相同相等
	txRsetsBytes, err := json.Marshal(txRsets)
	if err != nil {
		return false, err
	}
	autoRsetsBytes, err := json.Marshal(autoRsets)
	if err != nil {
		return false, err
	}
	txWsetsBytes, err := json.Marshal(txWsets)
	if err != nil {
		return false, err
	}
	autoWsetsBytes, err := json.Marshal(autoWsets)
	if err != nil {
		return false, err
	}

	if bytes.Compare(txRsetsBytes, autoRsetsBytes) != 0 {
		return false, fmt.Errorf("read set not equal")
	}
	if bytes.Compare(txWsetsBytes, autoWsetsBytes) != 0 {
		return false, fmt.Errorf("write set not equal")
	}

	return true, nil
}

func (t *State) verifyMarked(tx *pb.Transaction) (bool, bool, error) {
	isRelyOnMarkedTx := false
	if tx.GetModifyBlock() != nil && tx.ModifyBlock.Marked {
		isRelyOnMarkedTx := true
		err := t.verifyMarkedTx(tx)
		if err != nil {
			return false, isRelyOnMarkedTx, err
		}
		return true, isRelyOnMarkedTx, nil
	}
	ok, isRelyOnMarkedTx, err := t.verifyRelyOnMarkedTxs(tx)
	return ok, isRelyOnMarkedTx, err
}

func (t *State) verifyMarkedTx(tx *pb.Transaction) error {
	bytespk := []byte(tx.ModifyBlock.PublicKey)
	xcc, err := client.CreateCryptoClientFromJSONPublicKey(bytespk)
	if err != nil {
		return err
	}
	ecdsaKey, err := xcc.GetEcdsaPublicKeyFromJsonStr(tx.ModifyBlock.PublicKey)
	if err != nil {
		return err
	}
	isMatch, _ := xcc.VerifyAddressUsingPublicKey(t.utxo.ModifyBlockAddr, ecdsaKey)
	if !isMatch {
		return errors.New("address and public key not match")
	}

	bytesign, err := hex.DecodeString(tx.ModifyBlock.Sign)
	if err != nil {
		return errors.New("invalide arg type: sign byte")
	}
	tx.ModifyBlock = &pb.ModifyBlock{}
	digestHash, err := txhash.MakeTxDigestHash(tx)
	if err != nil {
		t.log.Warn("verifyMarkedTx call MakeTxDigestHash failed", "error", err)
		return err
	}
	ok, err := xcc.VerifyECDSA(ecdsaKey, bytesign, digestHash)
	if err != nil || !ok {
		t.log.Warn("verifyMarkedTx validateUpdateBlockChainData verifySignatures failed")
		return err
	}
	return nil
}

// verifyRelyOnMarkedTxs
// bool Pass verification or not
// bool isRelyOnMarkedTx
// err
func (t *State) verifyRelyOnMarkedTxs(tx *pb.Transaction) (bool, bool, error) {
	isRelyOnMarkedTx := false
	for _, txInput := range tx.GetTxInputs() {
		reftxid := txInput.RefTxid
		if string(reftxid) == "" {
			continue
		}
		ok, isRelyOnMarkedTx, err := t.checkRelyOnMarkedTxid(reftxid, tx.Blockid)
		if err != nil || !ok {
			return ok, isRelyOnMarkedTx, err
		}
	}
	for _, txIn := range tx.GetTxInputsExt() {
		reftxid := txIn.RefTxid
		if string(reftxid) == "" {
			continue
		}
		ok, isRelyOnMarkedTx, err := t.checkRelyOnMarkedTxid(reftxid, tx.Blockid)
		if !ok || err != nil {
			return ok, isRelyOnMarkedTx, err
		}
	}

	return true, isRelyOnMarkedTx, nil
}

// bool Pass verification or not
// bool isRely
// err
func (t *State) checkRelyOnMarkedTxid(reftxid []byte, blockid []byte) (bool, bool, error) {
	isRely := false
	reftx, err := t.sctx.Ledger.QueryTransaction(reftxid)
	if err != nil {
		return true, isRely, nil
	}
	if reftx.GetModifyBlock() != nil && reftx.ModifyBlock.Marked {
		isRely = true
		if string(blockid) != "" {
			ib, err := t.sctx.Ledger.QueryBlock(blockid)
			if err != nil {
				return false, isRely, err
			}
			if ib.Height <= reftx.ModifyBlock.EffectiveHeight {
				return true, isRely, nil
			}
		}
		return false, isRely, nil
	}
	return true, isRely, nil
}

// removeDuplicateUser combine initiator and auth_require and remove duplicate users
func (t *State) removeDuplicateUser(initiator string, authRequire []string) []string {
	dupCheck := make(map[string]bool)
	finalUsers := make([]string, 0)
	if aclu.IsAccount(initiator) == 0 {
		finalUsers = append(finalUsers, initiator)
		dupCheck[initiator] = true
	}
	for _, user := range authRequire {
		if dupCheck[user] {
			continue
		}
		finalUsers = append(finalUsers, user)
		dupCheck[user] = true
	}
	return finalUsers
}

func (t *State) GetAccountAddresses(accountName string) ([]string, error) {
	return t.sctx.AclMgr.GetAccountAddresses(accountName)
}

// VerifyContractPermission implement Contract ChainCore, used to verify contract permission while contract running
func (t *State) VerifyContractPermission(initiator string, authRequire []string, contractName, methodName string) (bool, error) {
	allUsers := t.removeDuplicateUser(initiator, authRequire)
	return aclu.CheckContractMethodPerm(t.sctx.AclMgr, allUsers, contractName, methodName)
}

// VerifyContractOwnerPermission implement Contract ChainCore, used to verify contract ownership permisson
func (t *State) VerifyContractOwnerPermission(contractName string, authRequire []string) error {
	versionData, confirmed, err := t.xmodel.GetWithTxStatus(aclu.GetContract2AccountBucket(), []byte(contractName))
	if err != nil {
		return err
	}
	if !confirmed {
		return errors.New("contract for account not confirmed")
	}
	accountName := string(versionData.GetPureData().GetValue())
	if accountName == "" {
		return errors.New("contract not found")
	}
	ok, err := aclu.IdentifyAccount(t.sctx.AclMgr, accountName, authRequire)
	if err != nil {
		return err
	}
	if !ok {
		return errors.New("verify contract owner permission failed")
	}
	return nil
}

func (t *State) MaxTxSizePerBlock() (int, error) {
	maxBlkSize := t.GetMaxBlockSize()
	return int(float64(maxBlkSize) * TxSizePercent), nil
}

func (t *State) GetMaxBlockSize() int64 {
	return t.meta.GetMaxBlockSize()
}

func (t *State) queryAccountACL(accountName string) (*protos.Acl, error) {
	if t.sctx.AclMgr == nil {
		return nil, errors.New("acl manager is nil")
	}
	return t.sctx.AclMgr.GetAccountACL(accountName)
}

func (t *State) verifyBlockTxs(block *pb.InternalBlock, isRootTx bool, unconfirmToConfirm map[string]bool) error {
	var err error
	var once sync.Once
	wg := sync.WaitGroup{}
	dags := txn.SplitToDags(block)
	for _, txs := range dags {
		wg.Add(1)
		go func(txs []*pb.Transaction) {
			defer wg.Done()
			verifyErr := t.verifyDAGTxs(block.Height, txs, isRootTx, unconfirmToConfirm)
			onceBody := func() {
				err = verifyErr
			}
			// err 只被赋值一次
			if verifyErr != nil {
				once.Do(onceBody)
			}
		}(txs)
	}
	wg.Wait()
	return err
}

func (t *State) verifyDAGTxs(blockHeight int64, txs []*pb.Transaction, isRootTx bool, unconfirmToConfirm map[string]bool) error {
	for _, tx := range txs {
		if tx == nil {
			return errors.New("verifyTx error, tx is nil")
		}
		txid := string(tx.GetTxid())
		if unconfirmToConfirm[txid] == false {
			if t.verifyAutogenTxValid(tx) {
				// 校验auto tx
				if ok, err := t.ImmediateVerifyAutoTx(blockHeight, tx, isRootTx); !ok {
					t.log.Warn("dotx failed to ImmediateVerifyAutoTx", "txid", fmt.Sprintf("%x", tx.Txid), "err", err)
					return errors.New("dotx failed to ImmediateVerifyAutoTx error")
				}
			}
			if !tx.Autogen && !tx.Coinbase {
				// 校验用户交易
				if ok, err := t.ImmediateVerifyTx(tx, isRootTx); !ok {
					t.log.Warn("dotx failed to ImmediateVerifyTx", "txid", fmt.Sprintf("%x", tx.Txid), "err", err)
					ok, isRelyOnMarkedTx, err := t.verifyMarked(tx)
					if isRelyOnMarkedTx {
						if !ok || err != nil {
							t.log.Warn("tx verification failed because it is blocked tx", "err", err)
						} else {
							t.log.Trace("blocked tx verification succeed")
						}
						return err
					}
					return errors.New("dotx failed to ImmediateVerifyTx error")
				}
			}
		}
	}

	return nil
}

// verifyAutogenTxValid verify if a autogen tx is valid, return true if tx is valid.
func (t *State) verifyAutogenTxValid(tx *pb.Transaction) bool {
	if !tx.Autogen {
		return false
	}

	if len(tx.TxInputsExt) == 0 && len(tx.TxOutputsExt) == 0 {
		// autogen tx must have  tx inputs/outputs extend
		return false
	}

	return true
}

func (t *State) GenRWSetFromTx(tx *pb.Transaction) ([]*kledger.VersionedData, []*kledger.PureData, error) {
	inputs := []*kledger.VersionedData{}
	outputs := []*kledger.PureData{}
	t.log.Trace("GenRWSetFromTx", "tx.TxInputsExt", tx.TxInputsExt, "tx.TxOutputsExt", tx.TxOutputsExt)
	for _, txIn := range tx.TxInputsExt {
		var verData *kledger.VersionedData
		var err error
		if len(tx.Blockid) == 0 {
			verData, err = t.xmodel.Get(txIn.Bucket, txIn.Key)
		} else {
			verData, err = t.xmodel.GetFromLedger(txIn)
		}
		if err != nil {
			return nil, nil, err
		}
		t.log.Trace("prepareEnv", "verData", verData, "txIn", txIn)
		if xmodel.GetVersion(verData) != GetVersion(txIn) {
			err := fmt.Errorf("prepareEnv fail, key:%s, inputs version is not valid: %s != %s", string(verData.PureData.Key), xmodel.GetVersion(verData), GetVersion(txIn))
			return nil, nil, err
		}
		inputs = append(inputs, verData)
	}
	for _, txOut := range tx.TxOutputsExt {
		outputs = append(outputs, &kledger.PureData{Bucket: txOut.Bucket, Key: txOut.Key, Value: txOut.Value})
	}
	return inputs, outputs, nil
}

func GetVersion(txIn *protos.TxInputExt) string {
	if txIn.RefTxid == nil {
		return ""
	}
	return fmt.Sprintf("%x_%d", txIn.RefTxid, txIn.RefOffset)
}

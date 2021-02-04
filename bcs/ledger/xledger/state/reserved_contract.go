package state

import (
	"bytes"
	"html/template"

	pb "github.com/xuperchain/xupercore/bcs/ledger/xledger/xldgpb"
	"github.com/xuperchain/xupercore/protos"
)

// reservedArgs used to get contractnames from InvokeRPCRequest
type reservedArgs struct {
	ContractNames string
}

func genArgs(req []*protos.InvokeRequest) *reservedArgs {
	ra := &reservedArgs{}
	for i, v := range req {
		ra.ContractNames += v.GetContractName()
		if i < len(req)-1 {
			ra.ContractNames += ","
		}
	}
	return ra
}

// It will check whether the transaction in reserved whitelist
// if the config of chain contains reserved contracts
// but the transaction does not contains reserved requests.
func (t *State) VerifyReservedWhitelist(tx *pb.Transaction) bool {
	// verify reservedContracts len
	reservedContracts := t.meta.GetReservedContracts()
	if len(reservedContracts) == 0 {
		t.log.Info("verifyReservedWhitelist false reservedReqs is empty")
		return false
	}

	// get white list account
	accountName := t.sctx.Ledger.GetGenesisBlock().GetConfig().GetReservedWhitelistAccount()
	t.log.Trace("verifyReservedWhitelist", "accountName", accountName)
	if accountName == "" {
		t.log.Info("verifyReservedWhitelist false, the chain does not have reserved whitelist", "accountName", accountName)
		return false
	}
	acl, err := t.sctx.AclMgr.GetAccountACL(accountName)
	if err != nil || acl == nil {
		t.log.Info("verifyReservedWhitelist false, get reserved whitelist acl failed",
			"err", err, "acl", acl)
		return false
	}

	// verify storage
	if tx.GetDesc() != nil ||
		tx.GetContractRequests() != nil ||
		tx.GetTxInputsExt() != nil ||
		tx.GetTxOutputsExt() != nil {
		t.log.Info("verifyReservedWhitelist false the storage info should be nil")
		return false
	}

	// verify utxo input
	if len(tx.GetTxInputs()) == 0 && len(tx.GetTxOutputs()) == 0 {
		t.log.Info("verifyReservedWhitelist true the utxo list is nil")
		return true
	}
	fromAddr := string(tx.GetTxInputs()[0].GetFromAddr())
	for _, v := range tx.GetTxInputs() {
		if string(v.GetFromAddr()) != fromAddr {
			t.log.Info("verifyReservedWhitelist false fromAddr should no more than one")
			return false
		}
	}

	// verify utxo output
	toAddrs := make(map[string]bool)
	for _, v := range tx.GetTxOutputs() {
		if bytes.Equal(v.GetToAddr(), []byte(FeePlaceholder)) {
			continue
		}
		toAddrs[string(v.GetToAddr())] = true
		if len(toAddrs) > 2 {
			t.log.Info("verifyReservedWhitelist false toAddrs should no more than two")
			return false
		}
	}

	// verify utxo output whitelist
	for k := range toAddrs {
		if k == fromAddr {
			continue
		}
		if _, ok := acl.GetAksWeight()[k]; !ok {
			t.log.Info("verifyReservedWhitelist false the toAddr should in whitelist acl")
			return false
		}
	}
	return true
}

func (t *State) VerifyReservedContractRequests(reservedReqs, txReqs []*protos.InvokeRequest) bool {
	if len(reservedReqs) > len(txReqs) {
		return false
	}
	for i := 0; i < len(reservedReqs); i++ {
		if (reservedReqs[i].GetModuleName() != txReqs[i].GetModuleName()) || (reservedReqs[i].GetContractName() != txReqs[i].GetContractName()) ||
			(reservedReqs[i].GetMethodName() != txReqs[i].GetMethodName()) {
			return false
		}
		for k, v := range txReqs[i].Args {
			if !bytes.Equal(reservedReqs[i].GetArgs()[k], v) {
				return false
			}
		}
	}
	return true
}

// geReservedContractRequest get reserved contract requests from system params, it doesn't consume gas.
func (t *State) GetReservedContractRequests(req []*protos.InvokeRequest, isPreExec bool) ([]*protos.InvokeRequest, error) {
	MetaReservedContracts := t.meta.GetReservedContracts()
	if MetaReservedContracts == nil {
		return nil, nil
	}
	reservedContractstpl := MetaReservedContracts
	t.log.Info("MetaReservedContracts", "reservedContracts", reservedContractstpl)

	// if all reservedContracts have not been updated, return nil, nil
	ra := &reservedArgs{}
	if isPreExec || len(reservedContractstpl) == 0 {
		ra = genArgs(req)
	} else {
		// req should contrain reservedContracts, so the len of req should no less than reservedContracts
		if len(req) < len(reservedContractstpl) {
			t.log.Warn("req should contain reservedContracts")
			return nil, ErrGetReservedContracts
		} else if len(req) > len(reservedContractstpl) {
			ra = genArgs(req[len(reservedContractstpl):])
		}
	}

	reservedContracts := []*protos.InvokeRequest{}
	for _, rc := range reservedContractstpl {
		rctmp := *rc
		rctmp.Args = make(map[string][]byte)
		for k, v := range rc.GetArgs() {
			buf := new(bytes.Buffer)
			tpl := template.Must(template.New("value").Parse(string(v)))
			tpl.Execute(buf, ra)
			rctmp.Args[k] = buf.Bytes()
		}
		reservedContracts = append(reservedContracts, &rctmp)
	}
	return reservedContracts, nil
}

package tx

import (
	"encoding/hex"
	"errors"
	"fmt"
	"math/big"
	"time"

	"github.com/spf13/cobra"

	"github.com/xuperchain/xupercore/bcs/ledger/xledger/state/utxo/txhash"
	"github.com/xuperchain/xupercore/bcs/ledger/xledger/xldgpb"
	"github.com/xuperchain/xupercore/example/xchain/cmd/client/client"
	"github.com/xuperchain/xupercore/example/xchain/cmd/client/common/global"
	xdef "github.com/xuperchain/xupercore/example/xchain/common/def"
	"github.com/xuperchain/xupercore/example/xchain/common/xchainpb"
	cryptoClient "github.com/xuperchain/xupercore/lib/crypto/client"
	cryptoBase "github.com/xuperchain/xupercore/lib/crypto/client/base"
	"github.com/xuperchain/xupercore/lib/utils"
	"github.com/xuperchain/xupercore/protos"
)

var (
	// ErrInvalidAmount error
	ErrInvalidAmount = errors.New("Invalid amount number")
	// ErrNegativeAmount error
	ErrNegativeAmount = errors.New("Amount in transaction can not be negative number")
	// ErrSelectUtxo error
	ErrSelectUtxo = errors.New("Select utxo error")
)

var (
	CommonTransferDesc = []byte("common transfer transaction")
)

type TransferTxCmd struct {
	global.BaseCmd
	To           string
	Amount       string
	FrozenHeight int64
}

func GetTransferTxCmd() *TransferTxCmd {
	transTxCmdIns := new(TransferTxCmd)

	transTxCmdIns.Cmd = &cobra.Command{
		Use:           "transfer",
		Short:         "start transfer transaction.",
		Example:       xdef.CmdLineName + " tx transfer --to [address] --amount [amount]",
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			return transTxCmdIns.transfer()
		},
	}

	// 设置命令行参数并绑定变量
	transTxCmdIns.Cmd.Flags().StringVarP(&transTxCmdIns.To, "to", "t", "", "to address")
	transTxCmdIns.Cmd.Flags().StringVarP(&transTxCmdIns.Amount, "amount", "a", "", "transfer amount")
	transTxCmdIns.Cmd.Flags().Int64VarP(&transTxCmdIns.FrozenHeight, "frozen", "f", 0, "frozen height of one tx")

	return transTxCmdIns
}

func (t *TransferTxCmd) transfer() error {
	xcli, err := client.NewXchainClient()
	if err != nil {
		fmt.Sprintf("grpc dial failed.err:%v\n", err)
		return fmt.Errorf("grpc dial failed")
	}
	amount, ok := big.NewInt(0).SetString(t.Amount, 10)
	if !ok {
		return ErrInvalidAmount
	}
	bigZero := big.NewInt(0)
	if amount.Cmp(bigZero) < 0 {
		return ErrNegativeAmount
	}

	resp, err := xcli.SelectUtxo(amount)
	if err != nil {
		fmt.Sprintf("select utxo failed.err:%v", err)
		return fmt.Errorf("select utxo failed")
	}
	//构造交易
	tx, err := t.generateTx(resp, amount)
	if err != nil {
		fmt.Sprintf("generate tx failed.err:%v", err)
		return fmt.Errorf("generate tx failed")
	}
	// 提交交易
	_, err = xcli.SubmitTx(tx)
	if err != nil {
		fmt.Sprintf("submit tx failed.err:%v", err)
		return fmt.Errorf("submit tx failed")
	}

	fmt.Println(hex.EncodeToString(tx.Txid))
	return nil
}

func (t *TransferTxCmd) generateTx(utxoRes *xchainpb.SelectUtxoResp, amount *big.Int) (*xldgpb.Transaction, error) {
	addr, err := global.LoadAccount(global.GFlagCrypto, global.GFlagKeys)
	if err != nil {
		return nil, fmt.Errorf("load account info failed.KeyPath:%s Err:%v", global.GFlagKeys, err)
	}
	cryptoClient, err := cryptoClient.CreateCryptoClient(global.GFlagCrypto)
	if err != nil {
		return nil, err
	}

	tx := &xldgpb.Transaction{
		Version:   1,
		Coinbase:  false,
		Desc:      CommonTransferDesc,
		Nonce:     utils.GenNonce(),
		Timestamp: time.Now().UnixNano(),
		Initiator: addr.Address,
	}
	txOutput := &protos.TxOutput{}
	txOutput.ToAddr = []byte(t.To)
	txOutput.Amount = amount.Bytes()
	txOutput.FrozenHeight = t.FrozenHeight
	tx.TxOutputs = append(tx.TxOutputs, txOutput)

	txInputs, deltaOutput, err := t.genTxInputs(utxoRes, amount, addr.Address)
	if err != nil {
		fmt.Sprintf("gen tx.err:%v", err)
		return nil, err
	}
	tx.TxInputs = txInputs
	if deltaOutput != nil {
		tx.TxOutputs = append(tx.TxOutputs, deltaOutput)
	}

	tx.AuthRequire = genAuthRequire(addr.Address)

	// 签名和生成txid
	signTx, err := txhash.ProcessSignTx(cryptoClient, tx, []byte(addr.PrivateKeyStr))
	if err != nil {
		return nil, err
	}
	signInfo := &protos.SignatureInfo{
		PublicKey: addr.PublicKeyStr,
		Sign:      signTx,
	}
	tx.InitiatorSigns = append(tx.InitiatorSigns, signInfo)
	tx.AuthRequireSigns, err = genAuthRequireSigns(cryptoClient, tx, addr.PrivateKeyStr, addr.PublicKeyStr)
	if err != nil {
		return nil, fmt.Errorf("Failed to genAuthRequireSigns %s", err)
	}
	tx.Txid, err = txhash.MakeTransactionID(tx)
	if err != nil {
		return nil, fmt.Errorf("Failed to gen txid %s", err)
	}

	return tx, nil
}

func (t *TransferTxCmd) genTxInputs(utxoRes *xchainpb.SelectUtxoResp, totalNeed *big.Int,
	initAddr string) ([]*protos.TxInput, *protos.TxOutput, error) {
	var txTxInputs []*protos.TxInput
	var txOutput *protos.TxOutput
	for _, utxo := range utxoRes.UtxoList {
		txInput := new(protos.TxInput)
		txInput.RefTxid = utxo.RefTxid
		txInput.RefOffset = utxo.RefOffset
		txInput.FromAddr = utxo.ToAddr
		txInput.Amount = utxo.Amount
		txTxInputs = append(txTxInputs, txInput)
	}
	utxoTotal, ok := big.NewInt(0).SetString(utxoRes.TotalAmount, 10)
	if !ok {
		return nil, nil, ErrSelectUtxo
	}
	// 多出来的utxo再转给自己
	if utxoTotal.Cmp(totalNeed) > 0 {
		delta := utxoTotal.Sub(utxoTotal, totalNeed)
		txOutput = &protos.TxOutput{
			ToAddr: []byte(initAddr),
			Amount: delta.Bytes(),
		}
	}
	return txTxInputs, txOutput, nil
}

func genAuthRequireSigns(cryptoClient cryptoBase.CryptoClient, tx *xldgpb.Transaction, initScrkey, initPubkey string) ([]*protos.SignatureInfo, error) {
	authRequireSigns := []*protos.SignatureInfo{}
	signTx, err := txhash.ProcessSignTx(cryptoClient, tx, []byte(initScrkey))
	if err != nil {
		return nil, err
	}
	signInfo := &protos.SignatureInfo{
		PublicKey: initPubkey,
		Sign:      signTx,
	}
	authRequireSigns = append(authRequireSigns, signInfo)
	return authRequireSigns, nil
}

func genAuthRequire(initAddr string) []string {
	authRequire := []string{}
	authRequire = append(authRequire, initAddr)
	return authRequire
}

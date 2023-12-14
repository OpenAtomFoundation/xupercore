package cmd

import (
	"crypto/ecdsa"
	"encoding/hex"
	"fmt"
	"os"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/spf13/cobra"
)

// NewAccountCommand new account cmd
func NewEthAccountCommand(cli *Cli) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "ethAccount",
		Short: "Operate eth account private and public key: generate|inspect.",
	}
	cmd.AddCommand(NewEthAccountGenerateCommand(cli))
	cmd.AddCommand(NewEthAccountInspectCommand(cli))

	return cmd
}

func init() {
	AddCommand(NewEthAccountCommand)
}

type EthAccountCommand struct {
	cli  *Cli
	cmd  *cobra.Command
	file string
}

func NewEthAccountGenerateCommand(cli *Cli) *cobra.Command {
	b := new(EthAccountCommand)
	b.cli = cli
	b.cmd = &cobra.Command{
		Use:   "generate [account/address]",
		Short: "Generate private key.",
		RunE: func(cmd *cobra.Command, args []string) error {
			return b.newPrivateKey()
		},
	}

	b.addFlags()

	return b.cmd
}

func (b *EthAccountCommand) addFlags() {
	b.cmd.Flags().StringVarP(&b.file, "file", "f", "./data/keys/eth.account", "private key file")
}

func (b *EthAccountCommand) newPrivateKey() error {
	privateKey, err := crypto.GenerateKey()
	if err != nil {
		return err
	}
	privateKeyBytes := crypto.FromECDSA(privateKey)
	hexPrivateKey := hex.EncodeToString(privateKeyBytes)
	publicKey := privateKey.Public().(*ecdsa.PublicKey)
	fmt.Println("Saved private key to file: ", b.file)
	fmt.Printf("Address: %s\n", crypto.PubkeyToAddress(privateKey.PublicKey))
	fmt.Printf("Public key: %s\n", hex.EncodeToString(crypto.FromECDSAPub(&privateKey.PublicKey)))
	fmt.Printf("Compress public key: %#x\n", crypto.CompressPubkey(publicKey))
	return os.WriteFile(b.file, []byte(hexPrivateKey), 0600)
}

func NewEthAccountInspectCommand(cli *Cli) *cobra.Command {
	b := new(EthAccountCommand)
	b.cli = cli
	b.cmd = &cobra.Command{
		Use:   "inspect [account/address]",
		Short: "Print public key by private key.",
		RunE: func(cmd *cobra.Command, args []string) error {
			return b.inspect()
		},
	}

	b.addFlags()

	return b.cmd
}

func (b *EthAccountCommand) inspect() error {
	fileContent, err := os.ReadFile(b.file)
	if err != nil {
		return err
	}
	hexPrivateKey := string(fileContent)
	privateKeyBytes, err := hex.DecodeString(hexPrivateKey)
	if err != nil {
		return err
	}
	privateKey, err := crypto.ToECDSA(privateKeyBytes)
	if err != nil {
		return err
	}
	publicKey := privateKey.Public().(*ecdsa.PublicKey)
	fmt.Printf("Address: %s\n", crypto.PubkeyToAddress(privateKey.PublicKey))
	fmt.Printf("Public key: %s\n", hex.EncodeToString(crypto.FromECDSAPub(&privateKey.PublicKey)))
	fmt.Printf("Compress public key: %#x\n", crypto.CompressPubkey(publicKey))
	return nil
}

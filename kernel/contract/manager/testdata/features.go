package main

import (
	"errors"
	"math/big"

	"github.com/xuperchain/contract-sdk-go/code"
	"github.com/xuperchain/contract-sdk-go/driver"
)

type features struct{}

func (c *features) Initialize(ctx code.Context) code.Response {
	return code.OK(nil)
}

func (c *features) QueryBlock(ctx code.Context) code.Response {
	block_id := ctx.Args()["block_id"]
	block, err := ctx.QueryBlock(string(block_id))
	if err != nil {
		return code.Error(err)
	}
	return code.JSON(block)
}

func (c *features) QueryTx(ctx code.Context) code.Response {
	txid := ctx.Args()["txid"]
	tx, err := ctx.QueryTx(string(txid))
	if err != nil {
		return code.Error(err)
	}
	return code.JSON(tx)
}
func (c *features) Logging(ctx code.Context) code.Response {
	ctx.Logf("log from contract")
	return code.OK(nil)
}
func (c *features) Transfer(ctx code.Context) code.Response {
	to := ctx.Args()["to"]
	amountBytes := ctx.Args()["amount"]
	amount, ok := new(big.Int).SetString(string(amountBytes), 10)
	if !ok {
		return code.Error(errors.New("bad amount format"))
	}
	err := ctx.Transfer(string(to), amount)
	if err != nil {
		return code.Error(err)
	}
	return code.OK(nil)
}
func main() {
	driver.Serve(new(features))
}

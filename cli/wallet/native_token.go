package wallet

import (
	"encoding/json"
	"errors"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/multiverse-dev/saiya/cli/flags"
	"github.com/multiverse-dev/saiya/cli/input"
	"github.com/multiverse-dev/saiya/cli/options"
	"github.com/multiverse-dev/saiya/pkg/core/transaction"
	"github.com/multiverse-dev/saiya/pkg/wallet"
	"github.com/urfave/cli"
)

var (
	balanceFlags = []cli.Flag{
		WalletPathFlag,
		flags.AddressFlag{
			Name:  "address, a",
			Usage: "address",
		},
	}
	transferFlags = []cli.Flag{
		WalletPathFlag,
		FromAddrFlag,
		toAddrFlag,
		cli.StringFlag{
			Name:  "amount",
			Usage: "Amount of asset to send",
		},
	}
)

func newNativeTokenCommands() []cli.Command {
	balanceFlags = append(balanceFlags, options.RPC...)
	transferFlags = append(transferFlags, options.RPC...)
	return []cli.Command{
		{
			Name:      "balance",
			Usage:     "get address GAS balance",
			UsageText: "balance --wallet <path> --rpc-endpoint <node> [--timeout <time>] [--address <address>]",
			Action:    balance,
			Flags:     balanceFlags,
		},
		{
			Name:      "transfer",
			Usage:     "transfer GAS to address",
			UsageText: "transfer --wallet <path> --rpc-endpoint <node> [--from <fromAddress>] --to <toAddress> --amount <amount>",
			Action:    transferNativeToken,
			Flags:     transferFlags,
		},
	}
}

func balance(ctx *cli.Context) error {
	var accounts []*wallet.Account

	wall, err := ReadWallet(ctx.String("wallet"))
	if err != nil {
		return cli.NewExitError(fmt.Errorf("bad wallet: %w", err), 1)
	}
	defer wall.Close()

	addr := ctx.Generic("address").(*flags.Address)
	if addr.IsSet {
		acc := wall.GetAccount(addr.Address())
		if acc == nil {
			return cli.NewExitError(fmt.Errorf("can't find account for the address: %s", addr), 1)
		}
		accounts = append(accounts, acc)
	} else {
		if len(wall.Accounts) == 0 {
			return cli.NewExitError(errors.New("no accounts in the wallet"), 1)
		}
		accounts = wall.Accounts
	}

	gctx, cancel := options.GetTimeoutContext(ctx)
	defer cancel()

	c, err := options.GetRPCClient(gctx, ctx)
	if err != nil {
		return cli.NewExitError(err, 1)
	}

	for k, acc := range accounts {
		addr := acc.Address
		if err != nil {
			return cli.NewExitError(fmt.Errorf("invalid account address: %w", err), 1)
		}
		balance, err := c.Eth_GetBalance(addr)
		fmt.Println(balance)
		if err != nil {
			return cli.NewExitError(err, 1)
		}

		if k != 0 {
			fmt.Fprintln(ctx.App.Writer)
		}
		fmt.Fprintf(ctx.App.Writer, "Account %s, Balance: %s\n", addr, balance)
	}
	return nil
}

func transferNativeToken(ctx *cli.Context) error {
	wall, err := ReadWallet(ctx.String("wallet"))
	if err != nil {
		return cli.NewExitError(err, 1)
	}
	toFlag := ctx.Generic("to").(*flags.Address)
	if !toFlag.IsSet {
		return cli.NewExitError(fmt.Errorf("missing to address"), 1)
	}
	to := toFlag.Address()
	if to == (common.Address{}) {
		return cli.NewExitError(fmt.Errorf("invalid to address %s", to), 1)
	}
	samount := ctx.String("amount")
	amount, ok := big.NewInt(0).SetString(samount, 10)
	if !ok {
		return cli.NewExitError(fmt.Errorf("could not parse amount: %s", samount), 1)
	}
	var fromAcc *wallet.Account
	fromFlag := ctx.Generic("from").(*flags.Address)
	if fromFlag.IsSet {
		from := fromFlag.Address()
		if from == (common.Address{}) {
			return cli.NewExitError(fmt.Errorf("invalid from address"), 1)
		}
		for _, acc := range wall.Accounts {
			if acc.Address == from {
				fromAcc = acc
			}
		}
		if fromAcc == nil {
			return cli.NewExitError(fmt.Errorf("could not find account in wallet: %s", from), 1)
		}
	} else {
		if len(wall.Accounts) == 0 {
			return cli.NewExitError(fmt.Errorf("could not find any account in wallet"), 1)
		}
		fromAcc = wall.Accounts[0]
		for _, acc := range wall.Accounts {
			if acc.Default {
				fromAcc = acc
			}
		}
	}
	pass, err := input.ReadPassword("Enter wallet password > ")
	if err != nil {
		return cli.NewExitError(fmt.Errorf("error reading password: %w", err), 1)
	}
	err = fromAcc.Decrypt(pass, wall.Scrypt)
	if err != nil {
		return cli.NewExitError(fmt.Errorf("unable to decrypt account: %s", fromAcc.Address), 1)
	}
	gctx, cancel := options.GetTimeoutContext(ctx)
	defer cancel()

	c, err := options.GetRPCClient(gctx, ctx)
	if err != nil {
		return cli.NewExitError(err, 1)
	}
	nonce, err := c.Eth_GetTransactionCount(fromAcc.Address)
	if err != nil {
		return cli.NewExitError(fmt.Errorf("failed get account nonce: %w", err), 1)
	}
	feePerByte, err := c.GetFeePerByte()
	if err != nil {
		return cli.NewExitError(fmt.Errorf("failed get fee per byte: %w", err), 1)
	}
	gasPrice, err := c.Eth_GasPrice()
	if err != nil {
		return cli.NewExitError(fmt.Errorf("failed get fee per byte: %w", err), 1)
	}
	t := &types.LegacyTx{
		Nonce:    nonce,
		GasPrice: gasPrice,
		Gas:      0,
		To:       &to,
		Value:    amount,
		Data:     []byte{},
	}
	tx := transaction.NewTx(t)
	netfee := transaction.CalculateNetworkFee(tx, feePerByte)
	if err != nil {
		return cli.NewExitError(fmt.Errorf("failed calculate network fee: %w", err), 1)
	}
	t.Gas = netfee
	chainId, err := c.Eth_ChainId()
	if err != nil {
		return cli.NewExitError(fmt.Errorf("failed to get chainId: %w", err), 1)
	}
	fromAcc.SignTx(chainId, tx)
	b, err := json.Marshal(tx)
	if err != nil {
		panic(err)
	}
	fmt.Println(string(b))
	b, err = rlp.EncodeToBytes(tx.LegacyTx)
	if err != nil {
		return cli.NewExitError(fmt.Errorf("failed encode tx to bytes: %w", err), 1)
	}
	hash, err := c.Eth_SendRawTransaction(b)
	if err != nil {
		return cli.NewExitError(fmt.Errorf("failed relay tx: %w", err), 1)
	}
	fmt.Fprintf(ctx.App.Writer, "TxHash: %s\n", hash)
	return nil
}

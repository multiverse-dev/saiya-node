package wallet

import (
	"errors"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/multiverse-dev/saiya/cli/flags"
	"github.com/multiverse-dev/saiya/cli/options"
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
			Usage:     "get address SAI balance",
			UsageText: "balance --wallet <path> --rpc-endpoint <node> [--timeout <time>] [--address <address>]",
			Action:    balance,
			Flags:     balanceFlags,
		},
		{
			Name:      "transfer",
			Usage:     "transfer SAI to address",
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
	var from common.Address
	fromFlag := ctx.Generic("from").(*flags.Address)
	if fromFlag.IsSet {
		from = fromFlag.Address()
		if from == (common.Address{}) {
			return cli.NewExitError(fmt.Errorf("invalid from address"), 1)
		}
	} else {
		if len(wall.Accounts) == 0 {
			return cli.NewExitError(fmt.Errorf("could not find any account in wallet"), 1)
		}
		facc := wall.Accounts[0]
		for _, acc := range wall.Accounts {
			if acc.Default {
				facc = acc
			}
		}
		from = facc.Address
	}
	return MakeNeoTx(ctx, wall, from, to, amount, []byte{})
}

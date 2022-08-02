package native

import (
	"fmt"
	"math/big"
	"strconv"

	"github.com/ethereum/go-ethereum/common"
	"github.com/multiverse-dev/saiya/cli/options"
	"github.com/multiverse-dev/saiya/cli/wallet"
	"github.com/multiverse-dev/saiya/pkg/core/native"
	"github.com/multiverse-dev/saiya/pkg/core/native/nativenames"
	"github.com/urfave/cli"
)

func newPolicyCommands() []cli.Command {
	flags := append(options.RPC, wallet.WalletPathFlag)
	return []cli.Command{
		{
			Name:      "block",
			Usage:     "block account",
			ArgsUsage: "<address>",
			Action:    blockAccount,
			Flags:     flags,
		},
		{
			Name:      "unblock",
			Usage:     "unblock account",
			ArgsUsage: "<address>",
			Action:    unblockAccount,
			Flags:     flags,
		},
		{
			Name:      "isblocked",
			Usage:     "isBlocked account",
			ArgsUsage: "<address>",
			Action:    isBlocked,
			Flags:     options.RPC,
		},
		{
			Name:      "set",
			Usage:     "set crucial parameters",
			ArgsUsage: "<address>",
			Subcommands: []cli.Command{
				{
					Name:      "feePerByte",
					Usage:     "set FeePerByte of tx",
					ArgsUsage: "<number>",
					Action:    setFeePerByte,
					Flags:     flags,
				},
				{
					Name:      "gasPrice",
					Usage:     "set GasPrice of tx",
					ArgsUsage: "<number>",
					Action:    setGasPrice,
					Flags:     flags,
				},
			},
		},
	}
}

func isBlocked(ctx *cli.Context) error {
	if len(ctx.Args()) < 1 {
		return cli.NewExitError(fmt.Errorf("please input address"), 1)
	}
	addrHex := ctx.Args().First()
	address := common.HexToAddress(addrHex)
	gctx, cancel := options.GetTimeoutContext(ctx)
	defer cancel()
	c, err := options.GetRPCClient(gctx, ctx)
	if err != nil {
		return cli.NewExitError(err, 1)
	}
	r, er := c.IsBlocked(address)
	if er != nil {
		return cli.NewExitError(fmt.Errorf("failed get isBlocked %w", er), 1)
	}
	fmt.Fprintf(ctx.App.Writer, "%t\n", r)
	return nil
}

func blockAccount(ctx *cli.Context) error {
	address, err := parseAddressInput(ctx)
	if err != nil {
		return err
	}
	return callPolicy(ctx, "blockAccount", address)
}

func unblockAccount(ctx *cli.Context) error {
	address, err := parseAddressInput(ctx)
	if err != nil {
		return err
	}
	return callPolicy(ctx, "unblockAccount", address)
}

func setFeePerByte(ctx *cli.Context) error {
	value, err := parseUint64Input(ctx)
	if err != nil {
		return err
	}
	return callPolicy(ctx, "setFeePerByte", value)
}

func setGasPrice(ctx *cli.Context) error {
	value, err := parseBigInput(ctx)
	if err != nil {
		return err
	}
	return callPolicy(ctx, "setGasPrice", value)
}

func parseAddressInput(ctx *cli.Context) (common.Address, error) {
	if len(ctx.Args()) < 1 {
		return common.Address{}, cli.NewExitError(fmt.Errorf("please input address"), 1)
	}
	addrHex := ctx.Args().First()
	address := common.HexToAddress(addrHex)
	return address, nil
}

func parseUint64Input(ctx *cli.Context) (uint64, error) {
	if len(ctx.Args()) < 1 {
		return 0, cli.NewExitError(fmt.Errorf("please input address"), 1)
	}
	num := ctx.Args().First()
	param, err := strconv.ParseUint(num, 10, 64)
	if err != nil {
		return 0, cli.NewExitError(fmt.Errorf("invalid number %s", num), 1)
	}
	return param, nil
}

func parseBigInput(ctx *cli.Context) ([]byte, error) {
	if len(ctx.Args()) < 1 {
		return nil, cli.NewExitError(fmt.Errorf("please input address"), 1)
	}
	num := ctx.Args().First()
	param, ok := big.NewInt(0).SetString(num, 10)
	if !ok {
		return nil, cli.NewExitError(fmt.Errorf("invalid number %s", num), 1)
	}
	return param.Bytes(), nil
}

func callPolicy(ctx *cli.Context, method string, args ...interface{}) error {
	pabi, err := getNativeContract(ctx, nativenames.Policy)
	if err != nil {
		return err
	}
	data, err := pabi.Pack(method, args...)
	if err != nil {
		return cli.NewExitError(fmt.Errorf("can't pack inputs for %s: %w", method, err), 1)
	}
	return makeCommitteeTx(ctx, native.PolicyAddress, data)
}

package native

import (
	"encoding/binary"
	"fmt"
	"math/big"
	"strconv"

	"github.com/ethereum/go-ethereum/common"
	"github.com/multiverse-dev/saiya/cli/options"
	"github.com/multiverse-dev/saiya/cli/wallet"
	"github.com/multiverse-dev/saiya/pkg/core/native"
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
			Name:      "is-blocked",
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
	input, err := parseAddressInput(ctx)
	if err != nil {
		return err
	}
	return callPolicy(ctx, native.PrefixBlockedAcount, input)
}

func unblockAccount(ctx *cli.Context) error {
	input, err := parseAddressInput(ctx)
	if err != nil {
		return err
	}
	return callPolicy(ctx, native.PrefixBlockedAcount+1, input)
}

func setFeePerByte(ctx *cli.Context) error {
	input, err := parseUint64Input(ctx)
	if err != nil {
		return err
	}
	return callPolicy(ctx, native.PrefixFeePerByte, input)
}

func setGasPrice(ctx *cli.Context) error {
	input, err := parseBigInput(ctx)
	if err != nil {
		return err
	}
	return callPolicy(ctx, native.PrefixGasPrice, input)
}

func parseAddressInput(ctx *cli.Context) ([]byte, error) {
	if len(ctx.Args()) < 1 {
		return nil, cli.NewExitError(fmt.Errorf("please input address"), 1)
	}
	addrHex := ctx.Args().First()
	address := common.HexToAddress(addrHex)
	return address.Bytes(), nil
}

func parseUint64Input(ctx *cli.Context) ([]byte, error) {
	if len(ctx.Args()) < 1 {
		return nil, cli.NewExitError(fmt.Errorf("please input address"), 1)
	}
	num := ctx.Args().First()
	param, err := strconv.ParseUint(num, 10, 64)
	if err != nil {
		return nil, cli.NewExitError(fmt.Errorf("invalid number %s", num), 1)
	}
	data := make([]byte, 8)
	binary.BigEndian.PutUint64(data, param)
	return data, nil
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

func callPolicy(ctx *cli.Context, method byte, input []byte) error {
	data := []byte{method}
	data = append(data, input...)
	return makeCommitteeTx(ctx, native.PolicyAddress, data)
}

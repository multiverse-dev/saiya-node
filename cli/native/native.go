package native

import (
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/multiverse-dev/saiya/cli/options"
	"github.com/multiverse-dev/saiya/cli/wallet"
	"github.com/urfave/cli"
)

func NewCommands() []cli.Command {
	return []cli.Command{{
		Name:  "native",
		Usage: "invoke native contract",
		Subcommands: []cli.Command{
			{
				Name:        "designate",
				Usage:       "designate committee or validators",
				Subcommands: newDesignateCommands(),
			},
			{
				Name:        "policy",
				Usage:       "manage policy",
				Subcommands: newPolicyCommands(),
			},
		},
	},
	}
}

func getNativeContract(ctx *cli.Context, name string) (*abi.ABI, error) {
	gctx, cancel := options.GetTimeoutContext(ctx)
	defer cancel()
	var err error
	c, err := options.GetRPCClient(gctx, ctx)
	if err != nil {
		return nil, cli.NewExitError(err, 1)
	}
	natives, err := c.GetNativeContracts()
	if err != nil {
		cli.NewExitError(fmt.Errorf("could not get native contracts: %w", err), 1)
	}
	var nabi *abi.ABI
	for _, n := range natives {
		if n.Name == name {
			nabi = &n.Abi
		}
	}
	if nabi == nil {
		return nil, cli.NewExitError("can't find designate contract", 1)
	}
	return nabi, nil
}

func makeCommitteeTx(ctx *cli.Context, to common.Address, data []byte) error {
	wall, err := wallet.ReadWallet(ctx.String("wallet"))
	if err != nil {
		return cli.NewExitError(err, 1)
	}
	gctx, cancel := options.GetTimeoutContext(ctx)
	defer cancel()
	c, err := options.GetRPCClient(gctx, ctx)
	if err != nil {
		return cli.NewExitError(err, 1)
	}
	committeeAddr, err := c.GetCommitteeAddress()
	if err != nil {
		return cli.NewExitError(fmt.Errorf("failed get committee address: %w", err), 1)
	}
	return wallet.MakeNeoTx(ctx, wall, committeeAddr, to, big.NewInt(0), data)
}

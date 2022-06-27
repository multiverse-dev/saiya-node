package query

import (
	"encoding/hex"
	"encoding/json"
	"fmt"

	"github.com/ethereum/go-ethereum/common"
	"github.com/multiverse-dev/saiya/cli/options"
	"github.com/urfave/cli"
)

// NewCommands returns 'query' command.
func NewCommands() []cli.Command {
	queryTxFlags := append([]cli.Flag{
		cli.BoolFlag{
			Name:  "verbose, v",
			Usage: "Output full tx info and execution logs",
		},
	}, options.RPC...)
	return []cli.Command{{
		Name:  "query",
		Usage: "Query data from RPC node",
		Subcommands: []cli.Command{
			{
				Name:   "committee",
				Usage:  "Get committee list",
				Action: queryCommittee,
				Flags:  options.RPC,
			},
			{
				Name:   "height",
				Usage:  "Get node height",
				Action: queryHeight,
				Flags:  options.RPC,
			},
			{
				Name:   "tx",
				Usage:  "Query transaction status",
				Action: queryTx,
				Flags:  queryTxFlags,
			},
		},
	}}
}

func queryTx(ctx *cli.Context) error {
	args := ctx.Args()
	if len(args) == 0 {
		return cli.NewExitError("Transaction hash is missing", 1)
	}

	txHash := common.HexToHash(args[0])

	gctx, cancel := options.GetTimeoutContext(ctx)
	defer cancel()
	var err error
	c, err := options.GetRPCClient(gctx, ctx)
	if err != nil {
		return cli.NewExitError(err, 1)
	}

	txOut, err := c.GetRawTransactionVerbose(txHash)
	if err != nil {
		return cli.NewExitError(err, 1)
	}
	b, err := json.Marshal(txOut)
	if err != nil {
		return cli.NewExitError(err, 1)
	}
	fmt.Fprintln(ctx.App.Writer, string(b))
	return dumpReceipt(txHash)
}

func dumpReceipt(txHash common.Hash) error {
	return nil
}

func queryCommittee(ctx *cli.Context) error {
	var err error
	gctx, cancel := options.GetTimeoutContext(ctx)
	defer cancel()

	c, err := options.GetRPCClient(gctx, ctx)
	if err != nil {
		return cli.NewExitError(err, 1)
	}
	comm, err := c.GetCommittee()
	if err != nil {
		return cli.NewExitError(err, 1)
	}
	for _, k := range comm {
		fmt.Fprintln(ctx.App.Writer, hex.EncodeToString(k.Bytes()))
	}
	return nil
}

func queryHeight(ctx *cli.Context) error {
	var err error

	gctx, cancel := options.GetTimeoutContext(ctx)
	defer cancel()

	c, err := options.GetRPCClient(gctx, ctx)
	if err != nil {
		return cli.NewExitError(err, 1)
	}

	blockCount, err := c.GetBlockCount()
	if err != nil {
		return cli.NewExitError(err, 1)
	}
	blockHeight := blockCount - 1 // GetBlockCount returns block count (including 0), not the highest block index.

	fmt.Fprintf(ctx.App.Writer, "Latest block: %d\n", blockHeight)

	stateHeight, err := c.GetStateHeight()
	if err == nil { // We can be talking to a node without getstateheight request support.
		fmt.Fprintf(ctx.App.Writer, "Validated state: %d\n", stateHeight.Validated)
	}

	return nil
}

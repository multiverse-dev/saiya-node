package native

import (
	"encoding/json"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/multiverse-dev/saiya/cli/input"
	"github.com/multiverse-dev/saiya/cli/options"
	"github.com/multiverse-dev/saiya/cli/wallet"
	"github.com/multiverse-dev/saiya/pkg/core/transaction"
	"github.com/multiverse-dev/saiya/pkg/evm"
	"github.com/multiverse-dev/saiya/pkg/rpc/response/result"
	coreW "github.com/multiverse-dev/saiya/pkg/wallet"
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
	committees, err := c.GetCommittee()
	if err != nil {
		return cli.NewExitError(fmt.Errorf("failed get committee: %w", err), 1)
	}
	var fromAccs []*coreW.Account
	if len(wall.Accounts) == 0 {
		return cli.NewExitError(fmt.Errorf("failed find any account in wallet"), 1)
	}
	for _, acc := range wall.Accounts {
		for _, p := range committees {
			if p.Address() == acc.Address {
				pass, err := input.ReadPassword(fmt.Sprintf("Enter password for %s > ", acc.Address))
				if err != nil {
					return cli.NewExitError(fmt.Errorf("error reading password: %w", err), 1)
				}
				err = acc.Decrypt(pass, wall.Scrypt)
				if err != nil {
					return cli.NewExitError(fmt.Errorf("unable to decrypt account: %s", acc.Address), 1)
				}
				fromAccs = append(fromAccs, acc)
			}
		}
	}
	if len(fromAccs) == 0 {
		return cli.NewExitError(fmt.Errorf("failed find any committee member in wallet"), 1)
	}
	feePerByte, err := c.GetFeePerByte()
	if err != nil {
		return cli.NewExitError(fmt.Errorf("failed get fee per byte: %w", err), 1)
	}
	gasPrice, err := c.Eth_GasPrice()
	if err != nil {
		return cli.NewExitError(fmt.Errorf("failed get gas price: %w", err), 1)
	}
	chainId, err := c.Eth_ChainId()
	if err != nil {
		return cli.NewExitError(fmt.Errorf("failed to get chainId: %w", err), 1)
	}
	committeeAddr, script, m, err := getCommitteeAddress(committees)
	if err != nil {
		return err
	}
	nonce, err := c.Eth_GetTransactionCount(committeeAddr)
	if err != nil {
		return cli.NewExitError(fmt.Errorf("failed get account nonce: %w", err), 1)
	}
	t := &transaction.SaiyaTx{
		Nonce:    nonce,
		GasPrice: gasPrice,
		Gas:      0,
		From:     committeeAddr,
		To:       &to,
		Value:    big.NewInt(0),
		Data:     data,
		Witness: transaction.Witness{
			VerificationScript: script,
		},
	}
	tx := transaction.NewTx(t)
	netfee := transaction.CalculateNetworkFee(tx, feePerByte)
	g, err := c.Eth_EstimateGas(&result.TransactionObject{
		From:     t.From,
		To:       t.To,
		Data:     t.Data,
		Value:    t.Value,
		GasPrice: t.GasPrice,
		Gas:      evm.TestGas,
	})
	if err != nil {
		return cli.NewExitError(fmt.Errorf("failed estimate gas fee: %w", err), 1)
	}
	t.Gas = netfee + g
	if m == 1 {
		fromAccs[0].SignTx(chainId, tx)

	} else {
		signContext := wallet.SignContext{
			ChainID:    chainId,
			Tx:         *t,
			Sigs:       make([][]byte, len(committees)),
			PublicKeys: committees,
			M:          m,
		}
		for _, acc := range fromAccs {
			for i, a := range committees {
				if acc.Address == a.Address() {
					signContext.Sigs[i] = acc.PrivateKey().SignHashable(chainId, t)
				}
			}
		}
		if signContext.IsComplete() {
			tx = signContext.CreateTx()
		} else {
			b, err := json.Marshal(signContext)
			if err != nil {
				return cli.NewExitError(fmt.Errorf("failed marshal sign context json: %w", err), 1)
			}
			fmt.Fprintf(ctx.App.Writer, "SignContext: %s\n", string(b))
			return nil
		}
	}
	b, err := tx.Bytes()
	if err != nil {
		return cli.NewExitError(fmt.Errorf("failed encode tx: %w", err), 1)
	}
	hash, err := c.SendRawTransaction(b[1:])
	if err != nil {
		return cli.NewExitError(fmt.Errorf("failed relay tx: %w", err), 1)
	}
	fmt.Fprintf(ctx.App.Writer, "TxHash: %s\n", hash)
	return nil
}

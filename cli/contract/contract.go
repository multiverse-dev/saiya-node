package contract

import (
	"encoding/hex"
	"errors"
	"fmt"
	"math/big"
	"os"
	"strconv"
	"strings"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/multiverse-dev/saiya/cli/flags"
	"github.com/multiverse-dev/saiya/cli/input"
	"github.com/multiverse-dev/saiya/cli/options"
	"github.com/multiverse-dev/saiya/cli/wallet"
	"github.com/multiverse-dev/saiya/pkg/rpc/response/result"
	corew "github.com/multiverse-dev/saiya/pkg/wallet"
	"github.com/urfave/cli"
)

func NewCommands() []cli.Command {
	return []cli.Command{
		{
			Name:        "contract",
			Description: "contract operations",
			Subcommands: []cli.Command{
				{
					Name:   "call",
					Usage:  "call [contractAddress] [abiFilePath] [method] [inputs...]",
					Action: call,
					Flags: append(options.RPC, []cli.Flag{
						wallet.WalletPathFlag,
						wallet.FromAddrFlag,
					}...),
				},
				{
					Name:   "deploy",
					Usage:  "deploy [byteCodeFilePath] [abiFilePath] [inputs...]",
					Action: deploy,
					Flags: append(options.RPC, []cli.Flag{
						wallet.WalletPathFlag,
						wallet.FromAddrFlag,
					}...),
				},
			},
		},
	}
}

func call(ctx *cli.Context) error {
	if len(ctx.Args()) < 3 {
		return cli.NewExitError("parameters not enough", 1)
	}
	address := common.HexToAddress(ctx.Args()[0])
	if address == (common.Address{}) {
		return cli.NewExitError("invalid contract address", 1)
	}
	abiFile := ctx.Args()[1]
	file, err := os.Open(abiFile)
	if err != nil {
		return err
	}
	defer file.Close()
	contractAbi, err := abi.JSON(file)
	if err != nil {
		return err
	}
	method := ctx.Args()[2]
	var inputs []interface{}
	if len(ctx.Args()) > 3 {
		inputs = make([]interface{}, len(ctx.Args())-3)
		for i := 3; i < len(ctx.Args()); i++ {
			pstr := ctx.Args()[i]
			inputs[i-3], err = parseParam(pstr)
			if err != nil {
				return err
			}
		}
	}
	data, err := contractAbi.Pack(method, inputs...)
	if err != nil {
		return err
	}
	facc, err := handleWalletAndFrom(ctx)
	if err != nil {
		return err
	}
	gctx, cancel := options.GetTimeoutContext(ctx)
	defer cancel()
	c, err := options.GetRPCClient(gctx, ctx)
	if err != nil {
		return cli.NewExitError(err, 1)
	}
	code, err := c.Eth_GetCode(address)
	if err != nil {
		return err
	}
	if len(code) == 0 {
		return cli.NewExitError("contract not found", 1)
	}
	gasPrice, err := c.Eth_GasPrice()
	if err != nil {
		return cli.NewExitError(err, 1)
	}
	ret, err := c.Eth_Call(&result.TransactionObject{
		From:     facc.Address,
		To:       &address,
		Value:    big.NewInt(0),
		GasPrice: gasPrice,
		Data:     data,
	})
	if err != nil {
		return cli.NewExitError(fmt.Errorf("contract call error: %w", err), 1)
	}
	fmt.Fprintf(ctx.App.Writer, "ret: %s\n", hexutil.Encode(ret))
	err = input.ConfirmTx()
	if err != nil {
		return cli.NewExitError(err, 1)
	}
	return wallet.MakeEthTx(ctx, facc, &address, big.NewInt(0), data)
}

func deploy(ctx *cli.Context) error {
	if len(ctx.Args()) < 2 {
		return cli.NewExitError("parameters not enough", 1)
	}
	txt, err := os.ReadFile(ctx.Args()[0])
	if err != nil {
		return cli.NewExitError(fmt.Errorf("can't read bytecode: %w", err), 1)
	}
	bin, err := hexutil.Decode(strings.Replace(string(txt), "\n", "", -1))
	if err != nil {
		return cli.NewExitError(fmt.Errorf("can't parse bytecode: %s, %w", string(txt), err), 1)
	}
	abiFile := ctx.Args()[1]
	file, err := os.Open(abiFile)
	if err != nil {
		return err
	}
	defer file.Close()
	contractAbi, err := abi.JSON(file)
	if err != nil {
		return err
	}
	data := bin
	needParamCount := len(contractAbi.Constructor.Inputs)
	if len(ctx.Args())-2 < needParamCount {
		return cli.NewExitError("constructor params not enough", 1)
	}
	if len(ctx.Args())-2 > needParamCount {
		return cli.NewExitError("too many params", 1)
	}
	if needParamCount > 0 {
		inputs := make([]interface{}, needParamCount)
		for i := 3; i < len(ctx.Args()); i++ {
			pstr := ctx.Args()[i]
			inputs[i-3], err = parseParam(pstr)
			if err != nil {
				return err
			}
		}
		arg, err := contractAbi.Constructor.Inputs.Pack(inputs...)
		if err != nil {
			return cli.NewExitError(fmt.Errorf("can't pack constructor inputs: %w", err), 1)
		}
		data = append(data, arg...)
	}
	facc, err := handleWalletAndFrom(ctx)
	if err != nil {
		return err
	}
	return wallet.MakeEthTx(ctx, facc, nil, big.NewInt(0), data)
}

func parseParam(pstr string) (interface{}, error) {
	if strings.HasPrefix(pstr, "0x") {
		str := pstr[2:]
		if len(str) == 2*common.AddressLength {
			return common.HexToAddress(pstr), nil
		} else if len(str) == 2*common.HashLength {
			return common.HexToHash(pstr), nil
		}
		b, err := hex.DecodeString(str)
		if err != nil {
			return nil, err
		}
		return big.NewInt(0).SetBytes(b), nil
	}
	val, err := strconv.ParseUint(pstr, 10, 32)
	if err == nil {
		return val, nil
	}
	val, err = strconv.ParseUint(pstr, 16, 32)
	if err == nil {
		return val, nil
	}
	b, err := hex.DecodeString(pstr)
	if err == nil {
		return big.NewInt(0).SetBytes(b), nil
	}
	return nil, errors.New("can't parse parameter")
}

func handleWalletAndFrom(ctx *cli.Context) (*corew.Account, error) {
	wall, err := wallet.ReadWallet(ctx.String("wallet"))
	if err != nil {
		return nil, cli.NewExitError(err, 1)
	}
	var facc *corew.Account
	fromFlag := ctx.Generic("from").(*flags.Address)
	if fromFlag.IsSet {
		from := fromFlag.Address()
		if from == (common.Address{}) {
			return nil, cli.NewExitError(fmt.Errorf("invalid from address"), 1)
		}
		for _, acc := range wall.Accounts {
			if acc.Address == from && !acc.IsMultiSig() {
				facc = acc
			}
		}
	} else {
		if len(wall.Accounts) == 0 {
			return nil, cli.NewExitError(fmt.Errorf("could not find any account in wallet"), 1)
		}
		facc = wall.Accounts[0]
		for _, acc := range wall.Accounts {
			if acc.Default {
				facc = acc
			}
		}
	}
	if facc == nil {
		return nil, cli.NewExitError("account not found", 1)
	}
	pass, err := input.ReadPassword(fmt.Sprintf("Enter %s password > ", facc.Address))
	if err != nil {
		return nil, cli.NewExitError(fmt.Errorf("error reading password: %w", err), 1)
	}
	err = facc.Decrypt(pass, wall.Scrypt)
	if err != nil {
		return nil, cli.NewExitError(fmt.Errorf("unable to decrypt account: %s", facc.Address), 1)
	}
	return facc, nil
}

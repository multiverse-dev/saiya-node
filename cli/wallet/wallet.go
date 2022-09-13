package wallet

import (
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/big"
	"os"
	"strings"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/multiverse-dev/saiya/cli/flags"
	"github.com/multiverse-dev/saiya/cli/input"
	"github.com/multiverse-dev/saiya/cli/options"
	"github.com/multiverse-dev/saiya/pkg/core/transaction"
	"github.com/multiverse-dev/saiya/pkg/crypto"
	"github.com/multiverse-dev/saiya/pkg/crypto/hash"
	"github.com/multiverse-dev/saiya/pkg/crypto/keys"
	"github.com/multiverse-dev/saiya/pkg/rpc/response/result"
	"github.com/multiverse-dev/saiya/pkg/wallet"
	"github.com/urfave/cli"
)

var (
	errNoPath         = errors.New("wallet path is mandatory and should be passed using (--wallet, -w) flags")
	errPhraseMismatch = errors.New("the entered pass-phrases do not match. Maybe you have misspelled them")
	errNoStdin        = errors.New("can't read wallet from stdin for this command")
)

var (
	WalletPathFlag = cli.StringFlag{
		Name:  "wallet, w",
		Usage: "Target location of the wallet file ('-' to read from stdin).",
	}
	keyFlag = cli.StringFlag{
		Name:  "key",
		Usage: "private key to import",
	}
	pswFlag = cli.StringFlag{
		Name:  "psw",
		Usage: "password to encypt private key",
	}
	decryptFlag = flags.AddressFlag{
		Name:  "decrypt, d",
		Usage: "Decrypt encrypted keys.",
	}
	outFlag = cli.StringFlag{
		Name:  "out",
		Usage: "file to put JSON transaction to",
	}
	inFlag = cli.StringFlag{
		Name:  "in",
		Usage: "file with JSON transaction",
	}
	FromAddrFlag = flags.AddressFlag{
		Name:  "from",
		Usage: "Address to send an asset from",
	}
	toAddrFlag = flags.AddressFlag{
		Name:  "to",
		Usage: "Address to send an asset to",
	}
	forceFlag = cli.BoolFlag{
		Name:  "force",
		Usage: "Do not ask for a confirmation",
	}
)

// NewCommands returns 'wallet' command.
func NewCommands() []cli.Command {
	listFlags := []cli.Flag{
		WalletPathFlag,
	}
	listFlags = append(listFlags, options.RPC...)
	return []cli.Command{{
		Name:  "wallet",
		Usage: "create, open and manage a saiya wallet",
		Subcommands: []cli.Command{
			{
				Name:   "init",
				Usage:  "create a new wallet",
				Action: createWallet,
				Flags: []cli.Flag{
					WalletPathFlag,
					cli.BoolFlag{
						Name:  "account, a",
						Usage: "Create a new account",
					},
				},
			},
			{
				Name:   "change-password",
				Usage:  "change password for accounts",
				Action: changePassword,
				Flags: []cli.Flag{
					WalletPathFlag,
					flags.AddressFlag{
						Name:  "address, a",
						Usage: "address to change password for",
					},
				},
			},
			{
				Name:   "create",
				Usage:  "add an account to the existing wallet",
				Action: addAccount,
				Flags: []cli.Flag{
					WalletPathFlag,
				},
			},
			{
				Name:   "dump-keys",
				Usage:  "dump public keys for account",
				Action: dumpKeys,
				Flags: []cli.Flag{
					WalletPathFlag,
					flags.AddressFlag{
						Name:  "address, a",
						Usage: "address to print public keys for",
					},
				},
			},
			{
				Name:      "export",
				Usage:     "export keys for address",
				UsageText: "export --wallet <path> --decrypt <address>",
				Action:    exportKeys,
				Flags: []cli.Flag{
					WalletPathFlag,
					decryptFlag,
				},
			},
			{
				Name:      "import",
				Usage:     "import private key",
				UsageText: "import --wallet <path> --key <privateKey> --psw <password> [--name <account_name>]",
				Action:    importWallet,
				Flags: []cli.Flag{
					WalletPathFlag,
					keyFlag,
					pswFlag,
					cli.StringFlag{
						Name:  "name, n",
						Usage: "Optional account name",
					},
				},
			},
			{
				Name:  "import-multisig",
				Usage: "import multisig account",
				UsageText: "import-multisig --wallet <path> [--name <account_name>] --min <n>" +
					" [<pubkey1> [<pubkey2> [...]]]",
				Action: importMultisig,
				Flags: []cli.Flag{
					WalletPathFlag,
					cli.StringFlag{
						Name:  "name, n",
						Usage: "Optional account name",
					},
					cli.IntFlag{
						Name:  "min, m",
						Usage: "Minimal number of signatures",
					},
				},
			},
			{
				Name:      "remove",
				Usage:     "remove an account from the wallet",
				UsageText: "remove --wallet <path> [--force] --address <addr>",
				Action:    removeAccount,
				Flags: []cli.Flag{
					WalletPathFlag,
					forceFlag,
					flags.AddressFlag{
						Name:  "address, a",
						Usage: "Account address or hash in LE form to be removed",
					},
				},
			},
			{
				Name:      "list",
				Usage:     "list addresses in wallet",
				UsageText: "list --wallet <path> --rpc-endpoint <node> [--timeout <time>]",
				Action:    listAddresses,
				Flags:     listFlags,
			},
			{
				Name:        "gas",
				Usage:       "work with native gas",
				Subcommands: newNativeTokenCommands(),
			},
			{
				Name:      "sign",
				Usage:     "sign sign_context",
				UsageText: "sign --wallet <path> --rpc-endpoint <node> [--timeout <time>] --context <contextJson>",
				Action:    sign,
				Flags: append(listFlags, cli.StringFlag{
					Name:  "context, c",
					Usage: "sign a context",
				}),
			},
		},
	}}
}

func listAddresses(ctx *cli.Context) error {
	wall, err := openWallet(ctx.String("wallet"))
	if err != nil {
		return cli.NewExitError(err, 1)
	}
	gctx, cancel := options.GetTimeoutContext(ctx)
	defer cancel()

	c, err := options.GetRPCClient(gctx, ctx)
	if err != nil {
		return cli.NewExitError(err, 1)
	}
	for _, acc := range wall.Accounts {
		bal, err := c.Eth_GetBalance(acc.Address)
		if err != nil {
			return cli.NewExitError(fmt.Errorf("could not get balance of account %s, err: %w", acc.Address, err), 1)
		}
		fmt.Fprintf(ctx.App.Writer, "%s Sai: %s\n", acc.Address, bal)
	}
	return nil
}

func changePassword(ctx *cli.Context) error {
	wall, err := openWallet(ctx.String("wallet"))
	if err != nil {
		return cli.NewExitError(err, 1)
	}
	addrFlag := ctx.Generic("address").(*flags.Address)
	if addrFlag.IsSet {
		// Check for account presence first before asking for password.
		acc := wall.GetAccount(addrFlag.Address())
		if acc == nil {
			return cli.NewExitError("account is missing", 1)
		}
		if acc.IsMultiSig() {
			return cli.NewExitError("can't change passord of a multi-sig account", 1)
		}
	}

	oldPass, err := input.ReadPassword("Enter password > ")
	if err != nil {
		return cli.NewExitError(fmt.Errorf("error reading old password: %w", err), 1)
	}

	for i := range wall.Accounts {
		if addrFlag.IsSet && wall.Accounts[i].Address != addrFlag.Address() {
			continue
		}
		err := wall.Accounts[i].Decrypt(oldPass, wall.Scrypt)
		if err != nil {
			return cli.NewExitError(fmt.Errorf("unable to decrypt account %s: %w", wall.Accounts[i].Address, err), 1)
		}
	}

	pass, err := readNewPassword()
	if err != nil {
		return cli.NewExitError(fmt.Errorf("error reading new password: %w", err), 1)
	}
	for i := range wall.Accounts {
		if addrFlag.IsSet && wall.Accounts[i].Address != addrFlag.Address() {
			continue
		}
		err := wall.Accounts[i].Encrypt(pass, wall.Scrypt)
		if err != nil {
			return cli.NewExitError(err, 1)
		}
	}
	err = wall.Save()
	if err != nil {
		return cli.NewExitError(fmt.Errorf("error saving the wallet: %w", err), 1)
	}
	return nil
}

func addAccount(ctx *cli.Context) error {
	wall, err := openWallet(ctx.String("wallet"))
	if err != nil {
		return cli.NewExitError(err, 1)
	}

	defer wall.Close()

	if err := createAccount(wall); err != nil {
		return cli.NewExitError(err, 1)
	}

	return nil
}

func exportKeys(ctx *cli.Context) error {
	wall, err := ReadWallet(ctx.String("wallet"))
	if err != nil {
		return cli.NewExitError(err, 1)
	}

	var addr common.Address

	decrypt := ctx.Generic("decrypt").(*flags.Address)
	if !decrypt.IsSet {
		return cli.NewExitError(fmt.Errorf("missing address to decrypt"), 1)
	}
	addr = decrypt.Address()

	var wifs []string

loop:
	for _, a := range wall.Accounts {
		if a.Address != addr {
			continue
		}
		if a.IsMultiSig() {
			return cli.NewExitError("can't export multi-sig account", 1)
		}
		for i := range wifs {
			if a.EncryptedWIF == wifs[i] {
				continue loop
			}
		}

		wifs = append(wifs, a.EncryptedWIF)
	}
	if len(wifs) == 0 {
		return cli.NewExitError(fmt.Errorf("address not found"), 1)
	}
	for _, wif := range wifs {
		pass, err := input.ReadPassword("Enter password > ")
		if err != nil {
			return cli.NewExitError(fmt.Errorf("error reading password: %w", err), 1)
		}

		pk, err := keys.NEP2Decrypt(wif, pass, wall.Scrypt)
		if err != nil {
			return cli.NewExitError(err, 1)
		}
		fmt.Fprintln(ctx.App.Writer, hexutil.Encode(pk.Bytes()))
	}

	return nil
}

func importWallet(ctx *cli.Context) error {
	wall, err := openWallet(ctx.String("wallet"))
	if err != nil {
		return cli.NewExitError(err, 1)
	}
	defer wall.Close()
	b, err := hexutil.Decode(ctx.String("key"))
	if err != nil {
		return cli.NewExitError(err, 1)
	}
	key, err := keys.NewPrivateKeyFromBytes(b)
	if err != nil {
		return cli.NewExitError(err, 1)
	}
	acc := wallet.NewAccountFromPrivateKey(key)
	if err != nil {
		return cli.NewExitError(err, 1)
	}
	pass := ctx.String("psw")
	if err := acc.Encrypt(pass, wall.Scrypt); err != nil {
		return err
	}
	if acc.Label == "" {
		acc.Label = ctx.String("name")
	}
	if err := addAccountAndSave(wall, acc); err != nil {
		return cli.NewExitError(err, 1)
	}

	return nil
}

func importMultisig(ctx *cli.Context) error {
	wall, err := openWallet(ctx.String("wallet"))
	if err != nil {
		return cli.NewExitError(err, 1)
	}

	m := ctx.Int("min")
	if ctx.NArg() < m {
		return cli.NewExitError(errors.New("insufficient number of public keys"), 1)
	}

	args := []string(ctx.Args())
	pubs := make(keys.PublicKeys, len(args))

	for i := range args {
		pubs[i], err = keys.NewPublicKeyFromString(args[i])
		if err != nil {
			return cli.NewExitError(fmt.Errorf("can't decode public key %d: %w", i, err), 1)
		}
	}
	script, err := pubs.CreateMultiSigVerificationScript(m)
	if err != nil {
		return cli.NewExitError(fmt.Errorf("can't create multisig verification script: %w", err), 1)
	}
	address := hash.Hash160(script)
	acc := &wallet.Account{
		Script:  script,
		Address: address,
	}
	if acc.Label == "" {
		acc.Label = ctx.String("name")
	}
	if err := addAccountAndSave(wall, acc); err != nil {
		return cli.NewExitError(err, 1)
	}
	fmt.Fprintf(ctx.App.Writer, "Multisig. Addr.: %s \n", address)
	return nil
}

func removeAccount(ctx *cli.Context) error {
	wall, err := openWallet(ctx.String("wallet"))
	if err != nil {
		return cli.NewExitError(err, 1)
	}
	defer wall.Close()

	addr := ctx.Generic("address").(*flags.Address)
	if !addr.IsSet {
		return cli.NewExitError("valid account address must be provided", 1)
	}
	acc := wall.GetAccount(addr.Address())
	if acc == nil {
		return cli.NewExitError("account wasn't found", 1)
	}

	if !ctx.Bool("force") {
		fmt.Fprintf(ctx.App.Writer, "Account %s will be removed. This action is irreversible.\n", addr.Address())
		if ok := askForConsent(ctx.App.Writer); !ok {
			return nil
		}
	}

	if err := wall.RemoveAccount(acc.Address.String()); err != nil {
		return cli.NewExitError(fmt.Errorf("error on remove: %w", err), 1)
	}
	if err := wall.Save(); err != nil {
		return cli.NewExitError(fmt.Errorf("error while saving wallet: %w", err), 1)
	}
	return nil
}

func askForConsent(w io.Writer) bool {
	response, err := input.ReadLine("Are you sure? [y/N]: ")
	if err == nil {
		response = strings.ToLower(strings.TrimSpace(response))
		if response == "y" || response == "yes" {
			return true
		}
	}
	fmt.Fprintln(w, "Cancelled.")
	return false
}

func dumpKeys(ctx *cli.Context) error {
	wall, err := ReadWallet(ctx.String("wallet"))
	if err != nil {
		return cli.NewExitError(err, 1)
	}
	accounts := wall.Accounts

	addrFlag := ctx.Generic("address").(*flags.Address)
	if addrFlag.IsSet {
		acc := wall.GetAccount(addrFlag.Address())
		if acc == nil {
			return cli.NewExitError("can't find address", 1)
		}
		accounts = []*wallet.Account{acc}
	}

	hasPrinted := false
	for _, acc := range accounts {
		if hasPrinted {
			fmt.Fprintln(ctx.App.Writer)
		}
		if acc.IsMultiSig() {
			fmt.Println("multiple signature contract:")
			fmt.Fprintf(ctx.App.Writer, "address: %s \n", acc.Address)
			fmt.Fprintf(ctx.App.Writer, "script: %s \n", hex.EncodeToString(acc.Script))
		} else {
			fmt.Println("simple signature contract:")
			fmt.Fprintf(ctx.App.Writer, "address: %s \n", acc.Address)
			fmt.Fprintf(ctx.App.Writer, "public key: %s \n", hex.EncodeToString((acc.Script)[1:]))
		}
		hasPrinted = true
	}
	return nil
}

func createWallet(ctx *cli.Context) error {
	path := ctx.String("wallet")
	if len(path) == 0 {
		return cli.NewExitError(errNoPath, 1)
	}
	wall, err := wallet.NewWallet(path)
	if err != nil {
		return cli.NewExitError(err, 1)
	}
	if err := wall.Save(); err != nil {
		return cli.NewExitError(err, 1)
	}

	if ctx.Bool("account") {
		if err := createAccount(wall); err != nil {
			return cli.NewExitError(err, 1)
		}
	}

	fmtPrintWallet(ctx.App.Writer, wall)
	fmt.Fprintf(ctx.App.Writer, "wallet successfully created, file location is %s\n", wall.Path())
	return nil
}

func readAccountInfo() (string, string, error) {
	name, err := input.ReadLine("Enter the name of the account > ")
	if err != nil {
		return "", "", err
	}
	phrase, err := readNewPassword()
	if err != nil {
		return "", "", err
	}
	return name, phrase, nil
}

func readNewPassword() (string, error) {
	phrase, err := input.ReadPassword("Enter passphrase > ")
	if err != nil {
		return "", fmt.Errorf("error reading password: %w", err)
	}
	phraseCheck, err := input.ReadPassword("Confirm passphrase > ")
	if err != nil {
		return "", fmt.Errorf("error reading password: %w", err)
	}

	if phrase != phraseCheck {
		return "", errPhraseMismatch
	}
	return phrase, nil
}

func createAccount(wall *wallet.Wallet) error {
	name, phrase, err := readAccountInfo()
	if err != nil {
		return err
	}
	return wall.CreateAccount(name, phrase)
}

func openWallet(path string) (*wallet.Wallet, error) {
	if len(path) == 0 {
		return nil, errNoPath
	}
	if path == "-" {
		return nil, errNoStdin
	}
	return wallet.NewWalletFromFile(path)
}

func ReadWallet(path string) (*wallet.Wallet, error) {
	if len(path) == 0 {
		return nil, errNoPath
	}
	if path == "-" {
		w := &wallet.Wallet{}
		if err := json.NewDecoder(os.Stdin).Decode(w); err != nil {
			return nil, fmt.Errorf("js %s", err)
		}
		return w, nil
	}
	return wallet.NewWalletFromFile(path)
}

func addAccountAndSave(w *wallet.Wallet, acc *wallet.Account) error {
	for i := range w.Accounts {
		if w.Accounts[i].Address == acc.Address {
			return fmt.Errorf("address '%s' is already in wallet", acc.Address)
		}
	}

	w.AddAccount(acc)
	return w.Save()
}

func fmtPrintWallet(w io.Writer, wall *wallet.Wallet) {
	b, _ := wall.JSON()
	fmt.Fprintln(w, "")
	fmt.Fprintln(w, string(b))
	fmt.Fprintln(w, "")
}

func sign(ctx *cli.Context) error {
	wall, err := ReadWallet(ctx.String("wallet"))
	if err != nil {
		return cli.NewExitError(err, 1)
	}
	defer wall.Close()

	signContext := new(SignContext)
	err = json.Unmarshal([]byte(ctx.String("context")), signContext)
	if err != nil {
		return cli.NewExitError(fmt.Errorf("sign context invalid: %w", err), 1)
	}

	err = Sign(wall, signContext)
	if err != nil {
		return cli.NewExitError(fmt.Errorf("sign context error: %w", err), 1)
	}
	var tx *transaction.Transaction
	if signContext.IsComplete() {
		tx, err = signContext.CreateTx()
		if err != nil {
			return cli.NewExitError(fmt.Errorf("failed to create tx: %w", err), 1)
		}
	} else {
		b, err := json.Marshal(*signContext)
		if err != nil {
			return cli.NewExitError(fmt.Errorf("failed marshal sign context json: %w", err), 1)
		}
		fmt.Fprintf(ctx.App.Writer, "SignContext: %s\n", string(b))
		return nil
	}
	b, err := tx.Bytes()
	if err != nil {
		return cli.NewExitError(fmt.Errorf("failed encode tx: %w", err), 1)
	}
	err = input.ConfirmTx()
	if err != nil {
		fmt.Fprintln(ctx.App.Writer, hex.EncodeToString(b[1:]))
		return cli.NewExitError(err, 1)
	}
	gctx, cancel := options.GetTimeoutContext(ctx)
	defer cancel()
	c, err := options.GetRPCClient(gctx, ctx)
	hash, err := c.SendRawTransaction(b[1:])
	if err != nil {
		return cli.NewExitError(fmt.Errorf("failed relay tx: %w", err), 1)
	}
	fmt.Fprintf(ctx.App.Writer, "TxHash: %s\n", hash)
	return nil
}

func MakeSaiTx(ctx *cli.Context, wall *wallet.Wallet, from common.Address, to common.Address, value *big.Int, data []byte) error {
	var err error
	var pks *keys.PublicKeys
	var script []byte
	isMulti := false
	m := 0
	signers := []*wallet.Account{}
	for _, acc := range wall.Accounts {
		if acc.Address == from {
			isMulti = acc.IsMultiSig()
			signers = append(signers, acc)
			break
		}
	}
	gctx, cancel := options.GetTimeoutContext(ctx)
	defer cancel()
	c, err := options.GetRPCClient(gctx, ctx)
	if err != nil {
		return cli.NewExitError(err, 1)
	}
	if len(signers) == 0 {
		committeeAddr, err := c.GetCommitteeAddress()
		if err != nil {
			return cli.NewExitError(fmt.Errorf("can't get committee address: %w", err), 1)
		}
		if from != committeeAddr {
			return cli.NewExitError("can't find account to sign", 1)
		}
		committee, err := c.GetCommittee()
		if err != nil {
			return cli.NewExitError(fmt.Errorf("failed get committee: %w", err), 1)
		}
		if committee.Len() == 1 {
			isMulti = false
			for _, acc := range wall.Accounts {
				if acc.Address == committeeAddr {
					signers = append(signers, acc)
					break
				}
			}

		} else {
			isMulti = true
			pks = &committee
			m = keys.GetMajorityHonestNodeCount(pks.Len())
			script, err = committee.CreateMajorityMultiSigRedeemScript()
			if err != nil {
				return cli.NewExitError(fmt.Errorf("can't create committee multi-sig script: %w", err), 1)
			}
		}

	}
	if len(signers) == 0 {
		return cli.NewExitError("can't find account to sign", 1)
	}
	if isMulti {
		if pks == nil {
			script = signers[0].Script
			pks, m, err = crypto.ParseMultiVerificationScript(signers[0].Script)
			if err != nil {
				return cli.NewExitError(fmt.Errorf("can't parse multi-sig account script: %w", err), 1)
			}
			signers = signers[1:]
		}
		for _, ac := range wall.Accounts {
			if ac.IsMultiSig() {
				continue
			}
			pk, err := crypto.ParseVerificationScript(ac.Script)
			if err != nil {
				return cli.NewExitError(fmt.Errorf("can't parse account script: %w", err), 1)
			}
			if pks.Contains(pk) {
				signers = append(signers, ac)
			}
			if len(signers) >= m {
				break
			}
		}
	} else {
		script = signers[0].Script
	}
	if len(signers) == 0 {
		return cli.NewExitError("can't find account to sign", 1)
	}
	for _, acc := range signers {
		pass, err := input.ReadPassword(fmt.Sprintf("Enter %s password > ", acc.Address))
		if err != nil {
			return cli.NewExitError(fmt.Errorf("error reading password: %w", err), 1)
		}
		err = acc.Decrypt(pass, wall.Scrypt)
		if err != nil {
			return cli.NewExitError(fmt.Errorf("unable to decrypt account: %s", acc.Address), 1)
		}
	}
	nonce, err := c.Eth_GetTransactionCount(from)
	if err != nil {
		return cli.NewExitError(fmt.Errorf("failed get account nonce: %w", err), 1)
	}
	gasPrice, err := c.Eth_GasPrice()
	if err != nil {
		return cli.NewExitError(fmt.Errorf("failed get fee per byte: %w", err), 1)
	}
	t := &transaction.SaiTx{
		Nonce:    nonce,
		GasPrice: gasPrice,
		Gas:      0,
		From:     from,
		To:       &to,
		Value:    value,
		Data:     data,
		Witness: transaction.Witness{
			VerificationScript: script,
		},
	}
	tx := transaction.NewTx(t)
	gas, err := c.CalculateGas(t)
	if err != nil {
		return cli.NewExitError(fmt.Errorf("failed estimate gas fee: %w", err), 1)
	}
	t.Gas = gas
	chainId, err := c.Eth_ChainId()
	if err != nil {
		return cli.NewExitError(fmt.Errorf("failed to get chainId: %w", err), 1)
	}
	if !isMulti {
		signers[0].SignTx(chainId, tx)
	} else {
		signContext := SignContext{
			ChainID:    chainId,
			Tx:         *t,
			Parameters: make(map[string][]byte),
			M:          m,
		}
		for _, acc := range signers {
			for _, a := range *pks {
				if acc.Address == a.Address() {
					signContext.Parameters[hex.EncodeToString(a.Bytes())] = acc.PrivateKey().SignHashable(chainId, t)
				}
			}
		}
		if signContext.IsComplete() {
			tx, err = signContext.CreateTx()
			if err != nil {
				return cli.NewExitError(fmt.Errorf("failed to create tx: %w", err), 1)
			}
		} else {
			b, err := json.Marshal(signContext)
			if err != nil {
				return cli.NewExitError(fmt.Errorf("failed marshal sign context json: %w", err), 1)
			}
			fmt.Fprintf(ctx.App.Writer, "SignContext: %s\n", string(b))
			return nil
		}
	}
	b, err := tx.SaiTx.Bytes()
	if err != nil {
		return cli.NewExitError(fmt.Errorf("failed encode tx to bytes: %w", err), 1)
	}
	hash, err := c.SendRawTransaction(b)
	if err != nil {
		return cli.NewExitError(fmt.Errorf("failed relay tx: %w", err), 1)
	}
	fmt.Fprintf(ctx.App.Writer, "TxHash: %s\n", hash)
	return nil
}

func MakeEthTx(ctx *cli.Context, facc *wallet.Account, to *common.Address, value *big.Int, data []byte) error {
	var err error
	gctx, cancel := options.GetTimeoutContext(ctx)
	defer cancel()
	c, err := options.GetRPCClient(gctx, ctx)
	if err != nil {
		return cli.NewExitError(err, 1)
	}
	chainId, err := c.Eth_ChainId()
	if err != nil {
		return cli.NewExitError(fmt.Errorf("failed to get chainId: %w", err), 1)
	}
	gasPrice, err := c.Eth_GasPrice()
	if err != nil {
		return cli.NewExitError(err, 1)
	}
	nonce, err := c.Eth_GetTransactionCount(facc.Address)
	if err != nil {
		return err
	}
	ltx := &types.LegacyTx{
		Nonce:    nonce,
		To:       to,
		GasPrice: gasPrice,
		Value:    value,
		Data:     data,
	}
	tx := &transaction.EthTx{
		Transaction: *types.NewTx(ltx),
	}
	gas, err := c.Eth_EstimateGas(&result.TransactionObject{
		From:     facc.Address,
		To:       tx.To(),
		GasPrice: tx.GasPrice(),
		Value:    tx.Value(),
		Data:     tx.Data(),
	})
	if err != nil {
		return err
	}
	ltx.Gas = gas
	tx.Transaction = *types.NewTx(ltx)
	err = facc.SignTx(chainId, transaction.NewTx(tx))
	if err != nil {
		return cli.NewExitError(fmt.Errorf("can't sign tx: %w", err), 1)
	}
	b, err := tx.Transaction.MarshalBinary()
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

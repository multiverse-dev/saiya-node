package wallet

import (
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/multiverse-dev/saiya/cli/flags"
	"github.com/multiverse-dev/saiya/cli/input"
	"github.com/multiverse-dev/saiya/cli/options"
	"github.com/multiverse-dev/saiya/pkg/crypto/keys"
	"github.com/multiverse-dev/saiya/pkg/encoding/address"
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
				Name:   "dump",
				Usage:  "check and dump an existing saiya wallet",
				Action: dumpWallet,
				Flags: []cli.Flag{
					WalletPathFlag,
					decryptFlag,
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
				UsageText: "list --wallet <path>",
				Action:    listAddresses,
				Flags:     listFlags,
			},
			{
				Name:        "gas",
				Usage:       "work with native gas",
				Subcommands: newNativeTokenCommands(),
			},
			{
				Name:   "sign",
				Usage:  "sign sign_context",
				Action: sign,
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
		fmt.Fprintf(ctx.App.Writer, "%s SAIYA: %s\n", acc.Address, bal)
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

func dumpWallet(ctx *cli.Context) error {
	wall, err := ReadWallet(ctx.String("wallet"))
	if err != nil {
		return cli.NewExitError(err, 1)
	}
	if ctx.Bool("decrypt") {
		pass, err := input.ReadPassword("Enter wallet password > ")
		if err != nil {
			return cli.NewExitError(fmt.Errorf("Error reading password: %w", err), 1)
		}
		for i := range wall.Accounts {
			// Just testing the decryption here.
			err := wall.Accounts[i].Decrypt(pass, wall.Scrypt)
			if err != nil {
				return cli.NewExitError(err, 1)
			}
		}
	}
	fmtPrintWallet(ctx.App.Writer, wall)
	return nil
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
			return cli.NewExitError("account is missing", 1)
		}
		accounts = []*wallet.Account{acc}
	}

	hasPrinted := false
	for _, acc := range accounts {
		if hasPrinted {
			fmt.Fprintln(ctx.App.Writer)
		}
		fmt.Fprintf(ctx.App.Writer, "%s (simple signature contract):\n", acc.Address)
		fmt.Fprintln(ctx.App.Writer, hex.EncodeToString(acc.PublicKey))
		hasPrinted = true
		if addrFlag.IsSet {
			return cli.NewExitError(fmt.Errorf("unknown script type for address %s", address.AddressToBase58(addrFlag.Address())), 1)
		}
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
	return nil
}

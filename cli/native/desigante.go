package native

import (
	"fmt"
	"sort"

	"github.com/multiverse-dev/saiya/cli/options"
	"github.com/multiverse-dev/saiya/cli/wallet"
	"github.com/multiverse-dev/saiya/pkg/core/native"
	"github.com/multiverse-dev/saiya/pkg/core/native/nativenames"
	"github.com/multiverse-dev/saiya/pkg/core/native/noderoles"
	"github.com/multiverse-dev/saiya/pkg/crypto/keys"
	"github.com/urfave/cli"
)

func newDesignateCommands() []cli.Command {
	designateFlags := append(options.RPC, wallet.WalletPathFlag)
	return []cli.Command{
		{
			Name:      "committee",
			Usage:     "designate committee",
			ArgsUsage: "<publicKey> <publicKey> ...",
			Action:    designateCommittee,
			Flags:     designateFlags,
		},
		{
			Name:      "statevalidators",
			Usage:     "designate state validators",
			ArgsUsage: "<publicKey> <publicKey> ...",
			Action:    designateStateValidators,
			Flags:     designateFlags,
		},
	}
}

func designateCommittee(ctx *cli.Context) error {
	return designate(ctx, noderoles.Committee)
}

func designateStateValidators(ctx *cli.Context) error {
	return designate(ctx, noderoles.StateValidator)
}

func designate(ctx *cli.Context, role noderoles.Role) error {
	args := ctx.Args()
	newCommittees := make(keys.PublicKeys, len(args))
	for i, arg := range args {
		p, err := keys.NewPublicKeyFromString(arg)
		if err != nil {
			return cli.NewExitError(fmt.Errorf("failed parse public key: %s", arg), 1)
		}
		newCommittees[i] = p
	}
	newCommittees = newCommittees.Unique()
	sort.Sort(newCommittees)
	if newCommittees.Len() == 0 {
		return cli.NewExitError(fmt.Errorf("please input public keys"), 1)
	}
	if newCommittees.Len() > native.MaxNodeCount {
		return cli.NewExitError(fmt.Errorf("too many public keys"), 1)
	}
	return callNative(ctx, nativenames.Designation, "designateAsRole", uint8(role), (&newCommittees).Bytes())
}

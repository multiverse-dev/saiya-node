package main

import (
	"os"

	"github.com/multiverse-dev/saiya/cli/native"
	"github.com/multiverse-dev/saiya/cli/query"
	"github.com/multiverse-dev/saiya/cli/server"
	"github.com/multiverse-dev/saiya/cli/utils"
	"github.com/multiverse-dev/saiya/cli/vm"
	"github.com/multiverse-dev/saiya/cli/wallet"
	"github.com/multiverse-dev/saiya/pkg/config"
	"github.com/urfave/cli"
)

func main() {
	ctl := newApp()

	if err := ctl.Run(os.Args); err != nil {
		panic(err)
	}
}

func newApp() *cli.App {
	ctl := cli.NewApp()
	ctl.Name = "saiya"
	ctl.Version = config.Version
	ctl.Usage = "Official Go client for multiverse"
	ctl.ErrWriter = os.Stdout

	ctl.Commands = append(ctl.Commands, server.NewCommands()...)
	ctl.Commands = append(ctl.Commands, wallet.NewCommands()...)
	ctl.Commands = append(ctl.Commands, vm.NewCommands()...)
	ctl.Commands = append(ctl.Commands, query.NewCommands()...)
	ctl.Commands = append(ctl.Commands, native.NewCommands()...)
	ctl.Commands = append(ctl.Commands, utils.NewCommands()...)
	return ctl
}

// Copyright 2019 ChainSafe Systems (ON) Corp.
// This file is part of gossamer.
//
// The gossamer library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The gossamer library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the gossamer library. If not, see <http://www.gnu.org/licenses/>.

package main

import (
	"os"
	"strconv"

	"github.com/ChainSafe/gossamer/cmd/utils"
	log "github.com/ChainSafe/log15"
	"github.com/urfave/cli"
)

var (
	app       = cli.NewApp()
	nodeFlags = []cli.Flag{
		utils.DataDirFlag,
		configFileFlag,
	}
	p2pFlags = []cli.Flag{
		utils.BootnodesFlag,
		utils.P2pPortFlag,
		utils.NoBootstrapFlag,
		utils.NoMdnsFlag,
	}
	rpcFlags = []cli.Flag{
		utils.RpcEnabledFlag,
		utils.RpcHostFlag,
		utils.RpcPortFlag,
		utils.RpcModuleFlag,
	}
	genesisFlags = []cli.Flag{
		utils.GenesisFlag,
	}
	cliFlags = []cli.Flag{
		utils.VerbosityFlag,
	}
)

var (
	dumpConfigCommand = cli.Command{
		Action:      dumpConfig,
		Name:        "dumpconfig",
		Usage:       "Show configuration values",
		ArgsUsage:   "",
		Flags:       append(append(nodeFlags, rpcFlags...)),
		Category:    "CONFIGURATION DEBUGGING",
		Description: `The dumpconfig command shows configuration values.`,
	}
	initCommand = cli.Command{
		Action:    MigrateFlags(initNode),
		Name:      "init",
		Usage:     "Initialize node genesis state",
		ArgsUsage: "",
		Flags: []cli.Flag{
			utils.DataDirFlag,
			utils.GenesisFlag,
			utils.VerbosityFlag,
			configFileFlag,
		},
		Category:    "INITIALIZATION",
		Description: `The init command initializes the node with a genesis state. Usage: gossamer init --genesis genesis.json`,
	}
	configFileFlag = cli.StringFlag{
		Name:  "config",
		Usage: "TOML configuration file",
	}
)

// init initializes CLI
func init() {
	app.Action = gossamer
	app.Copyright = "Copyright 2019 ChainSafe Systems Authors"
	app.Name = "gossamer"
	app.Usage = "Official gossamer command-line interface"
	app.Author = "ChainSafe Systems 2019"
	app.Version = "0.0.1"
	app.Commands = []cli.Command{
		dumpConfigCommand,
		initCommand,
	}
	app.Flags = append(app.Flags, nodeFlags...)
	app.Flags = append(app.Flags, p2pFlags...)
	app.Flags = append(app.Flags, rpcFlags...)
	app.Flags = append(app.Flags, genesisFlags...)
	app.Flags = append(app.Flags, cliFlags...)
}

func main() {
	if err := app.Run(os.Args); err != nil {
		os.Exit(1)
	}
}

func startLogger(ctx *cli.Context) error {
	logger := log.Root()
	handler := logger.GetHandler()
	var lvl log.Lvl

	if lvlToInt, err := strconv.Atoi(ctx.String(utils.VerbosityFlag.Name)); err == nil {
		lvl = log.Lvl(lvlToInt)
	} else if lvl, err = log.LvlFromString(ctx.String(utils.VerbosityFlag.Name)); err != nil {
		return err
	}
	log.Root().SetHandler(log.LvlFilterHandler(lvl, handler))

	return nil
}

// initNode loads the genesis file and loads the initial state into the DB
func initNode(ctx *cli.Context) error {
	err := startLogger(ctx)
	if err != nil {
		return err
	}

	err = loadGenesis(ctx)
	if err != nil {
		log.Error("error loading genesis state", "error", err)
		return err
	}

	log.Info("🕸\t Finished initializing node!")
	return nil
}

// MigrateFlags sets the global flag from a local flag when it's set.
func MigrateFlags(action func(ctx *cli.Context) error) func(*cli.Context) error {
	return func(ctx *cli.Context) error {
		for _, name := range ctx.FlagNames() {
			if ctx.IsSet(name) {
				err := ctx.GlobalSet(name, ctx.String(name))
				if err != nil {
					return nil
				}
			}
		}
		return action(ctx)
	}
}

// gossamer is the main entrypoint into the gossamer system
func gossamer(ctx *cli.Context) error {
	err := startLogger(ctx)
	if err != nil {
		return err
	}

	node, _, err := makeNode(ctx)
	if err != nil {
		log.Error("error starting gossamer", "err", err)
		return err
	}

	log.Info("🕸️\t Starting node...", "name", node.Name)
	node.Start()

	return nil
}

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

package cfg

import (
	"os"
	"os/user"
	"path/filepath"
	"runtime"

	"github.com/ChainSafe/gossamer/internal/api"
)

const (
	DefaultRpcHttpHost = "localhost" // Default host interface for the HTTP RPC server
	DefaultRpcHttpPort = 8545        // Default port for

	// P2P
	DefaultP2PPort = 7001

	DefaultGenesisPath = "./genesis.json"
)

var (
	// Must be non-nil to match toml parsing semantics
	DefaultP2PBootstrap = []string{}
	DefaultRpcModules   = []api.Module{"system"}
)

var (
	// Global
	DefaultGlobalConfig = GlobalConfig{
		DataDir: DefaultDataDir(),
	}

	// P2P
	DefaultP2PConfig = P2pCfg{
		Port:           DefaultP2PPort,
		BootstrapNodes: DefaultP2PBootstrap,
		NoBootstrap:    false,
		NoMdns:         false,
	}

	// RPC
	DefaultRpcConfig = RpcCfg{
		Host:    DefaultRpcHttpHost,
		Port:    DefaultRpcHttpPort,
		Modules: DefaultRpcModules,
	}
)

// DefaultConfig is the default settings used when a config.toml file is not passed in during instantiation
func DefaultConfig() *Config {
	return &Config{
		Global: DefaultGlobalConfig,
		P2p:    DefaultP2PConfig,
		Rpc:    DefaultRpcConfig,
	}
}

// DefaultDataDir is the default data directory to use for the databases and other
// persistence requirements.
func DefaultDataDir() string {
	// Try to place the data folder in the user's home dir
	home := homeDir()
	if home != "" {
		if runtime.GOOS == "darwin" {
			return filepath.Join(home, "Library", "Gossamer")
		} else if runtime.GOOS == "windows" {
			return filepath.Join(home, "AppData", "Roaming", "Gossamer")
		} else {
			return filepath.Join(home, ".gossamer")
		}
	}
	// As we cannot guess a stable location, return empty and handle later
	return ""
}

func homeDir() string {
	if home := os.Getenv("HOME"); home != "" {
		return home
	}
	if usr, err := user.Current(); err == nil {
		return usr.HomeDir
	}
	return ""
}

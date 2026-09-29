package app_test

import (
	"flag"

	simcli "github.com/cosmos/cosmos-sdk/x/simulation/client/cli"
)

func init() {
	if flag.Lookup("Enabled") == nil {
		simcli.GetSimulatorFlags()
	}
}

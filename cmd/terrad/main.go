package main

import (
	"os"

	svrcmd "github.com/cosmos/cosmos-sdk/server/cmd"

	terraapp "github.com/classic-terra/core/v3/app"
)

func main() {
	rootCmd, _ := NewRootCmd()

	if err := svrcmd.Execute(rootCmd, "", terraapp.DefaultNodeHome); err != nil {
		os.Exit(1)
	}
}

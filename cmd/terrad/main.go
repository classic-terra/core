package main

import (
	"os"

	terraapp "github.com/classic-terra/core/v4/app"
	svrcmd "github.com/cosmos/cosmos-sdk/server/cmd"
)

func init() {}

func main() {
	rootCmd, _ := NewRootCmd()

	if err := svrcmd.Execute(rootCmd, "", terraapp.DefaultNodeHome); err != nil {
		os.Exit(1)
	}
}

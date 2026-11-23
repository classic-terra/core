package main

import (
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"
)

func TestParseWasmAdminOverrides(t *testing.T) {
	sdk.GetConfig().SetBech32PrefixForAccount("terra", "terrapub")

	contract := "terra1anagcnqamh6jfym8emajk8f6kk0uvjusvzss9cx55t4k4js4rjrs3x8t98"
	contract2 := "terra13w6cruzc0xkdhpmrnspr5j282x3wyx0m4f6kjnr65tpq63usn8eqp5emy4"
	admin := "terra1yq3d4h9g4ncale3mcwuhcfdge8hzjx666umlhg"

	got, err := parseWasmAdminOverrides([]string{contract + "=" + admin, " " + contract2 + " = " + admin})
	require.NoError(t, err)
	require.Len(t, got, 2)
	require.Equal(t, contract, got[0].Contract.String())
	require.Equal(t, admin, got[0].NewAdmin.String())
	require.Equal(t, contract2, got[1].Contract.String())

	got, err = parseWasmAdminOverrides(nil)
	require.NoError(t, err)
	require.Empty(t, got)

	for name, entries := range map[string][]string{
		"missing separator": {contract},
		"bad contract":      {"nope=" + admin},
		"bad admin":         {contract + "=nope"},
		"empty admin":       {contract + "="},
		"wrong prefix":      {contract + "=cosmos1yq3d4h9g4ncale3mcwuhcfdge8hzjx66x3gd3l"},
		"duplicate":         {contract + "=" + admin, contract + "=" + admin},
	} {
		_, err := parseWasmAdminOverrides(entries)
		require.Error(t, err, name)
	}
}

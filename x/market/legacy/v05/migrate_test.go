package v05_test

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/cosmos/cosmos-sdk/client"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/classic-terra/core/v2/app"
	core "github.com/classic-terra/core/v2/types"
	v04market "github.com/classic-terra/core/v2/x/market/legacy/v04"
	v05market "github.com/classic-terra/core/v2/x/market/legacy/v05"
)

func TestMigrate(t *testing.T) {
	sdk.GetConfig().SetBech32PrefixForAccount(core.Bech32PrefixAccAddr, core.Bech32PrefixAccPub)
	encodingConfig := app.MakeEncodingConfig()
	clientCtx := client.Context{}.
		WithInterfaceRegistry(encodingConfig.InterfaceRegistry).
		WithTxConfig(encodingConfig.TxConfig).
		WithLegacyAmino(encodingConfig.Amino).
		WithCodec(encodingConfig.Marshaler)

	marketGenState := v04market.GenesisState{
		TerraPoolDelta: sdk.ZeroDec(),
		Params: v04market.Params{
			BasePool:           sdk.NewDec(1000000),
			PoolRecoveryPeriod: int64(10000),
			MinStabilitySpread: sdk.NewDecWithPrec(2, 2), // ATTENTION: The list of modes must be in alphabetical order, otherwise an error occurs in validateMaxSupplyCoin => !v.IsValid()
			MaxSupplyCoin: sdk.Coins{
				{Denom: "uaud", Amount: sdk.NewInt(500000000000)},
				{Denom: "ucad", Amount: sdk.NewInt(500000000000)},
				{Denom: "uchf", Amount: sdk.NewInt(500000000000)},
				{Denom: "ucny", Amount: sdk.NewInt(500000000000)},
				{Denom: "udkk", Amount: sdk.NewInt(500000000000)},
				{Denom: "ueur", Amount: sdk.NewInt(500000000000)},
				{Denom: "ugbp", Amount: sdk.NewInt(500000000000)},
				{Denom: "uhkd", Amount: sdk.NewInt(500000000000)},
				{Denom: "uidr", Amount: sdk.NewInt(500000000000)},
				{Denom: "uinr", Amount: sdk.NewInt(500000000000)},
				{Denom: "ujpy", Amount: sdk.NewInt(500000000000)},
				{Denom: "ukrw", Amount: sdk.NewInt(500000000000)},
				{Denom: "uluna", Amount: sdk.NewInt(1000000000000)},
				{Denom: "umnt", Amount: sdk.NewInt(500000000000)},
				{Denom: "unok", Amount: sdk.NewInt(500000000000)},
				{Denom: "uphp", Amount: sdk.NewInt(500000000000)},
				{Denom: "usdr", Amount: sdk.NewInt(500000000000)},
				{Denom: "usek", Amount: sdk.NewInt(500000000000)},
				{Denom: "usgd", Amount: sdk.NewInt(500000000000)},
				{Denom: "uthb", Amount: sdk.NewInt(500000000000)},
				{Denom: "uusd", Amount: sdk.NewInt(500000000000)},
			},
			PercentageSupplyMaxDescending: sdk.NewDecWithPrec(30, 2), // 30%
		},
	}

	migrated := v05market.Migrate(marketGenState)

	bz, err := clientCtx.Codec.MarshalJSON(migrated)
	require.NoError(t, err)

	// Indent the JSON bz correctly.
	var jsonObj map[string]interface{}
	err = json.Unmarshal(bz, &jsonObj)
	require.NoError(t, err)
	indentedBz, err := json.MarshalIndent(jsonObj, "", "\t")
	require.NoError(t, err)

	// Make sure about:
	// - BasePool to Mint & Burn pool
	expected := `{
		"params": {
		  "base_pool": "1000000.000000000000000000",
		  "min_stability_spread": "0.020000000000000000",
		  "pool_recovery_period": "10000",
		  "max_supply_coin": [
			{
			  "denom": "uaud",
			  "amount": "500000000000"
			},
			{
			  "denom": "ucad",
			  "amount": "500000000000"
			},
			{
			  "denom": "uchf",
			  "amount": "500000000000"
			},
			{
			  "denom": "ucny",
			  "amount": "500000000000"
			},
			{
			  "denom": "udkk",
			  "amount": "500000000000"
			},
			{
			  "denom": "ueur",
			  "amount": "500000000000"
			},
			{
			  "denom": "ugbp",
			  "amount": "500000000000"
			},
			{
				"denom": "uhkd",
				"amount": "500000000000"
			},
			{
			  "denom": "uidr",
			  "amount": "500000000000"
			},
			{
			  "denom": "uinr",
			  "amount": "500000000000"
			},
			{
			  "denom": "ujpy",
			  "amount": "500000000000"
			},
			{
			  "denom": "ukrw",
			  "amount": "500000000000"
			},
			{
			  "denom": "uluna",
			  "amount": "1000000000000"
			},
			{
			  "denom": "umnt",
			  "amount": "500000000000"
			},
			{
			  "denom": "unok",
			  "amount": "500000000000"
			},
			{
			  "denom": "uphp",
			  "amount": "500000000000"
			},
			{
			  "denom": "usdr",
			  "amount": "500000000000"
			},
			{
			  "denom": "usek",
			  "amount": "500000000000"
			},
			{
			  "denom": "usgd",
			  "amount": "500000000000"
			},
			{
			  "denom": "uthb",
			  "amount": "500000000000"
			},
			{
			  "denom": "uusd",
			  "amount": "500000000000"
			}
		  ],
		  "percentage_supply_max_descending": "0.300000000000000000"
		},
		"terra_pool_delta": "0.000000000000000000"
	  }`

	assert.JSONEq(t, expected, string(indentedBz))
}

package keeper

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/terra-money/core/x/wasm/config"
	"github.com/terra-money/core/x/wasm/types"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

func TestInstantiateExceedMaxGas(t *testing.T) {
	input := CreateTestInput(t, config.DefaultConfig())
	ctx, keeper := input.Ctx, input.WasmKeeper

	exampleContract := StoreExampleContract(t, input, "./testdata/hackatom.wasm")

	_, _, bob := keyPubAddr()
	_, _, fred := keyPubAddr()

	initMsg := HackatomExampleInitMsg{
		Verifier:    fred,
		Beneficiary: bob,
	}

	initMsgBz, err := json.Marshal(initMsg)
	require.NoError(t, err)

	// must panic
	require.Panics(t, func() {
		params := keeper.GetParams(ctx)
		params.MaxContractGas = types.InstantiateContractCosts(0) + 1
		keeper.SetParams(ctx, params)
		NewMsgServerImpl(keeper).InstantiateContract(ctx.Context(), types.NewMsgInstantiateContract(exampleContract.CreatorAddr, sdk.AccAddress{}, exampleContract.CodeID, initMsgBz, nil))
	})
}

func TestExecuteExceedMaxGas(t *testing.T) {
	input := CreateTestInput(t, config.DefaultConfig())
	ctx, keeper := input.Ctx, input.WasmKeeper

	exampleContract := StoreExampleContract(t, input, "./testdata/hackatom.wasm")

	_, _, bob := keyPubAddr()
	_, _, fred := keyPubAddr()

	initMsg := HackatomExampleInitMsg{
		Verifier:    fred,
		Beneficiary: bob,
	}

	initMsgBz, err := json.Marshal(initMsg)
	require.NoError(t, err)

	addr, _, err := keeper.InstantiateContract(ctx, exampleContract.CodeID, exampleContract.CreatorAddr, sdk.AccAddress{}, initMsgBz, nil)

	// must panic
	require.Panics(t, func() {
		params := keeper.GetParams(ctx)
		params.MaxContractGas = types.InstantiateContractCosts(0) + 1
		keeper.SetParams(ctx, params)
		NewMsgServerImpl(keeper).ExecuteContract(ctx.Context(), types.NewMsgExecuteContract(exampleContract.CreatorAddr, addr, []byte(`{"release":{}}`), nil))
	})
}

func TestMigrateExceedMaxGas(t *testing.T) {
	input := CreateTestInput(t, config.DefaultConfig())
	ctx, keeper := input.Ctx, input.WasmKeeper

	exampleContract := StoreExampleContract(t, input, "./testdata/hackatom.wasm")

	_, _, bob := keyPubAddr()
	_, _, fred := keyPubAddr()

	initMsg := HackatomExampleInitMsg{
		Verifier:    fred,
		Beneficiary: bob,
	}

	initMsgBz, err := json.Marshal(initMsg)
	require.NoError(t, err)

	addr, _, err := keeper.InstantiateContract(ctx, exampleContract.CodeID, exampleContract.CreatorAddr, sdk.AccAddress{}, initMsgBz, nil)

	// must panic
	require.Panics(t, func() {
		params := keeper.GetParams(ctx)
		params.MaxContractGas = types.InstantiateContractCosts(0) + 1
		keeper.SetParams(ctx, params)
		NewMsgServerImpl(keeper).MigrateContract(ctx.Context(), types.NewMsgMigrateContract(exampleContract.CreatorAddr, addr, exampleContract.CodeID, []byte(`{"release":{}}`)))
	})
}

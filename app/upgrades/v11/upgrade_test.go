//nolint:revive
package v11

import (
	"testing"

	"github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	"github.com/stretchr/testify/require"
)

func TestJustBurnItAlready(t *testing.T) {
	input := CreateTestInput(t)
	address := types.AccAddress("cosmos1gr0xesnseevzt3h4nxr64sh5gk4dwrwgszx3nw")
	acc := authtypes.NewBaseAccount(address, nil, 0, 0)
	input.AccountKeeper.NewAccount(input.Ctx, acc)
	uusdCoins := types.NewCoins(types.NewCoin("uusd", types.NewInt(1000000)))

	err := input.BankKeeper.MintCoins(input.Ctx, banktypes.ModuleName, uusdCoins)
	require.NoError(t, err)
	err = input.BankKeeper.SendCoinsFromModuleToAccount(input.Ctx, banktypes.ModuleName, address, uusdCoins)
	require.NoError(t, err)
	justBurnItAlready(input.Ctx, input.BankKeeper, address)

	afterBalance := input.BankKeeper.GetBalance(input.Ctx, address, "uusd")
	require.True(t, afterBalance.IsZero())
}

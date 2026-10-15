package app_test

import (
	"testing"

	terraapp "github.com/classic-terra/core/v4/app"
	vestingtypes "github.com/classic-terra/core/v4/x/vesting/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	"github.com/stretchr/testify/require"
)

func TestEncodingConfigLazyGradedVestingAccount(t *testing.T) {
	config := terraapp.MakeEncodingConfig()
	original := vestingtypes.NewLazyGradedVestingAccount(authtypes.NewBaseAccountWithAddress(sdk.AccAddress(make([]byte, 20))), sdk.NewCoins(), nil)
	bytes, err := config.Marshaler.MarshalInterface(original)
	require.NoError(t, err)
	var account sdk.AccountI
	require.NoError(t, config.Marshaler.UnmarshalInterface(bytes, &account))
	require.IsType(t, original, account)
	require.Equal(t, original.GetAddress(), account.GetAddress())
}

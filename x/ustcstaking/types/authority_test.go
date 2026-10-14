package types

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestParamsAuthorityCannotRotate(t *testing.T) {
	p := DefaultParams()
	p.Authority = sdk.AccAddress([]byte("a-wallet-address-123")).String()
	require.ErrorIs(t, p.Validate(), ErrInvalidAuthority)
}

package types

import (
	"encoding/json"
	"testing"
	"time"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"
)

func TestDefaultGenesisDoesNotExposeFundingAuthority(t *testing.T) {
	encoded, err := json.Marshal(DefaultGenesisState())
	require.NoError(t, err)
	require.NotContains(t, string(encoded), "funding_authority")
	require.Contains(t, string(encoded), "authority")
}

func TestDefaultGenesisInitializesStreamingState(t *testing.T) {
	genesis := DefaultGenesisState()
	require.True(t, genesis.RewardState.UnallocatedRewards.IsZero())
	require.True(t, genesis.RewardState.AllocatedRewards.IsZero())
	require.True(t, genesis.RewardState.EmissionRemainder.IsZero())
	require.True(t, genesis.RewardState.IndexRemainder.IsZero())
	require.Nil(t, genesis.RewardState.LastRewardTime)
}

func TestGenesisValidateRejectsNegativeRewardState(t *testing.T) {
	genesis := DefaultGenesisState()
	genesis.RewardState.RewardIndex = math.LegacyOneDec().Neg()

	require.ErrorIs(t, ValidateGenesis(genesis), ErrInvalidRewardState)
}

func FuzzGenesisAccountingValidationDoesNotPanic(f *testing.F) {
	f.Add(int64(10), int64(2), int64(1))
	f.Add(int64(0), int64(0), int64(0))
	f.Add(int64(-1), int64(1), int64(0))
	f.Fuzz(func(t *testing.T, shares, index, debt int64) {
		genesis := validGenesisPositionState()
		genesis.Positions[0].Shares = math.NewInt(shares)
		genesis.RewardState.TotalShares = math.NewInt(shares)
		genesis.RewardState.RewardIndex = math.LegacyNewDecFromInt(math.NewInt(index))
		genesis.Positions[0].RewardDebt = math.LegacyNewDecFromInt(math.NewInt(debt))
		_ = ValidateGenesis(genesis)
	})
}

func TestGenesisValidateRejectsActiveSharesMismatch(t *testing.T) {
	genesis := DefaultGenesisState()
	genesis.RewardState.TotalShares = math.NewInt(10)
	genesis.Positions = []Position{
		{
			Id:               1,
			Owner:            genesis.Params.Authority,
			Principal:        uusdCoin(10),
			LockTierId:       1,
			Shares:           math.NewInt(9),
			ShareMultiplier:  math.LegacyOneDec(),
			RewardDebt:       math.LegacyZeroDec(),
			LockDuration:     durationPtr(time.Hour),
			Status:           PositionStatus_POSITION_STATUS_ACTIVE,
			ClaimableRewards: uusdCoin(0),
		},
	}

	require.ErrorIs(t, ValidateGenesis(genesis), ErrInvalidRewardState)
}

func TestGenesisValidateAcceptsWithdrawnPositionWithZeroPrincipal(t *testing.T) {
	genesis := DefaultGenesisState()
	duration := time.Hour
	lastRewardTime := time.Date(2026, 10, 13, 0, 0, 0, 0, time.UTC)
	genesis.RewardState.LastRewardTime = &lastRewardTime
	genesis.Positions = []Position{
		{
			Id:               1,
			Owner:            genesis.Params.Authority,
			Principal:        uusdCoin(0),
			LockTierId:       1,
			Shares:           math.ZeroInt(),
			ShareMultiplier:  math.LegacyOneDec(),
			RewardDebt:       math.LegacyZeroDec(),
			LockDuration:     &duration,
			Status:           PositionStatus_POSITION_STATUS_WITHDRAWN,
			ClaimableRewards: uusdCoin(0),
		},
	}
	genesis.NextPositionId = 2

	require.NoError(t, ValidateGenesis(genesis))
}

func TestGenesisValidateRejectsInvalidPrincipalForPositionStatus(t *testing.T) {
	maturity := time.Now().UTC()
	tests := []struct {
		name      string
		principal sdk.Coin
		status    PositionStatus
		shares    math.Int
		maturity  *time.Time
	}{
		{name: "withdrawn positive", principal: uusdCoin(1), status: PositionStatus_POSITION_STATUS_WITHDRAWN, shares: math.ZeroInt()},
		{name: "withdrawn negative", principal: sdk.Coin{Denom: BondDenom, Amount: math.NewInt(-1)}, status: PositionStatus_POSITION_STATUS_WITHDRAWN, shares: math.ZeroInt()},
		{name: "withdrawn wrong denom", principal: sdk.NewCoin("uluna", math.ZeroInt()), status: PositionStatus_POSITION_STATUS_WITHDRAWN, shares: math.ZeroInt()},
		{name: "withdrawn nil amount", principal: sdk.Coin{Denom: BondDenom}, status: PositionStatus_POSITION_STATUS_WITHDRAWN, shares: math.ZeroInt()},
		{name: "active zero", principal: uusdCoin(0), status: PositionStatus_POSITION_STATUS_ACTIVE, shares: math.OneInt()},
		{name: "unbonding zero", principal: uusdCoin(0), status: PositionStatus_POSITION_STATUS_UNBONDING, shares: math.ZeroInt(), maturity: &maturity},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			genesis := DefaultGenesisState()
			genesis.Positions = []Position{
				{
					Id:               1,
					Owner:            genesis.Params.Authority,
					Principal:        tt.principal,
					LockTierId:       1,
					Shares:           tt.shares,
					ShareMultiplier:  math.LegacyOneDec(),
					LockDuration:     durationPtr(time.Hour),
					UnbondingEndTime: tt.maturity,
					Status:           tt.status,
					ClaimableRewards: uusdCoin(0),
				},
			}
			genesis.NextPositionId = 2
			if tt.status == PositionStatus_POSITION_STATUS_ACTIVE {
				genesis.RewardState.TotalShares = tt.shares
			}

			require.ErrorIs(t, ValidateGenesis(genesis), ErrInvalidPosition)
		})
	}
}

func TestGenesisValidateRejectsNilOrZeroLockSnapshot(t *testing.T) {
	genesis := validGenesisPositionState()
	genesis.Positions[0].LockDuration = nil
	require.ErrorIs(t, ValidateGenesis(genesis), ErrInvalidLockSnapshot)

	genesis = validGenesisPositionState()
	genesis.Positions[0].ShareMultiplier = math.LegacyZeroDec()
	require.ErrorIs(t, ValidateGenesis(genesis), ErrInvalidLockSnapshot)
}

func TestGenesisValidateRejectsActiveMaturityTimestamp(t *testing.T) {
	genesis := validGenesisPositionState()
	maturity := time.Now().UTC().Add(time.Hour)
	genesis.Positions[0].UnbondingEndTime = &maturity

	require.ErrorIs(t, ValidateGenesis(genesis), ErrInvalidPosition)
}

func TestGenesisValidateRejectsWrongClaimableDenom(t *testing.T) {
	genesis := validGenesisPositionState()
	genesis.Positions[0].ClaimableRewards = sdk.NewCoin("uluna", math.ZeroInt())

	require.ErrorIs(t, ValidateGenesis(genesis), ErrInvalidPosition)
}

func TestGenesisValidateRejectsNilArithmeticState(t *testing.T) {
	genesis := validGenesisPositionState()
	genesis.RewardState.RewardIndex = math.LegacyDec{}
	require.ErrorIs(t, ValidateGenesis(genesis), ErrInvalidRewardState)

	genesis = validGenesisPositionState()
	genesis.Positions[0].RewardDebt = math.LegacyDec{}
	require.ErrorIs(t, ValidateGenesis(genesis), ErrInvalidPosition)

	genesis = validGenesisPositionState()
	genesis.Positions[0].Shares = math.Int{}
	require.ErrorIs(t, ValidateGenesis(genesis), ErrInvalidPosition)
}

func TestGenesisValidateRejectsZeroTierIDAndZeroMaturity(t *testing.T) {
	genesis := validGenesisPositionState()
	genesis.Positions[0].LockTierId = 0
	require.ErrorIs(t, ValidateGenesis(genesis), ErrInvalidLockSnapshot)

	genesis = validGenesisPositionState()
	genesis.Positions[0].Status = PositionStatus_POSITION_STATUS_UNBONDING
	genesis.Positions[0].Shares = math.ZeroInt()
	genesis.Positions[0].UnbondingEndTime = &time.Time{}
	genesis.RewardState.TotalShares = math.ZeroInt()
	require.ErrorIs(t, ValidateGenesis(genesis), ErrInvalidPosition)
}

func TestGenesisValidateRejectsActiveDebtAboveAccruedRewards(t *testing.T) {
	genesis := validGenesisPositionState()
	genesis.Positions[0].RewardDebt = math.LegacyOneDec()
	require.ErrorIs(t, ValidateGenesis(genesis), ErrInvalidPosition)
}

func TestGenesisValidateRequiresZeroDebtForWithdrawnPosition(t *testing.T) {
	genesis := validGenesisPositionState()
	genesis.Positions[0].Status = PositionStatus_POSITION_STATUS_WITHDRAWN
	genesis.Positions[0].Principal = uusdCoin(0)
	genesis.Positions[0].Shares = math.ZeroInt()
	genesis.Positions[0].RewardDebt = math.LegacyOneDec()
	genesis.RewardState.TotalShares = math.ZeroInt()
	require.ErrorIs(t, ValidateGenesis(genesis), ErrInvalidPosition)
}

func TestGenesisValidateRejectsMaxPositionID(t *testing.T) {
	genesis := validGenesisPositionState()
	genesis.Positions[0].Id = ^uint64(0)
	genesis.NextPositionId = ^uint64(0)

	require.ErrorIs(t, ValidateGenesis(genesis), ErrInvalidGenesis)
}

func validGenesisPositionState() *GenesisState {
	duration := time.Hour
	genesis := DefaultGenesisState()
	genesis.Positions = []Position{{
		Id:               1,
		Owner:            genesis.Params.Authority,
		Principal:        uusdCoin(10),
		LockTierId:       1,
		Shares:           math.NewInt(10),
		ShareMultiplier:  math.LegacyOneDec(),
		RewardDebt:       math.LegacyZeroDec(),
		LockDuration:     &duration,
		Status:           PositionStatus_POSITION_STATUS_ACTIVE,
		ClaimableRewards: uusdCoin(0),
	}}
	genesis.RewardState.TotalShares = math.NewInt(10)
	genesis.NextPositionId = 2
	return genesis
}

func uusdCoin(amount int64) sdk.Coin {
	return sdk.NewCoin(BondDenom, math.NewInt(amount))
}

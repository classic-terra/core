package keeper

import (
	"context"
	"errors"
	"testing"
	"time"

	"cosmossdk.io/log"
	"cosmossdk.io/math"
	store "cosmossdk.io/store"
	storemetrics "cosmossdk.io/store/metrics"
	storetypes "cosmossdk.io/store/types"
	"github.com/classic-terra/core/v4/x/ustcstaking/types"
	tmproto "github.com/cometbft/cometbft/proto/tendermint/types"
	dbm "github.com/cosmos/cosmos-db"
	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	"github.com/stretchr/testify/require"
)

type recordingBankKeeper struct {
	accountToModuleCalls      int
	moduleToAccountCalls      int
	lastRecipientModule       string
	lastAccountToModuleAmount sdk.Coins
	rewardPoolBalance         sdk.Coin
	principalPoolBalance      sdk.Coin
	accountToModuleErr        error
	moduleToAccountErr        error
}

type recordingCommunityPoolKeeper struct {
	calls           int
	amount          sdk.Coins
	recipientModule string
	err             error
	bank            *recordingBankKeeper
}

func (d *recordingCommunityPoolKeeper) ReturnRewardDustToCommunityPool(_ context.Context, amount sdk.Coins) error {
	if d.err != nil {
		return d.err
	}
	d.bank.rewardPoolBalance = sdk.NewCoin(types.BondDenom, d.bank.rewardPoolBalance.Amount.Sub(amount.AmountOf(types.BondDenom)))
	return nil
}

func (d *recordingCommunityPoolKeeper) DistributeFromCommunityPoolToModule(_ context.Context, amount sdk.Coins, recipientModule string) error {
	d.calls++
	d.amount = amount
	d.recipientModule = recipientModule
	if d.err != nil {
		return d.err
	}
	if d.bank != nil {
		balance := d.bank.rewardPoolBalance.Amount
		if balance.IsNil() {
			balance = math.ZeroInt()
		}
		d.bank.rewardPoolBalance = sdk.NewCoin(types.BondDenom, balance.Add(amount.AmountOf(types.BondDenom)))
	}
	return nil
}

func (b *recordingBankKeeper) SendCoinsFromAccountToModule(_ context.Context, _ sdk.AccAddress, module string, amount sdk.Coins) error {
	if b.accountToModuleErr != nil {
		return b.accountToModuleErr
	}
	b.accountToModuleCalls++
	b.lastRecipientModule = module
	b.lastAccountToModuleAmount = amount
	if module != types.PrincipalPoolName {
		return errors.New("unexpected destination module")
	}
	b.principalPoolBalance = sdk.NewCoin(types.BondDenom, intOrZero(b.principalPoolBalance.Amount).Add(amount.AmountOf(types.BondDenom)))
	return nil
}

func (b *recordingBankKeeper) SendCoinsFromModuleToAccount(_ context.Context, module string, _ sdk.AccAddress, amount sdk.Coins) error {
	if b.moduleToAccountErr != nil {
		return b.moduleToAccountErr
	}
	var balance *sdk.Coin
	switch module {
	case types.PrincipalPoolName:
		balance = &b.principalPoolBalance
	case types.RewardPoolName:
		balance = &b.rewardPoolBalance
	default:
		return errors.New("unexpected source module")
	}
	debit := amount.AmountOf(types.BondDenom)
	if intOrZero(balance.Amount).LT(debit) {
		return errors.New("insufficient module balance")
	}
	b.moduleToAccountCalls++
	*balance = sdk.NewCoin(types.BondDenom, intOrZero(balance.Amount).Sub(debit))
	return nil
}

func (b *recordingBankKeeper) GetBalance(_ context.Context, addr sdk.AccAddress, denom string) sdk.Coin {
	if addr.Equals(authtypes.NewModuleAddress(types.PrincipalPoolName)) {
		if b.principalPoolBalance.Amount.IsNil() {
			return sdk.NewCoin(denom, math.ZeroInt())
		}
		return b.principalPoolBalance
	}
	return b.rewardPoolBalance
}

func intOrZero(value math.Int) math.Int {
	if value.IsNil() {
		return math.ZeroInt()
	}
	return value
}

func (b *recordingBankKeeper) GetAllBalances(_ context.Context, addr sdk.AccAddress) sdk.Coins {
	if addr.Equals(authtypes.NewModuleAddress(types.PrincipalPoolName)) {
		if b.principalPoolBalance.IsNil() {
			return sdk.NewCoins()
		}
		return sdk.NewCoins(b.principalPoolBalance)
	}
	if b.rewardPoolBalance.IsNil() {
		return sdk.NewCoins()
	}
	return sdk.NewCoins(b.rewardPoolBalance)
}

func newKeeperTest(t testing.TB) (sdk.Context, Keeper) {
	t.Helper()
	key := storetypes.NewKVStoreKey(types.StoreKey)
	db := dbm.NewMemDB()
	ms := store.NewCommitMultiStore(db, log.NewNopLogger(), storemetrics.NewNoOpMetrics())
	ms.MountStoreWithDB(key, storetypes.StoreTypeIAVL, db)
	require.NoError(t, ms.LoadLatestVersion())
	ctx := sdk.NewContext(ms, tmproto.Header{Time: time.Date(2026, 10, 13, 0, 0, 0, 0, time.UTC)}, false, log.NewNopLogger())
	cdc := codec.NewProtoCodec(codectypes.NewInterfaceRegistry())
	bank := &recordingBankKeeper{rewardPoolBalance: sdk.NewCoin(types.BondDenom, math.ZeroInt())}
	keeper := NewKeeper(cdc, key, bank, &recordingCommunityPoolKeeper{bank: bank})
	keeper.SetRewardState(ctx, newTestRewardState(ctx, math.LegacyZeroDec(), math.ZeroInt()))
	return ctx, keeper
}

func newTestRewardState(ctx sdk.Context, index math.LegacyDec, shares math.Int) types.RewardState {
	now := ctx.BlockTime()
	return types.RewardState{
		RewardIndex: index, TotalShares: shares,
		UnallocatedRewards: math.ZeroInt(), AllocatedRewards: math.ZeroInt(),
		LastRewardTime: &now, EmissionRemainder: math.ZeroInt(), IndexRemainder: math.ZeroInt(),
	}
}

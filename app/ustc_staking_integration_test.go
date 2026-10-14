package app_test

import (
	"context"
	"testing"
	"time"

	sdkmath "cosmossdk.io/math"
	helperstest "github.com/classic-terra/core/v4/app/testing"
	ustcstakingkeeper "github.com/classic-terra/core/v4/x/ustcstaking/keeper"
	ustcstakingtypes "github.com/classic-terra/core/v4/x/ustcstaking/types"
	abci "github.com/cometbft/cometbft/abci/types"
	tmproto "github.com/cometbft/cometbft/proto/tendermint/types"
	"github.com/cosmos/cosmos-sdk/client/tx"
	"github.com/cosmos/cosmos-sdk/types"
	txsigning "github.com/cosmos/cosmos-sdk/types/tx/signing"
	authsigning "github.com/cosmos/cosmos-sdk/x/auth/signing"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	banktestutil "github.com/cosmos/cosmos-sdk/x/bank/testutil"
	distrtypes "github.com/cosmos/cosmos-sdk/x/distribution/types"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
	"github.com/stretchr/testify/require"
)

func TestUSTCStakingUsesRealAppKeepers(t *testing.T) {
	var suite helperstest.KeeperTestHelper
	suite.SetT(t)
	suite.Setup(t, "ustc-staking-integration")

	owner := suite.TestAccs[0]
	suite.FundAcc(owner, types.NewCoins(types.NewCoin(ustcstakingtypes.BondDenom, sdkmath.NewInt(140))))

	principalModule := suite.App.AccountKeeper.GetModuleAccount(suite.Ctx, ustcstakingtypes.PrincipalPoolName)
	rewardModule := suite.App.AccountKeeper.GetModuleAccount(suite.Ctx, ustcstakingtypes.RewardPoolName)
	require.NotNil(t, principalModule)
	require.NotNil(t, rewardModule)
	require.True(t, suite.App.BankKeeper.BlockedAddr(rewardModule.GetAddress()))
	_, principalPermissions := suite.App.AccountKeeper.GetModuleAccountAndPermissions(suite.Ctx, ustcstakingtypes.PrincipalPoolName)
	_, rewardPermissions := suite.App.AccountKeeper.GetModuleAccountAndPermissions(suite.Ctx, ustcstakingtypes.RewardPoolName)
	require.Empty(t, principalPermissions)
	require.Empty(t, rewardPermissions)

	params := ustcstakingtypes.DefaultParams()
	params.Authority = authtypes.NewModuleAddress(govtypes.ModuleName).String()
	params.LockTiers = []ustcstakingtypes.LockTier{{
		Id:         1,
		Duration:   durationPtr(time.Hour),
		Multiplier: sdkmath.LegacyOneDec(),
	}}
	suite.App.UstcStakingKeeper.SetParams(suite.Ctx, params)
	require.NoError(t, banktestutil.FundModuleAccount(suite.Ctx, suite.App.BankKeeper, distrtypes.ModuleName,
		types.NewCoins(types.NewCoin(ustcstakingtypes.BondDenom, sdkmath.NewInt(40)))))
	feePool, err := suite.App.DistrKeeper.FeePool.Get(suite.Ctx)
	require.NoError(t, err)
	feePool.CommunityPool = types.NewDecCoinsFromCoins(types.NewCoin(ustcstakingtypes.BondDenom, sdkmath.NewInt(40)))
	require.NoError(t, suite.App.DistrKeeper.FeePool.Set(suite.Ctx, feePool))
	distributionModule := suite.App.AccountKeeper.GetModuleAccount(suite.Ctx, distrtypes.ModuleName)
	require.NotNil(t, distributionModule)
	require.Equal(t, sdkmath.NewInt(40), suite.App.BankKeeper.GetBalance(suite.Ctx, distributionModule.GetAddress(), ustcstakingtypes.BondDenom).Amount)

	initialSupply := suite.App.BankKeeper.GetSupply(suite.Ctx, ustcstakingtypes.BondDenom)
	msgServer := ustcstakingkeeper.NewMsgServerImpl(suite.App.UstcStakingKeeper)
	stakeResponse, err := msgServer.Stake(
		suite.Ctx,
		&ustcstakingtypes.MsgStake{Owner: owner.String(), Amount: types.NewCoin(ustcstakingtypes.BondDenom, sdkmath.NewInt(100)), LockTierId: 1},
	)
	require.NoError(t, err)
	require.Equal(t, uint64(1), stakeResponse.PositionId)

	_, err = msgServer.FundRewards(
		suite.Ctx,
		&ustcstakingtypes.MsgFundRewards{Authority: params.Authority, Amount: types.NewCoin(ustcstakingtypes.BondDenom, sdkmath.NewInt(40))},
	)
	require.NoError(t, err)
	require.Equal(t, sdkmath.NewInt(40), suite.App.BankKeeper.GetBalance(suite.Ctx, rewardModule.GetAddress(), ustcstakingtypes.BondDenom).Amount)
	feePool, err = suite.App.DistrKeeper.FeePool.Get(suite.Ctx)
	require.NoError(t, err)
	require.True(t, feePool.CommunityPool.IsZero())
	require.True(t, suite.App.BankKeeper.GetBalance(suite.Ctx, distributionModule.GetAddress(), ustcstakingtypes.BondDenom).IsZero())

	claimResponse, err := msgServer.ClaimRewards(
		suite.Ctx,
		&ustcstakingtypes.MsgClaimRewards{Owner: owner.String(), PositionId: stakeResponse.PositionId},
	)
	require.NoError(t, err)
	require.Equal(t, sdkmath.NewInt(40), claimResponse.Amount.Amount)

	require.Equal(t, sdkmath.NewInt(100), suite.App.BankKeeper.GetBalance(suite.Ctx, principalModule.GetAddress(), ustcstakingtypes.BondDenom).Amount)
	require.True(t, suite.App.BankKeeper.GetBalance(suite.Ctx, rewardModule.GetAddress(), ustcstakingtypes.BondDenom).IsZero())
	feePool, err = suite.App.DistrKeeper.FeePool.Get(suite.Ctx)
	require.NoError(t, err)
	require.True(t, feePool.CommunityPool.IsZero())
	require.True(t, suite.App.BankKeeper.GetBalance(suite.Ctx, distributionModule.GetAddress(), ustcstakingtypes.BondDenom).IsZero())
	require.Equal(t, initialSupply, suite.App.BankKeeper.GetSupply(suite.Ctx, ustcstakingtypes.BondDenom))
	require.NoError(t, suite.App.UstcStakingKeeper.ValidateState(suite.Ctx))
}

func TestUSTCStakingRejectsFundingAboveCommunityPool(t *testing.T) {
	var suite helperstest.KeeperTestHelper
	suite.SetT(t)
	suite.Setup(t, "ustc-staking-insufficient-community-pool")

	owner := suite.TestAccs[0]
	suite.FundAcc(owner, types.NewCoins(types.NewCoin(ustcstakingtypes.BondDenom, sdkmath.NewInt(20))))
	params := ustcstakingtypes.DefaultParams()
	params.LockTiers = []ustcstakingtypes.LockTier{{
		Id: 1, Duration: durationPtr(time.Hour), Multiplier: sdkmath.LegacyOneDec(),
	}}
	suite.App.UstcStakingKeeper.SetParams(suite.Ctx, params)
	distributionModule := suite.App.AccountKeeper.GetModuleAccount(suite.Ctx, distrtypes.ModuleName)
	require.NotNil(t, distributionModule)
	require.NoError(t, banktestutil.FundModuleAccount(suite.Ctx, suite.App.BankKeeper, distrtypes.ModuleName,
		types.NewCoins(types.NewCoin(ustcstakingtypes.BondDenom, sdkmath.NewInt(10)))))
	feePool, err := suite.App.DistrKeeper.FeePool.Get(suite.Ctx)
	require.NoError(t, err)
	feePool.CommunityPool = types.NewDecCoinsFromCoins(types.NewCoin(ustcstakingtypes.BondDenom, sdkmath.NewInt(5)))
	require.NoError(t, suite.App.DistrKeeper.FeePool.Set(suite.Ctx, feePool))
	_, err = ustcstakingkeeper.NewMsgServerImpl(suite.App.UstcStakingKeeper).Stake(suite.Ctx, &ustcstakingtypes.MsgStake{
		Owner: owner.String(), Amount: types.NewCoin(ustcstakingtypes.BondDenom, sdkmath.NewInt(20)), LockTierId: 1,
	})
	require.NoError(t, err)

	_, err = ustcstakingkeeper.NewMsgServerImpl(suite.App.UstcStakingKeeper).FundRewards(suite.Ctx, &ustcstakingtypes.MsgFundRewards{
		Authority: params.Authority, Amount: types.NewCoin(ustcstakingtypes.BondDenom, sdkmath.NewInt(10)),
	})

	require.ErrorIs(t, err, distrtypes.ErrBadDistribution)
	feePool, err = suite.App.DistrKeeper.FeePool.Get(suite.Ctx)
	require.NoError(t, err)
	require.Equal(t, sdkmath.NewInt(5), feePool.CommunityPool.AmountOf(ustcstakingtypes.BondDenom).TruncateInt())
	require.Equal(t, sdkmath.NewInt(10), suite.App.BankKeeper.GetBalance(suite.Ctx, distributionModule.GetAddress(), ustcstakingtypes.BondDenom).Amount)
	rewardModule := suite.App.AccountKeeper.GetModuleAccount(suite.Ctx, ustcstakingtypes.RewardPoolName)
	require.True(t, suite.App.BankKeeper.GetBalance(suite.Ctx, rewardModule.GetAddress(), ustcstakingtypes.BondDenom).IsZero())
	require.True(t, suite.App.UstcStakingKeeper.GetRewardState(suite.Ctx).RewardIndex.IsZero())
}

func TestUSTCStakingAppFailuresDoNotChangeStateOrSupply(t *testing.T) {
	var suite helperstest.KeeperTestHelper
	suite.SetT(t)
	suite.Setup(t, "ustc-staking-failure-rollback")

	owner, other := suite.TestAccs[0], suite.TestAccs[1]
	suite.FundAcc(owner, types.NewCoins(types.NewCoin(ustcstakingtypes.BondDenom, sdkmath.NewInt(100))))
	suite.FundAcc(other, types.NewCoins(types.NewCoin(ustcstakingtypes.BondDenom, sdkmath.NewInt(100))))
	params := ustcstakingtypes.DefaultParams()
	params.Authority = authtypes.NewModuleAddress(govtypes.ModuleName).String()
	params.LockTiers = []ustcstakingtypes.LockTier{{Id: 1, Duration: durationPtr(time.Hour), Multiplier: sdkmath.LegacyOneDec()}}
	suite.App.UstcStakingKeeper.SetParams(suite.Ctx, params)
	server := ustcstakingkeeper.NewMsgServerImpl(suite.App.UstcStakingKeeper)

	position, err := server.Stake(suite.Ctx, &ustcstakingtypes.MsgStake{
		Owner: owner.String(), Amount: types.NewCoin(ustcstakingtypes.BondDenom, sdkmath.NewInt(50)), LockTierId: 1,
	})
	require.NoError(t, err)

	t.Run("wrong authority", func(t *testing.T) {
		before := snapshotUSTCStakingFailureState(t, &suite, owner, other, position.PositionId)
		_, err := server.FundRewards(suite.Ctx, &ustcstakingtypes.MsgFundRewards{
			Authority: other.String(), Amount: types.NewCoin(ustcstakingtypes.BondDenom, sdkmath.NewInt(1)),
		})
		require.ErrorIs(t, err, ustcstakingtypes.ErrUnauthorized)
		assertUSTCStakingFailureStateUnchanged(t, &suite, owner, other, position.PositionId, before)
	})

	t.Run("wrong denom", func(t *testing.T) {
		before := snapshotUSTCStakingFailureState(t, &suite, owner, other, position.PositionId)
		_, err := server.Stake(suite.Ctx, &ustcstakingtypes.MsgStake{
			Owner: owner.String(), Amount: types.NewCoin("uluna", sdkmath.NewInt(1)), LockTierId: 1,
		})
		require.Error(t, err)
		assertUSTCStakingFailureStateUnchanged(t, &suite, owner, other, position.PositionId, before)
	})

	t.Run("paused", func(t *testing.T) {
		paused := suite.App.UstcStakingKeeper.GetParams(suite.Ctx)
		paused.Paused = true
		suite.App.UstcStakingKeeper.SetParams(suite.Ctx, paused)
		before := snapshotUSTCStakingFailureState(t, &suite, owner, other, position.PositionId)
		_, err := server.Stake(suite.Ctx, &ustcstakingtypes.MsgStake{
			Owner: owner.String(), Amount: types.NewCoin(ustcstakingtypes.BondDenom, sdkmath.NewInt(1)), LockTierId: 1,
		})
		require.ErrorIs(t, err, ustcstakingtypes.ErrModulePaused)
		assertUSTCStakingFailureStateUnchanged(t, &suite, owner, other, position.PositionId, before)
		paused.Paused = false
		suite.App.UstcStakingKeeper.SetParams(suite.Ctx, paused)
	})

	t.Run("wrong owner", func(t *testing.T) {
		before := snapshotUSTCStakingFailureState(t, &suite, owner, other, position.PositionId)
		_, err := server.BeginUnbonding(suite.Ctx, &ustcstakingtypes.MsgBeginUnbonding{
			Owner: other.String(), PositionId: position.PositionId,
		})
		require.ErrorIs(t, err, ustcstakingtypes.ErrPositionOwner)
		assertUSTCStakingFailureStateUnchanged(t, &suite, owner, other, position.PositionId, before)
	})

	t.Run("premature maturity", func(t *testing.T) {
		_, err := server.BeginUnbonding(suite.Ctx, &ustcstakingtypes.MsgBeginUnbonding{Owner: owner.String(), PositionId: position.PositionId})
		require.NoError(t, err)
		before := snapshotUSTCStakingFailureState(t, &suite, owner, other, position.PositionId)
		_, err = server.Withdraw(suite.Ctx, &ustcstakingtypes.MsgWithdraw{Owner: owner.String(), PositionId: position.PositionId})
		require.ErrorIs(t, err, ustcstakingtypes.ErrNotMatured)
		assertUSTCStakingFailureStateUnchanged(t, &suite, owner, other, position.PositionId, before)
	})

	t.Run("principal transfer failure", func(t *testing.T) {
		suite.Ctx = suite.Ctx.WithBlockTime(suite.Ctx.BlockTime().Add(2 * time.Hour))
		// Create a deterministic bank-send failure while preserving a valid, mature
		// unbonding position. The snapshot is taken after this deliberate setup.
		require.NoError(t, suite.App.BankKeeper.SendCoinsFromModuleToAccount(suite.Ctx,
			ustcstakingtypes.PrincipalPoolName, other, types.NewCoins(types.NewCoin(ustcstakingtypes.BondDenom, sdkmath.NewInt(50)))))
		before := snapshotUSTCStakingFailureState(t, &suite, owner, other, position.PositionId)
		_, err := server.Withdraw(suite.Ctx, &ustcstakingtypes.MsgWithdraw{Owner: owner.String(), PositionId: position.PositionId})
		require.Error(t, err)
		assertUSTCStakingFailureStateUnchanged(t, &suite, owner, other, position.PositionId, before)
	})
}

func TestUSTCStakingFailedMultiMessageTxDiscardsEarlierStake(t *testing.T) {
	var suite helperstest.KeeperTestHelper
	suite.SetT(t)
	suite.Setup(t, "ustc-staking-multimsg-cache-rollback")

	privateKey, publicKey, owner := suite.Ed25519PubAddr()
	suite.FundAcc(owner, types.NewCoins(types.NewCoin(ustcstakingtypes.BondDenom, sdkmath.NewInt(50))))
	params := ustcstakingtypes.DefaultParams()
	params.LockTiers = []ustcstakingtypes.LockTier{{Id: 1, Duration: durationPtr(time.Hour), Multiplier: sdkmath.LegacyOneDec()}}
	suite.App.UstcStakingKeeper.SetParams(suite.Ctx, params)

	account := suite.App.AccountKeeper.GetAccount(suite.Ctx, owner)
	ownerBefore := suite.App.BankKeeper.GetAllBalances(suite.Ctx, owner)
	supplyBefore := suite.App.BankKeeper.GetSupply(suite.Ctx, ustcstakingtypes.BondDenom)
	principalModule := authtypes.NewModuleAddress(ustcstakingtypes.PrincipalPoolName)
	principalBefore := suite.App.BankKeeper.GetAllBalances(suite.Ctx, principalModule)
	nextIDBefore := suite.App.UstcStakingKeeper.GetNextPositionID(suite.Ctx)
	rewardStateBefore := suite.App.UstcStakingKeeper.GetRewardState(suite.Ctx)

	builder := suite.App.GetTxConfig().NewTxBuilder()
	require.NoError(t, builder.SetMsgs(
		&ustcstakingtypes.MsgStake{Owner: owner.String(), Amount: types.NewCoin(ustcstakingtypes.BondDenom, sdkmath.NewInt(10)), LockTierId: 1},
		&ustcstakingtypes.MsgStake{Owner: owner.String(), Amount: types.NewCoin("uluna", sdkmath.NewInt(10)), LockTierId: 1},
	))
	builder.SetGasLimit(1_000_000)
	builder.SetFeeAmount(types.NewCoins())
	require.NoError(t, builder.SetSignatures(txsigning.SignatureV2{
		PubKey:   publicKey,
		Data:     &txsigning.SingleSignatureData{SignMode: txsigning.SignMode_SIGN_MODE_DIRECT},
		Sequence: account.GetSequence(),
	}))
	signature, err := tx.SignWithPrivKey(context.Background(), txsigning.SignMode_SIGN_MODE_DIRECT,
		authsigning.SignerData{ChainID: suite.Ctx.ChainID(), AccountNumber: account.GetAccountNumber(), Sequence: account.GetSequence()},
		builder, privateKey, suite.App.GetTxConfig(), 0)
	require.NoError(t, err)
	require.NoError(t, builder.SetSignatures(signature))
	txBytes, err := suite.App.GetTxConfig().TxEncoder()(builder.GetTx())
	require.NoError(t, err)
	now := suite.Ctx.BlockTime().Add(time.Second)
	height := suite.App.LastBlockHeight() + 1
	finalized, err := suite.App.FinalizeBlock(&abci.RequestFinalizeBlock{Height: height, Time: now, Txs: [][]byte{txBytes}})
	require.NoError(t, err)
	require.Len(t, finalized.TxResults, 1)
	require.NotZero(t, finalized.TxResults[0].Code, finalized.TxResults[0].Log)
	_, err = suite.App.Commit()
	require.NoError(t, err)
	suite.Ctx = suite.App.NewUncachedContext(false, tmproto.Header{Height: height, ChainID: "ustc-staking-multimsg-cache-rollback", Time: now})

	require.Equal(t, ownerBefore, suite.App.BankKeeper.GetAllBalances(suite.Ctx, owner))
	require.Equal(t, principalBefore, suite.App.BankKeeper.GetAllBalances(suite.Ctx, principalModule))
	require.Equal(t, supplyBefore, suite.App.BankKeeper.GetSupply(suite.Ctx, ustcstakingtypes.BondDenom))
	require.Equal(t, nextIDBefore, suite.App.UstcStakingKeeper.GetNextPositionID(suite.Ctx))
	require.Equal(t, rewardStateBefore, suite.App.UstcStakingKeeper.GetRewardState(suite.Ctx))
	_, found := suite.App.UstcStakingKeeper.GetPosition(suite.Ctx, nextIDBefore)
	require.False(t, found)
}

type ustcStakingFailureSnapshot struct {
	OwnerBalances    types.Coins
	OtherBalances    types.Coins
	PrincipalBalance types.Coin
	RewardBalance    types.Coin
	Distribution     types.Coin
	FeePool          distrtypes.FeePool
	Supply           types.Coin
	NextPositionID   uint64
	RewardState      ustcstakingtypes.RewardState
	Params           ustcstakingtypes.Params
	Position         ustcstakingtypes.Position
}

func snapshotUSTCStakingFailureState(t *testing.T, suite *helperstest.KeeperTestHelper, owner, other types.AccAddress, positionID uint64) ustcStakingFailureSnapshot {
	t.Helper()
	principalAddress := authtypes.NewModuleAddress(ustcstakingtypes.PrincipalPoolName)
	rewardAddress := authtypes.NewModuleAddress(ustcstakingtypes.RewardPoolName)
	distributionAddress := authtypes.NewModuleAddress(distrtypes.ModuleName)
	feePool, err := suite.App.DistrKeeper.FeePool.Get(suite.Ctx)
	require.NoError(t, err)
	position, found := suite.App.UstcStakingKeeper.GetPosition(suite.Ctx, positionID)
	require.True(t, found)
	return ustcStakingFailureSnapshot{
		OwnerBalances:    suite.App.BankKeeper.GetAllBalances(suite.Ctx, owner),
		OtherBalances:    suite.App.BankKeeper.GetAllBalances(suite.Ctx, other),
		PrincipalBalance: suite.App.BankKeeper.GetBalance(suite.Ctx, principalAddress, ustcstakingtypes.BondDenom),
		RewardBalance:    suite.App.BankKeeper.GetBalance(suite.Ctx, rewardAddress, ustcstakingtypes.BondDenom),
		Distribution:     suite.App.BankKeeper.GetBalance(suite.Ctx, distributionAddress, ustcstakingtypes.BondDenom),
		FeePool:          feePool,
		Supply:           suite.App.BankKeeper.GetSupply(suite.Ctx, ustcstakingtypes.BondDenom),
		NextPositionID:   suite.App.UstcStakingKeeper.GetNextPositionID(suite.Ctx),
		RewardState:      suite.App.UstcStakingKeeper.GetRewardState(suite.Ctx),
		Params:           suite.App.UstcStakingKeeper.GetParams(suite.Ctx),
		Position:         position,
	}
}

func assertUSTCStakingFailureStateUnchanged(t *testing.T, suite *helperstest.KeeperTestHelper, owner, other types.AccAddress, positionID uint64, before ustcStakingFailureSnapshot) {
	t.Helper()
	require.Equal(t, before, snapshotUSTCStakingFailureState(t, suite, owner, other, positionID))
}

func TestUSTCStakingEightUserCompleteLifecyclePreservesSupplyAndValidators(t *testing.T) {
	var suite helperstest.KeeperTestHelper
	suite.SetT(t)
	suite.Setup(t, "ustc-staking-eight-user-lifecycle")

	owners := suite.RandomAccountAddresses(8)
	for _, owner := range owners {
		suite.FundAcc(owner, types.NewCoins(types.NewCoin(ustcstakingtypes.BondDenom, sdkmath.NewInt(10))))
	}

	params := ustcstakingtypes.DefaultParams()
	params.Authority = authtypes.NewModuleAddress(govtypes.ModuleName).String()
	params.LockTiers = []ustcstakingtypes.LockTier{{Id: 1, Duration: durationPtr(time.Hour), Multiplier: sdkmath.LegacyOneDec()}}
	suite.App.UstcStakingKeeper.SetParams(suite.Ctx, params)
	require.NoError(t, banktestutil.FundModuleAccount(suite.Ctx, suite.App.BankKeeper, distrtypes.ModuleName,
		types.NewCoins(types.NewCoin(ustcstakingtypes.BondDenom, sdkmath.NewInt(80)))))
	feePool, err := suite.App.DistrKeeper.FeePool.Get(suite.Ctx)
	require.NoError(t, err)
	feePool.CommunityPool = types.NewDecCoinsFromCoins(types.NewCoin(ustcstakingtypes.BondDenom, sdkmath.NewInt(80)))
	require.NoError(t, suite.App.DistrKeeper.FeePool.Set(suite.Ctx, feePool))

	initialSupply := suite.App.BankKeeper.GetSupply(suite.Ctx, ustcstakingtypes.BondDenom)
	initialValidators, err := suite.App.StakingKeeper.GetAllValidators(suite.Ctx)
	require.NoError(t, err)
	msgServer := ustcstakingkeeper.NewMsgServerImpl(suite.App.UstcStakingKeeper)
	positionIDs := make([]uint64, 0, len(owners))
	for _, owner := range owners {
		response, stakeErr := msgServer.Stake(suite.Ctx, &ustcstakingtypes.MsgStake{
			Owner: owner.String(), Amount: types.NewCoin(ustcstakingtypes.BondDenom, sdkmath.NewInt(10)), LockTierId: 1,
		})
		require.NoError(t, stakeErr)
		positionIDs = append(positionIDs, response.PositionId)
	}
	_, err = msgServer.FundRewards(suite.Ctx, &ustcstakingtypes.MsgFundRewards{
		Authority: params.Authority, Amount: types.NewCoin(ustcstakingtypes.BondDenom, sdkmath.NewInt(80)),
	})
	require.NoError(t, err)

	for i, owner := range owners {
		claim, claimErr := msgServer.ClaimRewards(suite.Ctx, &ustcstakingtypes.MsgClaimRewards{Owner: owner.String(), PositionId: positionIDs[i]})
		require.NoError(t, claimErr)
		require.Equal(t, sdkmath.NewInt(10), claim.Amount.Amount)
		_, unbondErr := msgServer.BeginUnbonding(suite.Ctx, &ustcstakingtypes.MsgBeginUnbonding{Owner: owner.String(), PositionId: positionIDs[i]})
		require.NoError(t, unbondErr)
	}

	suite.Ctx = suite.Ctx.WithBlockTime(suite.Ctx.BlockTime().Add(2 * time.Hour))
	for i, owner := range owners {
		_, withdrawErr := msgServer.Withdraw(suite.Ctx, &ustcstakingtypes.MsgWithdraw{Owner: owner.String(), PositionId: positionIDs[i]})
		require.NoError(t, withdrawErr)
	}

	require.Equal(t, initialSupply, suite.App.BankKeeper.GetSupply(suite.Ctx, ustcstakingtypes.BondDenom))
	finalValidators, err := suite.App.StakingKeeper.GetAllValidators(suite.Ctx)
	require.NoError(t, err)
	require.Equal(t, initialValidators, finalValidators)
	require.NoError(t, suite.App.UstcStakingKeeper.ValidateState(suite.Ctx))
	validation, err := ustcstakingkeeper.NewQueryServerImpl(suite.App.UstcStakingKeeper).ValidateState(
		types.WrapSDKContext(suite.Ctx), &ustcstakingtypes.QueryValidateStateRequest{},
	)
	require.NoError(t, err)
	require.True(t, validation.Valid, validation.Detail)
}

func durationPtr(value time.Duration) *time.Duration {
	return &value
}

func TestUSTCStakingDustReturnsSurplusPreservingRealClaims(t *testing.T) {
	var suite helperstest.KeeperTestHelper
	suite.SetT(t)
	suite.Setup(t, "ustc-dust-integration")
	k := suite.App.UstcStakingKeeper
	params := ustcstakingtypes.DefaultParams()
	params.LockTiers = []ustcstakingtypes.LockTier{{Id: 1, Duration: durationPtr(time.Second), Multiplier: sdkmath.LegacyOneDec()}}
	k.SetParams(suite.Ctx, params)
	owners := suite.TestAccs[:2]
	for _, owner := range owners {
		suite.FundAcc(owner, types.NewCoins(types.NewInt64Coin(ustcstakingtypes.BondDenom, 3)))
	}
	require.NoError(t, banktestutil.FundModuleAccount(suite.Ctx, suite.App.BankKeeper, distrtypes.ModuleName, types.NewCoins(types.NewInt64Coin(ustcstakingtypes.BondDenom, 5))))
	fee, err := suite.App.DistrKeeper.FeePool.Get(suite.Ctx)
	require.NoError(t, err)
	fee.CommunityPool = types.NewDecCoinsFromCoins(types.NewInt64Coin(ustcstakingtypes.BondDenom, 5))
	require.NoError(t, suite.App.DistrKeeper.FeePool.Set(suite.Ctx, fee))
	supply := suite.App.BankKeeper.GetSupply(suite.Ctx, ustcstakingtypes.BondDenom)
	server := ustcstakingkeeper.NewMsgServerImpl(k)
	ids := make([]uint64, len(owners))
	for i, owner := range owners {
		response, err := server.Stake(suite.Ctx, &ustcstakingtypes.MsgStake{Owner: owner.String(), Amount: types.NewInt64Coin(ustcstakingtypes.BondDenom, 3), LockTierId: 1})
		require.NoError(t, err)
		ids[i] = response.PositionId
	}
	_, err = server.FundRewards(suite.Ctx, &ustcstakingtypes.MsgFundRewards{Authority: k.GetAuthority(), Amount: types.NewInt64Coin(ustcstakingtypes.BondDenom, 5)})
	require.NoError(t, err)
	for i, owner := range owners {
		_, err = server.BeginUnbonding(suite.Ctx, &ustcstakingtypes.MsgBeginUnbonding{Owner: owner.String(), PositionId: ids[i]})
		require.NoError(t, err)
	}
	suite.Ctx = suite.Ctx.WithBlockTime(suite.Ctx.BlockTime().Add(time.Second))
	for i, owner := range owners {
		_, err = server.Withdraw(suite.Ctx, &ustcstakingtypes.MsgWithdraw{Owner: owner.String(), PositionId: ids[i]})
		require.NoError(t, err)
	}
	require.Equal(t, sdkmath.NewInt(4), k.GetReservedRewards(suite.Ctx))
	state := k.GetRewardState(suite.Ctx)
	params.Paused = true
	k.SetParams(suite.Ctx, params)
	_, err = server.ReturnRewardDust(suite.Ctx, &ustcstakingtypes.MsgReturnRewardDust{Authority: k.GetAuthority(), Amount: types.NewInt64Coin(ustcstakingtypes.BondDenom, 2)})
	require.Error(t, err)
	_, err = server.ReturnRewardDust(suite.Ctx, &ustcstakingtypes.MsgReturnRewardDust{Authority: k.GetAuthority(), Amount: types.NewInt64Coin(ustcstakingtypes.BondDenom, 1)})
	require.NoError(t, err)
	fee, err = suite.App.DistrKeeper.FeePool.Get(suite.Ctx)
	require.NoError(t, err)
	require.Equal(t, sdkmath.LegacyOneDec(), fee.CommunityPool.AmountOf(ustcstakingtypes.BondDenom))
	require.Equal(t, state, k.GetRewardState(suite.Ctx))
	require.True(t, suite.App.BankKeeper.GetBalance(suite.Ctx, authtypes.NewModuleAddress(ustcstakingtypes.PrincipalPoolName), ustcstakingtypes.BondDenom).IsZero())
	require.NoError(t, k.ValidateState(suite.Ctx))
	for i, owner := range owners {
		response, err := server.ClaimRewards(suite.Ctx, &ustcstakingtypes.MsgClaimRewards{Owner: owner.String(), PositionId: ids[i]})
		require.NoError(t, err)
		require.Equal(t, sdkmath.NewInt(2), response.Amount.Amount)
		_, found := k.GetPosition(suite.Ctx, ids[i])
		require.False(t, found)
	}
	require.True(t, k.GetReservedRewards(suite.Ctx).IsZero())
	require.Equal(t, supply, suite.App.BankKeeper.GetSupply(suite.Ctx, ustcstakingtypes.BondDenom))
	require.NoError(t, k.ValidateState(suite.Ctx))
}

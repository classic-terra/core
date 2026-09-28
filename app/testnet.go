package app

import (
	"fmt"
	"time"

	"cosmossdk.io/math"
	storetypes "cosmossdk.io/store/types"
	upgradetypes "cosmossdk.io/x/upgrade/types"
	cmtcrypto "github.com/cometbft/cometbft/crypto"
	cmtbytes "github.com/cometbft/cometbft/libs/bytes"
	cmtproto "github.com/cometbft/cometbft/proto/tendermint/types"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	cryptocodec "github.com/cosmos/cosmos-sdk/crypto/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/bech32"
	distrtypes "github.com/cosmos/cosmos-sdk/x/distribution/types"
	minttypes "github.com/cosmos/cosmos-sdk/x/mint/types"
	slashingtypes "github.com/cosmos/cosmos-sdk/x/slashing/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
)

// TestnetOptions configures how InitTerraAppForTestnet rewrites the local state.
type TestnetOptions struct {
	NewChainID            string
	NewValAddr            cmtbytes.HexBytes
	NewValPubKey          cmtcrypto.PubKey
	NewOperatorAddress    string
	UpgradeToTrigger      string
	VotingPeriod          time.Duration
	ExpeditedVotingPeriod time.Duration
	FundAccounts          []sdk.AccAddress
	FundCoins             sdk.Coins
}

// InitTerraAppForTestnet rewrites the state of a node's data directory so that a
// single local validator controls the network. It is only used by the
// `in-place-testnet` command and never runs as part of normal block processing.
func InitTerraAppForTestnet(app *TerraApp, opts TestnetOptions) (*TerraApp, error) {
	ctx := app.BaseApp.NewUncachedContext(false, cmtproto.Header{
		ChainID: opts.NewChainID,
		Height:  app.LastBlockHeight(),
		Time:    time.Now(),
	})

	pubKey, err := cryptocodec.FromCmtPubKeyInterface(opts.NewValPubKey)
	if err != nil {
		return nil, fmt.Errorf("convert validator pubkey: %w", err)
	}
	pubKeyAny, err := codectypes.NewAnyWithValue(pubKey)
	if err != nil {
		return nil, err
	}

	_, operatorBz, err := bech32.DecodeAndConvert(opts.NewOperatorAddress)
	if err != nil {
		return nil, fmt.Errorf("decode operator address: %w", err)
	}
	valAddr := sdk.ValAddress(operatorBz)
	operatorAcc := sdk.AccAddress(operatorBz)

	//
	// STAKING
	//

	bondDenom, err := app.StakingKeeper.BondDenom(ctx)
	if err != nil {
		return nil, err
	}

	// The new validator takes over all bonded tokens, so the bonded pool stays
	// consistent and the validator holds 100% of the governance voting power.
	bondedTokens := app.BankKeeper.GetBalance(ctx, app.StakingKeeper.GetBondedPool(ctx).GetAddress(), bondDenom).Amount
	if !bondedTokens.IsPositive() {
		return nil, fmt.Errorf("bonded pool holds no %s", bondDenom)
	}

	newVal := stakingtypes.Validator{
		OperatorAddress: valAddr.String(),
		ConsensusPubkey: pubKeyAny,
		Jailed:          false,
		Status:          stakingtypes.Bonded,
		Tokens:          bondedTokens,
		DelegatorShares: math.LegacyNewDecFromInt(bondedTokens),
		Description:     stakingtypes.NewDescription("rehearsal", "", "", "", ""),
		Commission: stakingtypes.NewCommission(
			math.LegacyNewDecWithPrec(5, 2),
			math.LegacyNewDecWithPrec(20, 2),
			math.LegacyNewDecWithPrec(1, 2),
		),
		MinSelfDelegation: math.OneInt(),
	}

	// Remove all existing validators from the power index, last validator
	// powers, validator records and the unbonding validator queue, together
	// with the delegations to them. The self delegation created below backs all
	// bonded tokens, so surviving delegations would count those tokens twice.
	stakingStore := ctx.KVStore(app.GetKey(stakingtypes.StoreKey))
	for _, prefix := range [][]byte{
		stakingtypes.ValidatorsByPowerIndexKey,
		stakingtypes.LastValidatorPowerKey,
		stakingtypes.ValidatorsKey,
		stakingtypes.ValidatorQueueKey,
		stakingtypes.DelegationKey,
		stakingtypes.DelegationByValIndexKey,
	} {
		deletePrefix(stakingStore, prefix)
	}

	// The removed validators' distribution records go with them. Their
	// outstanding rewards are coins held by the distribution module, so they
	// move to the community pool to keep the module balance accounted for.
	outstanding := sdk.DecCoins{}
	app.DistrKeeper.IterateValidatorOutstandingRewards(ctx, func(_ sdk.ValAddress, rewards distrtypes.ValidatorOutstandingRewards) (stop bool) {
		outstanding = outstanding.Add(rewards.Rewards...)
		return false
	})
	feePool, err := app.DistrKeeper.FeePool.Get(ctx)
	if err != nil {
		return nil, err
	}
	feePool.CommunityPool = feePool.CommunityPool.Add(outstanding...)
	if err := app.DistrKeeper.FeePool.Set(ctx, feePool); err != nil {
		return nil, err
	}

	distrStore := ctx.KVStore(app.GetKey(distrtypes.StoreKey))
	for _, prefix := range [][]byte{
		distrtypes.ValidatorOutstandingRewardsPrefix,
		distrtypes.DelegatorStartingInfoPrefix,
		distrtypes.ValidatorHistoricalRewardsPrefix,
		distrtypes.ValidatorCurrentRewardsPrefix,
		distrtypes.ValidatorAccumulatedCommissionPrefix,
		distrtypes.ValidatorSlashEventPrefix,
	} {
		deletePrefix(distrStore, prefix)
	}

	if err := app.StakingKeeper.SetValidator(ctx, newVal); err != nil {
		return nil, err
	}
	if err := app.StakingKeeper.SetValidatorByConsAddr(ctx, newVal); err != nil {
		return nil, err
	}
	if err := app.StakingKeeper.SetValidatorByPowerIndex(ctx, newVal); err != nil {
		return nil, err
	}
	// A last power of zero makes the next EndBlock send the real voting power to
	// CometBFT, replacing the placeholder power set by testnetify.
	power := newVal.ConsensusPower(app.StakingKeeper.PowerReduction(ctx))
	if err := app.StakingKeeper.SetLastValidatorPower(ctx, valAddr, 0); err != nil {
		return nil, err
	}
	// Initializes distribution records and the slashing pubkey mapping.
	if err := app.StakingKeeper.Hooks().AfterValidatorCreated(ctx, valAddr); err != nil {
		return nil, err
	}

	// Self delegation backing all of the validator's shares.
	if err := app.StakingKeeper.Hooks().BeforeDelegationCreated(ctx, operatorAcc, valAddr); err != nil {
		return nil, err
	}
	if err := app.StakingKeeper.SetDelegation(ctx, stakingtypes.NewDelegation(
		operatorAcc.String(), valAddr.String(), newVal.DelegatorShares,
	)); err != nil {
		return nil, err
	}
	if err := app.StakingKeeper.Hooks().AfterDelegationModified(ctx, operatorAcc, valAddr); err != nil {
		return nil, err
	}

	//
	// SLASHING
	//

	newConsAddr := sdk.ConsAddress(opts.NewValAddr.Bytes())
	if err := app.SlashingKeeper.SetValidatorSigningInfo(ctx, newConsAddr, slashingtypes.NewValidatorSigningInfo(
		newConsAddr, app.LastBlockHeight(), 0, time.Unix(0, 0).UTC(), false, 0,
	)); err != nil {
		return nil, err
	}

	//
	// DISTRIBUTION
	//

	// AfterValidatorCreated already initialized rewards; make sure the
	// outstanding rewards record exists even if hooks change.
	if _, err := app.DistrKeeper.GetValidatorOutstandingRewards(ctx, valAddr); err != nil {
		if err := app.DistrKeeper.SetValidatorOutstandingRewards(ctx, valAddr, distrtypes.ValidatorOutstandingRewards{Rewards: sdk.DecCoins{}}); err != nil {
			return nil, err
		}
	}

	//
	// ORACLE
	//

	// A single validator without a price feeder would otherwise be slashed and
	// jailed at the end of the oracle slash window, halting the network.
	oracleParams := app.OracleKeeper.GetParams(ctx)
	oracleParams.MinValidPerWindow = math.LegacyZeroDec()
	app.OracleKeeper.SetParams(ctx, oracleParams)

	//
	// GOV
	//

	govParams, err := app.GovKeeper.Params.Get(ctx)
	if err != nil {
		return nil, err
	}
	if opts.VotingPeriod > 0 {
		govParams.VotingPeriod = &opts.VotingPeriod
	}
	if opts.ExpeditedVotingPeriod > 0 {
		govParams.ExpeditedVotingPeriod = &opts.ExpeditedVotingPeriod
	}
	if govParams.ExpeditedVotingPeriod != nil && govParams.VotingPeriod != nil &&
		*govParams.ExpeditedVotingPeriod >= *govParams.VotingPeriod {
		half := *govParams.VotingPeriod / 2
		govParams.ExpeditedVotingPeriod = &half
	}
	if err := app.GovKeeper.Params.Set(ctx, govParams); err != nil {
		return nil, err
	}

	//
	// BANK
	//

	if !opts.FundCoins.IsZero() {
		accounts := append([]sdk.AccAddress{operatorAcc}, opts.FundAccounts...)
		for _, acc := range accounts {
			if err := app.BankKeeper.MintCoins(ctx, minttypes.ModuleName, opts.FundCoins); err != nil {
				return nil, err
			}
			if err := app.BankKeeper.SendCoinsFromModuleToAccount(ctx, minttypes.ModuleName, acc, opts.FundCoins); err != nil {
				return nil, err
			}
		}
	}

	//
	// UPGRADE
	//

	if opts.UpgradeToTrigger != "" {
		if err := app.UpgradeKeeper.ScheduleUpgrade(ctx, upgradetypes.Plan{
			Name:   opts.UpgradeToTrigger,
			Height: app.LastBlockHeight() + 10,
		}); err != nil {
			return nil, err
		}
	}

	app.Logger().Info("in-place testnet state applied",
		"chain_id", opts.NewChainID,
		"operator", valAddr.String(),
		"account", operatorAcc.String(),
		"power", power,
		"bond_denom", bondDenom,
	)

	return app, nil
}

func deletePrefix(store storetypes.KVStore, prefix []byte) {
	iterator := storetypes.KVStorePrefixIterator(store, prefix)
	var keys [][]byte
	for ; iterator.Valid(); iterator.Next() {
		keys = append(keys, iterator.Key())
	}
	iterator.Close()
	for _, key := range keys {
		store.Delete(key)
	}
}

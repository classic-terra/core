package keeper

import (
	"cosmossdk.io/math"
	v2customtypes "github.com/classic-terra/core/v3/custom/gov/types"
	core "github.com/classic-terra/core/v3/types"
	markettypes "github.com/classic-terra/core/v3/x/market/types"
	"github.com/cosmos/cosmos-sdk/baseapp"
	"github.com/cosmos/cosmos-sdk/codec"
	storetypes "github.com/cosmos/cosmos-sdk/store/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/gov/keeper"
	"github.com/cosmos/cosmos-sdk/x/gov/types"
)

// Keeper defines the governance module Keeper
type Keeper struct {
	*keeper.Keeper
	authKeeper   types.AccountKeeper
	bankKeeper   types.BankKeeper
	oracleKeeper markettypes.OracleKeeper

	// The reference to the DelegationSet and ValidatorSet to get information about validators and delegators
	sk types.StakingKeeper

	// The (unexposed) keys used to access the stores from the Context.
	storeKey storetypes.StoreKey

	// The codec for binary encoding/decoding.
	cdc codec.BinaryCodec

	// Msg server router
	router *baseapp.MsgServiceRouter

	config types.Config

	// the address capable of executing a MsgUpdateParams message. Typically, this
	// should be the x/gov module account.
	authority string
}

// NewKeeper returns a governance keeper. It handles:
// - submitting governance proposals
// - depositing funds into proposals, and activating upon sufficient funds being deposited
// - users voting on proposals, with weight proportional to stake in the system
// - and tallying the result of the vote.
//
// CONTRACT: the parameter Subspace must have the param key table already initialized
func NewKeeper(
	cdc codec.BinaryCodec, key storetypes.StoreKey, authKeeper types.AccountKeeper,
	bankKeeper types.BankKeeper, sk types.StakingKeeper,
	oracleKeeper markettypes.OracleKeeper,
	router *baseapp.MsgServiceRouter, config types.Config, authority string,
) *Keeper {
	return &Keeper{
		Keeper:       keeper.NewKeeper(cdc, key, authKeeper, bankKeeper, sk, router, config, authority),
		storeKey:     key,
		authKeeper:   authKeeper,
		bankKeeper:   bankKeeper,
		oracleKeeper: oracleKeeper,
		sk:           sk,
		cdc:          cdc,
		router:       router,
		config:       config,
		authority:    authority,
	}
}

// assertMetadataLength returns an error if given metadata length
// is greater than a pre-defined MaxMetadataLen.
func (keeper Keeper) assertMetadataLength(metadata string) error {
	if metadata != "" && uint64(len(metadata)) > keeper.config.MaxMetadataLen {
		return types.ErrMetadataTooLong.Wrapf("got metadata with length %d", len(metadata))
	}
	return nil
}

func (keeper *Keeper) SetHooks(gh types.GovHooks) *Keeper {
	keeper.Keeper = keeper.Keeper.SetHooks(gh)
	return keeper
}

// SetDepositLimitBaseUstc sets a limit deposit(Lunc) base on Ustc to store.
func (keeper Keeper) SetDepositLimitBaseUstc(ctx sdk.Context, proposalID uint64, amount math.Int) error {
	store := ctx.KVStore(keeper.storeKey)
	key := v2customtypes.TotalDepositKey(proposalID)
	bz, err := amount.Marshal()
	if err == nil {
		store.Set(key, bz)
	}
	return err
}

// GetDepositLimitBaseUstc: calculate the minimum LUNC amount to deposit base on Ustc for the proposal
func (keeper Keeper) GetMinimumDepositBaseUstc(ctx sdk.Context) (math.Int, error) {
	// Get exchange rate betweent Lunc/uusd from oracle
	// save it to store
	price, err := keeper.oracleKeeper.GetLunaExchangeRate(ctx, core.MicroUSDDenom)
	// If can't get price from oracle, use min deposit
	if err != nil {
		return keeper.GetParams(ctx).MinDeposit[0].Amount, nil
	}

	MinUusdDeposit := keeper.GetParams(ctx).MinUusdDeposit
	totalUlunaDeposit := sdk.NewDecFromInt(MinUusdDeposit.Amount).Quo(price).TruncateInt()

	return totalUlunaDeposit, nil
}

// GetDepositLimitBaseUstc gets the deposit limit (Lunc) for a specific proposal
func (keeper Keeper) GetDepositLimitBaseUstc(ctx sdk.Context, proposalID uint64) (depositLimit math.Int) {
	store := ctx.KVStore(keeper.storeKey)
	key := v2customtypes.TotalDepositKey(proposalID)
	bz := store.Get(key)
	if bz == nil {
		return sdk.ZeroInt()
	}
	err := depositLimit.Unmarshal(bz)
	if err != nil {
		return sdk.ZeroInt()
	}

	return depositLimit
}

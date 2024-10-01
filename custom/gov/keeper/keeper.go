package keeper

import (
	"fmt"

	markettypes "github.com/classic-terra/core/v3/x/market/types"
	"github.com/cosmos/cosmos-sdk/baseapp"
	"github.com/cosmos/cosmos-sdk/codec"
	storetypes "github.com/cosmos/cosmos-sdk/store/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/gov/keeper"
	"github.com/cosmos/cosmos-sdk/x/gov/types"
	"github.com/cosmos/cosmos-sdk/x/gov/types/v1beta1"
)

// Keeper defines the governance module Keeper
type Keeper struct {
	baseKeeper   keeper.Keeper
	authKeeper   types.AccountKeeper
	bankKeeper   types.BankKeeper
	oracleKeeper markettypes.OracleKeeper

	// The reference to the DelegationSet and ValidatorSet to get information about validators and delegators
	sk types.StakingKeeper

	// GovHooks
	hooks types.GovHooks

	// The (unexposed) keys used to access the stores from the Context.
	storeKey storetypes.StoreKey

	// The codec for binary encoding/decoding.
	cdc codec.BinaryCodec

	// Legacy Proposal router
	legacyRouter v1beta1.Router

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
	// ensure governance module account is set
	if addr := authKeeper.GetModuleAddress(types.ModuleName); addr == nil {
		panic(fmt.Sprintf("%s module account has not been set", types.ModuleName))
	}

	if _, err := sdk.AccAddressFromBech32(authority); err != nil {
		panic(fmt.Sprintf("invalid authority address: %s", authority))
	}

	// If MaxMetadataLen not set by app developer, set to default value.
	if config.MaxMetadataLen == 0 {
		config.MaxMetadataLen = types.DefaultConfig().MaxMetadataLen
	}

	return &Keeper{
		storeKey:     key,
		baseKeeper:   *keeper.NewKeeper(cdc, key, authKeeper, bankKeeper, sk, router, config, authority),
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

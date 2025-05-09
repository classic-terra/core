package staking

import (
	"context"
	"time"

	legacyupgrade "github.com/classic-terra/core/v3/custom/upgrade/legacy"
	sdk "github.com/cosmos/cosmos-sdk/types"
	paramtypes "github.com/cosmos/cosmos-sdk/x/params/types"
	"github.com/cosmos/cosmos-sdk/x/staking/keeper"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
)

type LegacyParams struct {
	// unbonding_time is the time duration of unbonding.
	UnbondingTime time.Duration `protobuf:"bytes,1,opt,name=unbonding_time,json=unbondingTime,proto3,stdduration" json:"unbonding_time"`
	// max_validators is the maximum number of validators.
	MaxValidators uint32 `protobuf:"varint,2,opt,name=max_validators,json=maxValidators,proto3" json:"max_validators,omitempty"`
	// max_entries is the max entries for either unbonding delegation or redelegation (per pair/trio).
	MaxEntries uint32 `protobuf:"varint,3,opt,name=max_entries,json=maxEntries,proto3" json:"max_entries,omitempty"`
	// historical_entries is the number of historical entries to persist.
	HistoricalEntries uint32 `protobuf:"varint,4,opt,name=historical_entries,json=historicalEntries,proto3" json:"historical_entries,omitempty"`
	// bond_denom defines the bondable coin denomination.
	BondDenom string `protobuf:"bytes,5,opt,name=bond_denom,json=bondDenom,proto3" json:"bond_denom,omitempty"`
}

// ParamKeyTable returns the parameter key table for wasm module
func ParamKeyTable() paramtypes.KeyTable {
	return paramtypes.NewKeyTable().RegisterParamSet(&LegacyParams{})
}

// ParamSetPairs implements the ParamSet interface and returns all the key/value pairs
func (p *LegacyParams) ParamSetPairs() paramtypes.ParamSetPairs {
	return paramtypes.ParamSetPairs{
		paramtypes.NewParamSetPair([]byte("unbonding_time"), &p.UnbondingTime, func(i interface{}) error { return nil }),
		paramtypes.NewParamSetPair([]byte("max_validators"), &p.MaxValidators, func(i interface{}) error { return nil }),
		paramtypes.NewParamSetPair([]byte("max_entries"), &p.MaxEntries, func(i interface{}) error { return nil }),
		paramtypes.NewParamSetPair([]byte("historical_entries"), &p.HistoricalEntries, func(i interface{}) error { return nil }),
		paramtypes.NewParamSetPair([]byte("bond_denom"), &p.BondDenom, func(i interface{}) error { return nil }),
	}
}

// LegacyQueryServer wraps the staking QueryServer and sets legacy parameters for pre-upgrade height queries
type LegacyQueryServer struct {
	// Embed the original query server to inherit all methods
	stakingtypes.QueryServer
	keeper         *keeper.Keeper
	legacySubspace paramtypes.Subspace
}

// NewLegacyQueryServer creates a new LegacyQueryServer instance
func NewLegacyQueryServer(
	originalServer stakingtypes.QueryServer,
	legacySubspace paramtypes.Subspace,
	keeper *keeper.Keeper,
) stakingtypes.QueryServer {
	return &LegacyQueryServer{
		QueryServer:    originalServer,
		keeper:         keeper,
		legacySubspace: legacySubspace,
	}
}

// ensureLegacyParams ensures that legacy parameters are set for pre-upgrade height queries
func (q *LegacyQueryServer) ensureLegacyParams(ctx context.Context) context.Context {
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	// Only set legacy params for pre-upgrade heights
	legacyMode := legacyupgrade.GetLegacyHandling(sdkCtx.ChainID(), sdkCtx.BlockHeight())
	sdkCtx.Logger().Info("Setting legacy params for pre-upgrade height queries",
		"block_height", sdkCtx.BlockHeight(),
		"legacy_mode", legacyMode,
		"chain_id", sdkCtx.ChainID(),
		"ctx", sdkCtx,
	)

	if legacyMode == legacyupgrade.LegacyHandlingV1 {
		var params LegacyParams
		q.legacySubspace.WithKeyTable(ParamKeyTable()).GetParamSet(sdkCtx, &params)

		// Set the params directly in the keeper
		q.keeper.SetParams(sdkCtx, stakingtypes.Params{
			UnbondingTime:     params.UnbondingTime,
			MaxValidators:     params.MaxValidators,
			MaxEntries:        params.MaxEntries,
			HistoricalEntries: params.HistoricalEntries,
			BondDenom:         params.BondDenom,
			MinCommissionRate: sdk.ZeroDec(),
		})

		// Return updated context
		sdkCtx.Logger().Info("Legacy params set for pre-upgrade height queries",
			"block_height", sdkCtx.BlockHeight(),
			"chain_id", sdkCtx.ChainID(),
			"params", params,
			"legacy_mode", legacyMode,
			"ctx", sdkCtx,
		)
		return sdk.WrapSDKContext(sdkCtx)
	}

	if legacyMode == legacyupgrade.LegacyHandlingV2 {
		var params stakingtypes.Params
		q.legacySubspace.WithKeyTable(stakingtypes.ParamKeyTable()).GetParamSet(sdkCtx, &params)

		// Set the params directly in the keeper
		q.keeper.SetParams(sdkCtx, params)

		// Return updated context
		sdkCtx.Logger().Info("Legacy params set for pre-upgrade height queries",
			"block_height", sdkCtx.BlockHeight(),
			"chain_id", sdkCtx.ChainID(),
			"params", params,
			"legacy_mode", legacyMode,
			"ctx", sdkCtx,
		)
		return sdk.WrapSDKContext(sdkCtx)
	}

	return ctx
}

// Implement the gRPC query service methods by forwarding to the original server
// after ensuring legacy parameters are set

func (q *LegacyQueryServer) Validators(ctx context.Context, req *stakingtypes.QueryValidatorsRequest) (*stakingtypes.QueryValidatorsResponse, error) {
	return q.QueryServer.Validators(q.ensureLegacyParams(ctx), req)
}

func (q *LegacyQueryServer) Validator(ctx context.Context, req *stakingtypes.QueryValidatorRequest) (*stakingtypes.QueryValidatorResponse, error) {
	return q.QueryServer.Validator(q.ensureLegacyParams(ctx), req)
}

func (q *LegacyQueryServer) ValidatorDelegations(ctx context.Context, req *stakingtypes.QueryValidatorDelegationsRequest) (*stakingtypes.QueryValidatorDelegationsResponse, error) {
	return q.QueryServer.ValidatorDelegations(q.ensureLegacyParams(ctx), req)
}

func (q *LegacyQueryServer) ValidatorUnbondingDelegations(ctx context.Context, req *stakingtypes.QueryValidatorUnbondingDelegationsRequest) (*stakingtypes.QueryValidatorUnbondingDelegationsResponse, error) {
	return q.QueryServer.ValidatorUnbondingDelegations(q.ensureLegacyParams(ctx), req)
}

func (q *LegacyQueryServer) Delegation(ctx context.Context, req *stakingtypes.QueryDelegationRequest) (*stakingtypes.QueryDelegationResponse, error) {
	return q.QueryServer.Delegation(q.ensureLegacyParams(ctx), req)
}

func (q *LegacyQueryServer) UnbondingDelegation(ctx context.Context, req *stakingtypes.QueryUnbondingDelegationRequest) (*stakingtypes.QueryUnbondingDelegationResponse, error) {
	return q.QueryServer.UnbondingDelegation(q.ensureLegacyParams(ctx), req)
}

func (q *LegacyQueryServer) DelegatorDelegations(ctx context.Context, req *stakingtypes.QueryDelegatorDelegationsRequest) (*stakingtypes.QueryDelegatorDelegationsResponse, error) {
	return q.QueryServer.DelegatorDelegations(q.ensureLegacyParams(ctx), req)
}

func (q *LegacyQueryServer) DelegatorUnbondingDelegations(ctx context.Context, req *stakingtypes.QueryDelegatorUnbondingDelegationsRequest) (*stakingtypes.QueryDelegatorUnbondingDelegationsResponse, error) {
	return q.QueryServer.DelegatorUnbondingDelegations(q.ensureLegacyParams(ctx), req)
}

func (q *LegacyQueryServer) Redelegations(ctx context.Context, req *stakingtypes.QueryRedelegationsRequest) (*stakingtypes.QueryRedelegationsResponse, error) {
	return q.QueryServer.Redelegations(q.ensureLegacyParams(ctx), req)
}

func (q *LegacyQueryServer) DelegatorValidators(ctx context.Context, req *stakingtypes.QueryDelegatorValidatorsRequest) (*stakingtypes.QueryDelegatorValidatorsResponse, error) {
	return q.QueryServer.DelegatorValidators(q.ensureLegacyParams(ctx), req)
}

func (q *LegacyQueryServer) DelegatorValidator(ctx context.Context, req *stakingtypes.QueryDelegatorValidatorRequest) (*stakingtypes.QueryDelegatorValidatorResponse, error) {
	return q.QueryServer.DelegatorValidator(q.ensureLegacyParams(ctx), req)
}

func (q *LegacyQueryServer) HistoricalInfo(ctx context.Context, req *stakingtypes.QueryHistoricalInfoRequest) (*stakingtypes.QueryHistoricalInfoResponse, error) {
	return q.QueryServer.HistoricalInfo(q.ensureLegacyParams(ctx), req)
}

func (q *LegacyQueryServer) Pool(ctx context.Context, req *stakingtypes.QueryPoolRequest) (*stakingtypes.QueryPoolResponse, error) {
	return q.QueryServer.Pool(q.ensureLegacyParams(ctx), req)
}

func (q *LegacyQueryServer) Params(ctx context.Context, req *stakingtypes.QueryParamsRequest) (*stakingtypes.QueryParamsResponse, error) {
	return q.QueryServer.Params(q.ensureLegacyParams(ctx), req)
}

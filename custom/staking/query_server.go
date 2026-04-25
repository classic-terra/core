package staking

import (
	"context"
	"strconv"
	"strings"

	"cosmossdk.io/math"
	"cosmossdk.io/store/prefix"
	storetypes "cosmossdk.io/store/types"
	legacytypes "github.com/classic-terra/core/v4/custom/staking/types"
	legacyupgrade "github.com/classic-terra/core/v4/custom/upgrade/legacy"
	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/query"
	paramtypes "github.com/cosmos/cosmos-sdk/x/params/types"
	"github.com/cosmos/cosmos-sdk/x/staking/keeper"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// LegacyQueryServer wraps the staking QueryServer and sets legacy parameters for pre-upgrade height queries
type LegacyQueryServer struct {
	// Embed the original query server to inherit all methods
	stakingtypes.QueryServer
	keeper         *keeper.Keeper
	legacySubspace paramtypes.Subspace
	cdc            codec.BinaryCodec
	storeKey       storetypes.StoreKey
}

// NewLegacyQueryServer creates a new LegacyQueryServer instance.
//
// `cdc` and `storeKey` are required for the pre-v5-staking-migration
// ValidatorDelegations fallback path, which scans the primary DelegationKey
// (0x31) prefix directly when the SDK's reverse-index (0x71) hasn't been
// backfilled at the queried height.
func NewLegacyQueryServer(
	originalServer stakingtypes.QueryServer,
	legacySubspace paramtypes.Subspace,
	keeper *keeper.Keeper,
	cdc codec.BinaryCodec,
	storeKey storetypes.StoreKey,
) stakingtypes.QueryServer {
	return &LegacyQueryServer{
		QueryServer:    originalServer,
		keeper:         keeper,
		legacySubspace: legacySubspace,
		cdc:            cdc,
		storeKey:       storeKey,
	}
}

// ensureLegacyParams ensures that legacy parameters are set for pre-upgrade height queries
func (q *LegacyQueryServer) ensureLegacyParams(ctx context.Context) context.Context {
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	// Only set legacy params for pre-upgrade heights
	legacyMode := legacyupgrade.GetLegacyHandling(sdkCtx.ChainID(), sdkCtx.BlockHeight())
	sdkCtx.Logger().Debug("Setting legacy params for pre-upgrade height queries",
		"block_height", sdkCtx.BlockHeight(),
		"legacy_mode", legacyMode,
		"chain_id", sdkCtx.ChainID(),
	)

	if legacyMode == legacyupgrade.LegacyHandlingV1 {
		if !q.legacySubspace.HasKeyTable() {
			q.legacySubspace.WithKeyTable(legacytypes.ParamKeyTable())
		}

		var params legacytypes.LegacyParams
		q.legacySubspace.GetParamSet(sdkCtx, &params)

		// Set the params directly in the keeper
		q.keeper.SetParams(sdkCtx, stakingtypes.Params{
			UnbondingTime:     params.UnbondingTime,
			MaxValidators:     params.MaxValidators,
			MaxEntries:        params.MaxEntries,
			HistoricalEntries: params.HistoricalEntries,
			BondDenom:         params.BondDenom,
			MinCommissionRate: math.LegacyZeroDec(),
		})

		// Return updated context
		sdkCtx.Logger().Debug("Legacy params set for pre-upgrade height queries",
			"block_height", sdkCtx.BlockHeight(),
			"chain_id", sdkCtx.ChainID(),
			"params", params,
			"legacy_mode", legacyMode,
			"ctx", sdkCtx,
		)
		return sdk.WrapSDKContext(sdkCtx)
	}

	if legacyMode == legacyupgrade.LegacyHandlingV2 {
		if !q.legacySubspace.HasKeyTable() {
			q.legacySubspace.WithKeyTable(stakingtypes.ParamKeyTable())
		}

		var params stakingtypes.Params
		q.legacySubspace.GetParamSet(sdkCtx, &params)

		// Set the params directly in the keeper
		q.keeper.SetParams(sdkCtx, params)

		// Return updated context
		sdkCtx.Logger().Debug("Legacy params set for pre-upgrade height queries",
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
	ensuredCtx := q.ensureLegacyParams(ctx)
	sdkCtx := sdk.UnwrapSDKContext(ensuredCtx)
	if legacyupgrade.IsPreStakingV5(sdkCtx.ChainID(), sdkCtx.BlockHeight()) {
		return q.validatorDelegationsLegacy(sdkCtx, req)
	}
	return q.QueryServer.ValidatorDelegations(ensuredCtx, req)
}

// validatorDelegationsLegacy reproduces cosmos-sdk's unexported
// `getValidatorDelegationsLegacy` (x/staking/keeper/grpc_query.go): it scans
// the primary DelegationKey (0x31) prefix and filters by validator. Used for
// archive queries at heights before the v4→v5 staking migration backfilled the
// DelegationByValIndexKey (0x71) reverse-index that the SDK's default
// ValidatorDelegations now relies on.
func (q *LegacyQueryServer) validatorDelegationsLegacy(
	ctx sdk.Context, req *stakingtypes.QueryValidatorDelegationsRequest,
) (*stakingtypes.QueryValidatorDelegationsResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "empty request")
	}
	if req.ValidatorAddr == "" {
		return nil, status.Error(codes.InvalidArgument, "validator address cannot be empty")
	}
	if _, err := sdk.ValAddressFromBech32(req.ValidatorAddr); err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	store := ctx.KVStore(q.storeKey)
	delStore := prefix.NewStore(store, stakingtypes.DelegationKey)

	dels, pageRes, err := query.GenericFilteredPaginate(
		q.cdc, delStore, req.Pagination,
		func(_ []byte, d *stakingtypes.Delegation) (*stakingtypes.Delegation, error) {
			if !strings.EqualFold(d.GetValidatorAddr(), req.ValidatorAddr) {
				return nil, nil
			}
			return d, nil
		},
		func() *stakingtypes.Delegation { return &stakingtypes.Delegation{} },
	)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	delegations := make(stakingtypes.Delegations, 0, len(dels))
	for _, d := range dels {
		delegations = append(delegations, *d)
	}

	delResps, err := q.delegationsToDelegationResponses(ctx, delegations)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &stakingtypes.QueryValidatorDelegationsResponse{
		DelegationResponses: delResps,
		Pagination:          pageRes,
	}, nil
}

// delegationsToDelegationResponses mirrors the unexported helper of the same
// name in cosmos-sdk's staking keeper: it looks up the validator for each
// delegation and converts shares to bonded balance.
func (q *LegacyQueryServer) delegationsToDelegationResponses(
	ctx sdk.Context, delegations stakingtypes.Delegations,
) (stakingtypes.DelegationResponses, error) {
	bondDenom, err := q.keeper.BondDenom(ctx)
	if err != nil {
		return nil, err
	}
	resps := make(stakingtypes.DelegationResponses, 0, len(delegations))
	for _, d := range delegations {
		valAddr, err := sdk.ValAddressFromBech32(d.GetValidatorAddr())
		if err != nil {
			return nil, err
		}
		val, err := q.keeper.GetValidator(ctx, valAddr)
		if err != nil {
			return nil, err
		}
		balance := val.TokensFromShares(d.Shares).TruncateInt()
		resps = append(resps, stakingtypes.NewDelegationResp(
			d.GetDelegatorAddr(), d.GetValidatorAddr(), d.Shares, sdk.NewCoin(bondDenom, balance),
		))
	}
	return resps, nil
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
	ensuredCtx := q.ensureLegacyParams(ctx)
	sdkCtx := sdk.UnwrapSDKContext(ensuredCtx)
	if legacyupgrade.IsPreStakingV5(sdkCtx.ChainID(), sdkCtx.BlockHeight()) {
		return q.historicalInfoLegacy(sdkCtx, req)
	}
	return q.QueryServer.HistoricalInfo(ensuredCtx, req)
}

// historicalInfoLegacy reads HistoricalInfo using the pre-v5-staking-migration
// key encoding: prefix 0x50 followed by the ASCII-decimal height string. The
// v5 migration (cosmos-sdk@v0.53.6/x/staking/migrations/v5/store.go:39) re-keys
// every entry to a big-endian uint64; before that migration ran (block
// 28214400 on Columbus, 28917279 on Rebel-2) IAVL state contains only the old
// string-format keys, so the SDK's GetHistoricalInfo — which constructs the
// new binary key — misses and returns NotFound.
func (q *LegacyQueryServer) historicalInfoLegacy(
	ctx sdk.Context, req *stakingtypes.QueryHistoricalInfoRequest,
) (*stakingtypes.QueryHistoricalInfoResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "empty request")
	}
	if req.Height < 0 {
		return nil, status.Error(codes.InvalidArgument, "height cannot be negative")
	}

	store := ctx.KVStore(q.storeKey)
	legacyKey := append([]byte{}, stakingtypes.HistoricalInfoKey...)
	legacyKey = append(legacyKey, []byte(strconv.FormatInt(req.Height, 10))...)
	bz := store.Get(legacyKey)
	if bz == nil {
		return nil, status.Errorf(codes.NotFound, "historical info for height %d not found", req.Height)
	}

	var hi stakingtypes.HistoricalInfo
	if err := q.cdc.Unmarshal(bz, &hi); err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	return &stakingtypes.QueryHistoricalInfoResponse{Hist: &hi}, nil
}

func (q *LegacyQueryServer) Pool(ctx context.Context, req *stakingtypes.QueryPoolRequest) (*stakingtypes.QueryPoolResponse, error) {
	return q.QueryServer.Pool(q.ensureLegacyParams(ctx), req)
}

func (q *LegacyQueryServer) Params(ctx context.Context, req *stakingtypes.QueryParamsRequest) (*stakingtypes.QueryParamsResponse, error) {
	return q.QueryServer.Params(q.ensureLegacyParams(ctx), req)
}

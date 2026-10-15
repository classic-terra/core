package simulation

import (
	"fmt"
	"math/rand"
	"time"

	"cosmossdk.io/math"
	"github.com/classic-terra/core/v4/x/ustcstaking/keeper"
	"github.com/classic-terra/core/v4/x/ustcstaking/types"
	"github.com/cosmos/cosmos-sdk/baseapp"
	"github.com/cosmos/cosmos-sdk/testutil/sims"
	simtestutil "github.com/cosmos/cosmos-sdk/testutil/sims"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"
	simtypes "github.com/cosmos/cosmos-sdk/types/simulation"
	sdksim "github.com/cosmos/cosmos-sdk/x/simulation"
)

// Keepers are simulation-only transaction dependencies, not additions to the
// native keeper or validator consensus lifecycle.
type Keepers struct {
	Accounts sdksim.AccountKeeper
	Bank     sdksim.BankKeeper
}

func WeightedOperations(state module.SimulationState, k keeper.Keeper, dependencies Keepers) []simtypes.WeightedOperation {
	messages := []sdk.Msg{&types.MsgStake{}, &types.MsgClaimRewards{}, &types.MsgBeginUnbonding{}, &types.MsgWithdraw{}}
	result := make([]simtypes.WeightedOperation, 0, len(messages))
	for _, msg := range messages {
		key := "op_weight_ustcstaking_" + sdk.MsgTypeURL(msg)
		weight := 100
		state.AppParams.GetOrGenerate(key, &weight, nil, func(_ *rand.Rand) { weight = 100 })
		result = append(result, sdksim.NewWeightedOperation(weight, nativeOperation(state, k, dependencies, msg)))
	}
	return result
}

func nativeOperation(state module.SimulationState, k keeper.Keeper, dependencies Keepers, kind sdk.Msg) simtypes.Operation {
	return func(r *rand.Rand, app *baseapp.BaseApp, ctx sdk.Context, accounts []simtypes.Account, chainID string) (simtypes.OperationMsg, []simtypes.FutureOperation, error) {
		if dependencies.Accounts == nil || dependencies.Bank == nil || state.TxConfig == nil {
			return simtypes.NoOpMsg(types.ModuleName, sdk.MsgTypeURL(kind), "missing simulation transaction dependencies"), nil, fmt.Errorf("USTC simulation requires real account/bank keepers and TxConfig")
		}
		var account simtypes.Account
		var msg sdk.Msg
		var position types.Position
		switch kind.(type) {
		case *types.MsgStake:
			params := k.GetParams(ctx)
			if params.Paused || len(params.LockTiers) == 0 {
				return simtypes.NoOpMsg(types.ModuleName, sdk.MsgTypeURL(kind), "staking is paused or no configured tier"), nil, nil
			}
			eligible := make([]simtypes.Account, 0, len(accounts))
			for _, candidate := range accounts {
				if dependencies.Bank.SpendableCoins(ctx, candidate.Address).AmountOf(types.BondDenom).GT(math.OneInt()) {
					eligible = append(eligible, candidate)
				}
			}
			if len(eligible) == 0 {
				return simtypes.NoOpMsg(types.ModuleName, sdk.MsgTypeURL(kind), "no funded simulation account"), nil, nil
			}
			account = eligible[r.Intn(len(eligible))]
			balance := dependencies.Bank.SpendableCoins(ctx, account.Address).AmountOf(types.BondDenom)
			limit := balance.QuoRaw(2)
			if limit.GT(math.NewInt(1_000_000)) {
				limit = math.NewInt(1_000_000)
			}
			amount := math.NewInt(1 + r.Int63n(limit.Int64()))
			tier := params.LockTiers[r.Intn(len(params.LockTiers))]
			msg = &types.MsgStake{Owner: account.Address.String(), Amount: sdk.NewCoin(types.BondDenom, amount), LockTierId: tier.Id}
		default:
			positions := make([]types.Position, 0)
			k.IteratePositions(ctx, func(candidate types.Position) bool {
				eligible := false
				switch kind.(type) {
				case *types.MsgClaimRewards:
					eligible = true
				case *types.MsgBeginUnbonding:
					eligible = candidate.Status == types.PositionStatus_POSITION_STATUS_ACTIVE
				case *types.MsgWithdraw:
					eligible = candidate.Status == types.PositionStatus_POSITION_STATUS_UNBONDING && candidate.UnbondingEndTime != nil && !ctx.BlockTime().Before(*candidate.UnbondingEndTime)
				}
				if eligible {
					for _, account := range accounts {
						if account.Address.String() == candidate.Owner {
							positions = append(positions, candidate)
							break
						}
					}
				}
				return false
			})
			if len(positions) == 0 {
				return simtypes.NoOpMsg(types.ModuleName, sdk.MsgTypeURL(kind), "no eligible owned position"), nil, nil
			}
			position = positions[r.Intn(len(positions))]
			for _, candidate := range accounts {
				if candidate.Address.String() == position.Owner {
					account = candidate
					break
				}
			}
			switch kind.(type) {
			case *types.MsgClaimRewards:
				msg = &types.MsgClaimRewards{Owner: position.Owner, PositionId: position.Id}
			case *types.MsgBeginUnbonding:
				msg = &types.MsgBeginUnbonding{Owner: position.Owner, PositionId: position.Id}
			case *types.MsgWithdraw:
				msg = &types.MsgWithdraw{Owner: position.Owner, PositionId: position.Id}
			}
		}
		txContext := ctx.WithChainID(chainID)
		signerAccount := dependencies.Accounts.GetAccount(txContext, account.Address)
		tx, err := simtestutil.GenSignedMockTx(r, state.TxConfig, []sdk.Msg{msg}, sdk.NewCoins(), sims.DefaultGenTxGas, chainID, []uint64{signerAccount.GetAccountNumber()}, []uint64{signerAccount.GetSequence()}, account.PrivKey)
		if err != nil {
			return simtypes.NoOpMsg(types.ModuleName, sdk.MsgTypeURL(msg), "unable to sign simulation transaction"), nil, err
		}
		_, result, err := app.SimDeliver(state.TxConfig.TxEncoder(), tx)
		if err != nil {
			return simtypes.NoOpMsg(types.ModuleName, sdk.MsgTypeURL(msg), "simulation delivery failed"), nil, err
		}
		if result == nil {
			return simtypes.NoOpMsg(types.ModuleName, sdk.MsgTypeURL(msg), "simulation transaction returned no result"), nil, nil
		}
		operation := simtypes.NewOperationMsg(msg, true, "")
		if err := k.ValidateState(ctx); err != nil {
			return operation, nil, err
		}
		var future []simtypes.FutureOperation
		switch msg.(type) {
		case *types.MsgStake:
			future = []simtypes.FutureOperation{
				{BlockHeight: int(ctx.BlockHeight() + 1), Op: nativeOperation(state, k, dependencies, &types.MsgClaimRewards{})},
				{BlockHeight: int(ctx.BlockHeight() + 2), Op: nativeOperation(state, k, dependencies, &types.MsgBeginUnbonding{})},
			}
		case *types.MsgBeginUnbonding:
			future = []simtypes.FutureOperation{{BlockTime: ctx.BlockTime().Add(*position.LockDuration).Add(time.Nanosecond), Op: nativeOperation(state, k, dependencies, &types.MsgWithdraw{})}}
		}
		return operation, future, nil
	}
}

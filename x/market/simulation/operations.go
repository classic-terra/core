package simulation

// DONTCOVER

import (
	"math/rand"
	"strings"

	"cosmossdk.io/math"
	core "github.com/classic-terra/core/v4/types"
	"github.com/classic-terra/core/v4/x/market/types"
	"github.com/cosmos/cosmos-sdk/baseapp"
	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/cosmos/cosmos-sdk/std"
	simtestutil "github.com/cosmos/cosmos-sdk/testutil/sims"
	sdk "github.com/cosmos/cosmos-sdk/types"
	simtypes "github.com/cosmos/cosmos-sdk/types/simulation"
	"github.com/cosmos/cosmos-sdk/x/auth/tx"
	banksim "github.com/cosmos/cosmos-sdk/x/bank/simulation"
	"github.com/cosmos/cosmos-sdk/x/simulation"
)

// Simulation operation weights constants
const (
	OpWeightMsgSwap = "op_weight_msg_swap" //#nosec
)

// allowedSimSwapDenoms mirrors the market module's code-fixed
// allowedSwapDenoms (production default: only uusd). The simulation must only
// pick ask/offer denoms from this set; otherwise every swap hits
// ErrInvalidSwapPair because allowedSwapDenoms is intentionally not a gov param.
var allowedSimSwapDenoms = map[string]bool{
	core.MicroUSDDenom: true,
}

// pickSwapDenom returns a random oracle denom that is also in the
// allowedSwapDenoms set. Returns "" (no-op) if none qualify.
func pickSwapDenom(r *rand.Rand, oracleDenoms []string) string {
	allowed := make([]string, 0, len(oracleDenoms))
	for _, d := range oracleDenoms {
		if allowedSimSwapDenoms[d] {
			allowed = append(allowed, d)
		}
	}
	if len(allowed) == 0 {
		return ""
	}
	return allowed[r.Intn(len(allowed))]
}

// WeightedOperations returns all the operations from the module with their respective weights
func WeightedOperations(
	appParams simtypes.AppParams,
	cdc codec.JSONCodec,
	ak types.AccountKeeper,
	bk types.BankKeeper,
	ok types.OracleKeeper,
) simulation.WeightedOperations {
	var weightMsgSwap int
	appParams.GetOrGenerate(OpWeightMsgSwap, &weightMsgSwap, nil,
		func(*rand.Rand) {
			weightMsgSwap = banksim.DefaultWeightMsgSend
		},
	)

	return simulation.WeightedOperations{
		simulation.NewWeightedOperation(
			weightMsgSwap,
			SimulateMsgSwap(ak, bk, ok),
		),
	}
}

// SimulateMsgSwap generates a MsgSwap with random values.
func SimulateMsgSwap(
	ak types.AccountKeeper,
	bk types.BankKeeper,
	ok types.OracleKeeper,
) simtypes.Operation {
	return func(
		r *rand.Rand, app *baseapp.BaseApp, ctx sdk.Context, accs []simtypes.Account, chainID string,
	) (simtypes.OperationMsg, []simtypes.FutureOperation, error) {
		simAccount, _ := simtypes.RandomAcc(r, accs)
		account := ak.GetAccount(ctx, simAccount.Address)

		spendable := bk.SpendableCoins(ctx, simAccount.Address)
		fees, err := simtypes.RandomFees(r, ctx, spendable)
		if err != nil {
			return simtypes.NoOpMsg(types.ModuleName, types.TypeMsgSwap, "unable to generate fees"), nil, err
		}

		var whitelist []string
		ok.IterateLunaExchangeRates(ctx, func(denom string, ex math.LegacyDec) bool {
			whitelist = append(whitelist, denom)
			return false
		})

		// Only pick denoms the market module actually allows (code-fixed set).
		swapDenom := pickSwapDenom(r, whitelist)
		if swapDenom == "" {
			return simtypes.NoOpMsg(types.ModuleName, types.TypeMsgSwap, "no allowed swap denom with exchange rate"), nil, nil
		}

		// One side is always uluna; the other is an allowed denom.
		var offerDenom, askDenom string
		if r.Intn(2) == 0 {
			offerDenom = core.MicroLunaDenom
			askDenom = swapDenom
		} else {
			offerDenom = swapDenom
			askDenom = core.MicroLunaDenom
		}

		amount := simtypes.RandomAmount(r, spendable.AmountOf(offerDenom).Sub(fees.AmountOf(offerDenom)))
		if amount.Equal(math.ZeroInt()) {
			return simtypes.NoOpMsg(types.ModuleName, types.TypeMsgSwap, "not enough offer denom amount"), nil, nil
		}

		msg := types.NewMsgSwap(simAccount.Address, sdk.NewCoin(offerDenom, amount), askDenom)

		ir := codectypes.NewInterfaceRegistry()
		std.RegisterInterfaces(ir)
		txGen := tx.NewTxConfig(codec.NewProtoCodec(ir), tx.DefaultSignModes)
		tx, err := simtestutil.GenSignedMockTx(
			r,
			txGen,
			[]sdk.Msg{msg},
			fees,
			simtestutil.DefaultGenTxGas,
			chainID,
			[]uint64{account.GetAccountNumber()},
			[]uint64{account.GetSequence()},
			simAccount.PrivKey,
		)
		if err != nil {
			return simtypes.NoOpMsg(types.ModuleName, msg.Type(), "unable to generate mock tx"), nil, err
		}

		_, _, err = app.SimDeliver(txGen.TxEncoder(), tx)
		if err != nil {
			return simtypes.NoOpMsg(types.ModuleName, msg.Type(), "unable to deliver tx"), nil, err
		}

		return simtypes.NewOperationMsg(msg, true, ""), nil, nil
	}
}

// SimulateMsgSwapSend generates a MsgSwapSend with random values.
func SimulateMsgSwapSend(
	ak types.AccountKeeper,
	bk types.BankKeeper,
	ok types.OracleKeeper,
) simtypes.Operation {
	return func(
		r *rand.Rand, app *baseapp.BaseApp, ctx sdk.Context, accs []simtypes.Account, chainID string,
	) (simtypes.OperationMsg, []simtypes.FutureOperation, error) {
		simAccount, _ := simtypes.RandomAcc(r, accs)
		receiverAccount, _ := simtypes.RandomAcc(r, accs)
		account := ak.GetAccount(ctx, simAccount.Address)

		spendable := bk.SpendableCoins(ctx, simAccount.Address)
		fees, err := simtypes.RandomFees(r, ctx, spendable)
		if err != nil {
			return simtypes.NoOpMsg(types.ModuleName, types.TypeMsgSwapSend, "unable to generate fees"), nil, err
		}

		var whitelist []string
		ok.IterateLunaExchangeRates(ctx, func(denom string, ex math.LegacyDec) bool {
			whitelist = append(whitelist, denom)
			return false
		})

		// Only pick denoms the market module actually allows (code-fixed set).
		swapDenom := pickSwapDenom(r, whitelist)
		if swapDenom == "" {
			return simtypes.NoOpMsg(types.ModuleName, types.TypeMsgSwapSend, "no allowed swap denom with exchange rate"), nil, nil
		}

		// One side is always uluna; the other is an allowed denom.
		var offerDenom, askDenom string
		if r.Intn(2) == 0 {
			offerDenom = core.MicroLunaDenom
			askDenom = swapDenom
		} else {
			offerDenom = swapDenom
			askDenom = core.MicroLunaDenom
		}

		// Check send_enabled status of offer denom
		if !bk.IsSendEnabledCoin(ctx, sdk.Coin{Denom: offerDenom, Amount: math.NewInt(1)}) {
			return simtypes.NoOpMsg(types.ModuleName, types.TypeMsgSwapSend, err.Error()), nil, nil
		}

		amount := simtypes.RandomAmount(r, spendable.AmountOf(offerDenom).Sub(fees.AmountOf(offerDenom)))
		if amount.Equal(math.ZeroInt()) {
			return simtypes.NoOpMsg(types.ModuleName, types.TypeMsgSwapSend, "not enough offer denom amount"), nil, nil
		}

		msg := types.NewMsgSwapSend(simAccount.Address, receiverAccount.Address, sdk.NewCoin(offerDenom, amount), askDenom)

		ir := codectypes.NewInterfaceRegistry()
		std.RegisterInterfaces(ir)
		txGen := tx.NewTxConfig(codec.NewProtoCodec(ir), tx.DefaultSignModes)
		tx, err := simtestutil.GenSignedMockTx(
			r,
			txGen,
			[]sdk.Msg{msg},
			fees,
			simtestutil.DefaultGenTxGas,
			chainID,
			[]uint64{account.GetAccountNumber()},
			[]uint64{account.GetSequence()},
			simAccount.PrivKey,
		)
		if err != nil {
			return simtypes.NoOpMsg(types.ModuleName, msg.Type(), "unable to generate mock tx"), nil, err
		}

		_, _, err = app.SimDeliver(txGen.TxEncoder(), tx)
		if err != nil {
			if strings.Contains(err.Error(), "insufficient fee") {
				return simtypes.NoOpMsg(types.ModuleName, msg.Type(), "ignore tax error"), nil, nil
			}

			return simtypes.NoOpMsg(types.ModuleName, msg.Type(), "unable to deliver tx"), nil, err
		}

		return simtypes.NewOperationMsg(msg, true, ""), nil, nil
	}
}

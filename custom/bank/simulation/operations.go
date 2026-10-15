package simulation

import (
	"math/rand"
	"strings"

	sdkmath "cosmossdk.io/math"
	"cosmossdk.io/x/tx/signing"
	"github.com/cosmos/cosmos-sdk/baseapp"
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/codec"
	addresscodec "github.com/cosmos/cosmos-sdk/codec/address"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	cryptotypes "github.com/cosmos/cosmos-sdk/crypto/types"
	"github.com/cosmos/cosmos-sdk/std"
	simtestutil "github.com/cosmos/cosmos-sdk/testutil/sims"
	sdk "github.com/cosmos/cosmos-sdk/types"
	simtypes "github.com/cosmos/cosmos-sdk/types/simulation"
	"github.com/cosmos/cosmos-sdk/x/auth/tx"
	"github.com/cosmos/cosmos-sdk/x/bank/keeper"
	banksim "github.com/cosmos/cosmos-sdk/x/bank/simulation"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	"github.com/cosmos/cosmos-sdk/x/simulation"
	"github.com/cosmos/gogoproto/proto"
)

// Simulation operation weights constants
const (
	OpWeightMsgSend      = "op_weight_msg_send"      //#nosec
	OpWeightMsgMultiSend = "op_weight_msg_multisend" //#nosec
)

// WeightedOperations returns all the operations from the module with their respective weights
func WeightedOperations(
	appParams simtypes.AppParams, cdc codec.JSONCodec, ak banktypes.AccountKeeper, bk keeper.Keeper,
) simulation.WeightedOperations {
	var weightMsgSend, weightMsgMultiSend int
	appParams.GetOrGenerate(OpWeightMsgSend, &weightMsgSend, nil,
		func(*rand.Rand) {
			weightMsgSend = banksim.DefaultWeightMsgSend
		},
	)

	appParams.GetOrGenerate(OpWeightMsgMultiSend, &weightMsgMultiSend, nil,
		func(*rand.Rand) {
			weightMsgMultiSend = banksim.DefaultWeightMsgMultiSend
		},
	)

	return simulation.WeightedOperations{
		simulation.NewWeightedOperation(
			weightMsgSend,
			SimulateMsgSend(ak, bk),
		),
		simulation.NewWeightedOperation(
			weightMsgMultiSend,
			SimulateMsgMultiSend(ak, bk),
		),
	}
}

// makeTxConfig constructs a minimal client.TxConfig for simulations without pulling simapp.
func makeTxConfig() client.TxConfig {
	amino := codec.NewLegacyAmino()
	ir, err := codectypes.NewInterfaceRegistryWithOptions(codectypes.InterfaceRegistryOptions{
		ProtoFiles: proto.HybridResolver,
		SigningOptions: signing.Options{
			AddressCodec:          addresscodec.NewBech32Codec(sdk.GetConfig().GetBech32AccountAddrPrefix()),
			ValidatorAddressCodec: addresscodec.NewBech32Codec(sdk.GetConfig().GetBech32ValidatorAddrPrefix()),
		},
	})
	if err != nil {
		panic(err)
	}
	cdc := codec.NewProtoCodec(ir)
	std.RegisterInterfaces(ir)
	std.RegisterLegacyAminoCodec(amino)
	return tx.NewTxConfig(cdc, tx.DefaultSignModes)
}

// SimulateMsgSend tests and runs a single msg send where both
// accounts already exist.
func SimulateMsgSend(ak banktypes.AccountKeeper, bk keeper.Keeper) simtypes.Operation {
	return func(
		r *rand.Rand, app *baseapp.BaseApp, ctx sdk.Context,
		accs []simtypes.Account, chainID string,
	) (simtypes.OperationMsg, []simtypes.FutureOperation, error) {
		simAccount, toSimAcc, coins, skip := randomSendFields(r, ctx, accs, bk, ak)

		// Check send_enabled status of each coin denom
		if err := bk.IsSendEnabledCoins(ctx, coins...); err != nil {
			return simtypes.NoOpMsg(banktypes.ModuleName, sdk.MsgTypeURL(&banktypes.MsgSend{}), err.Error()), nil, nil
		}

		if skip {
			return simtypes.NoOpMsg(banktypes.ModuleName, sdk.MsgTypeURL(&banktypes.MsgSend{}), "skip all transfers"), nil, nil
		}

		msg := banktypes.NewMsgSend(simAccount.Address, toSimAcc.Address, coins)

		err := sendMsgSend(r, app, bk, ak, msg, ctx, chainID, []cryptotypes.PrivKey{simAccount.PrivKey})
		if err != nil {
			if strings.Contains(err.Error(), "insufficient fee") {
				return simtypes.NoOpMsg(banktypes.ModuleName, sdk.MsgTypeURL(&banktypes.MsgSend{}), "skip low fee due to tax"), nil, nil
			}

			return simtypes.NoOpMsg(banktypes.ModuleName, sdk.MsgTypeURL(&banktypes.MsgSend{}), "invalid transfers"), nil, err
		}

		return simtypes.NewOperationMsg(msg, true, ""), nil, nil
	}
}

// sendMsgSend sends a transaction with a MsgSend from a provided random account.
func sendMsgSend(
	r *rand.Rand, app *baseapp.BaseApp, bk keeper.Keeper, ak banktypes.AccountKeeper,
	msg *banktypes.MsgSend, ctx sdk.Context, chainID string, privkeys []cryptotypes.PrivKey,
) error {
	var (
		fees sdk.Coins
		err  error
	)

	from, err := sdk.AccAddressFromBech32(msg.FromAddress)
	if err != nil {
		return err
	}

	account := ak.GetAccount(ctx, from)
	spendable := bk.SpendableCoins(ctx, account.GetAddress())

	coins, hasNeg := spendable.SafeSub(msg.Amount...)
	if !hasNeg {
		fees, err = simtypes.RandomFees(r, ctx, coins)
		if err != nil {
			return err
		}
	}
	txGen := makeTxConfig()
	tx, err := simtestutil.GenSignedMockTx(
		r,
		txGen,
		[]sdk.Msg{msg},
		fees,
		simtestutil.DefaultGenTxGas,
		chainID,
		[]uint64{account.GetAccountNumber()},
		[]uint64{account.GetSequence()},
		privkeys...,
	)
	if err != nil {
		return err
	}

	_, _, err = app.SimDeliver(txGen.TxEncoder(), tx)
	if err != nil {
		return err
	}

	return nil
}

// SimulateMsgMultiSend tests and runs a single msg multisend, with randomized, capped number of inputs/outputs.
// all accounts in msg fields exist in state
func SimulateMsgMultiSend(ak banktypes.AccountKeeper, bk keeper.Keeper) simtypes.Operation {
	return func(
		r *rand.Rand, app *baseapp.BaseApp, ctx sdk.Context,
		accs []simtypes.Account, chainID string,
	) (simtypes.OperationMsg, []simtypes.FutureOperation, error) {
		// Use one input and split its first denomination into at most three
		// positive outputs. The remaining denominations are assigned whole to
		// random outputs so the transfer totals stay exactly balanced.
		simAccount, _, coins, skip := randomSendFields(r, ctx, accs, bk, ak)
		if skip {
			return simtypes.NoOpMsg(banktypes.ModuleName, sdk.MsgTypeURL(&banktypes.MsgMultiSend{}), "skip all transfers"), nil, nil
		}
		if err := bk.IsSendEnabledCoins(ctx, coins...); err != nil {
			return simtypes.NoOpMsg(banktypes.ModuleName, sdk.MsgTypeURL(&banktypes.MsgMultiSend{}), err.Error()), nil, nil
		}
		outputCount := 3
		if !coins[0].Amount.GT(sdk.NewInt64Coin(coins[0].Denom, 3).Amount) {
			outputCount = int(coins[0].Amount.Int64())
		}
		outputs := make([]banktypes.Output, outputCount)
		for i := range outputs {
			to, _ := simtypes.RandomAcc(r, accs)
			outputs[i] = banktypes.NewOutput(to.Address, sdk.NewCoins())
		}
		remaining := coins[0].Amount
		for i := 0; i < outputCount-1; i++ {
			max := remaining.Sub(sdkmath.NewInt(int64(outputCount - i - 1)))
			part, err := simtypes.RandPositiveInt(r, max)
			if err != nil {
				return simtypes.NoOpMsg(banktypes.ModuleName, sdk.MsgTypeURL(&banktypes.MsgMultiSend{}), err.Error()), nil, nil
			}
			outputs[i].Coins = outputs[i].Coins.Add(sdk.NewCoin(coins[0].Denom, part))
			remaining = remaining.Sub(part)
		}
		outputs[outputCount-1].Coins = outputs[outputCount-1].Coins.Add(sdk.NewCoin(coins[0].Denom, remaining))
		for _, coin := range coins[1:] {
			index := r.Intn(outputCount)
			outputs[index].Coins = outputs[index].Coins.Add(coin)
		}
		privs := []cryptotypes.PrivKey{simAccount.PrivKey}

		msg := &banktypes.MsgMultiSend{
			Inputs:  []banktypes.Input{banktypes.NewInput(simAccount.Address, coins)},
			Outputs: outputs,
		}

		err := sendMsgMultiSend(r, app, bk, ak, msg, ctx, chainID, privs)
		if err != nil {
			if strings.Contains(err.Error(), "insufficient fee") || strings.Contains(err.Error(), "sum inputs != sum outputs") {
				return simtypes.NoOpMsg(banktypes.ModuleName, sdk.MsgTypeURL(&banktypes.MsgMultiSend{}), "skip transfer invalidated by tax"), nil, nil
			}

			return simtypes.NoOpMsg(banktypes.ModuleName, sdk.MsgTypeURL(msg), "invalid transfers"), nil, err
		}

		return simtypes.NewOperationMsg(msg, true, ""), nil, nil
	}
}

// sendMsgMultiSend sends a transaction with a MsgMultiSend from a provided random
// account.
func sendMsgMultiSend(
	r *rand.Rand, app *baseapp.BaseApp, bk keeper.Keeper, ak banktypes.AccountKeeper,
	msg *banktypes.MsgMultiSend, ctx sdk.Context, chainID string, privkeys []cryptotypes.PrivKey,
) error {
	accountNumbers := make([]uint64, len(msg.Inputs))
	sequenceNumbers := make([]uint64, len(msg.Inputs))

	for i := 0; i < len(msg.Inputs); i++ {
		addr, err := sdk.AccAddressFromBech32(msg.Inputs[i].Address)
		if err != nil {
			panic(err)
		}
		acc := ak.GetAccount(ctx, addr)
		accountNumbers[i] = acc.GetAccountNumber()
		sequenceNumbers[i] = acc.GetSequence()
	}

	var (
		fees sdk.Coins
		err  error
	)

	addr, err := sdk.AccAddressFromBech32(msg.Inputs[0].Address)
	if err != nil {
		panic(err)
	}

	// feePayer is the first signer, i.e. first input address
	feePayer := ak.GetAccount(ctx, addr)
	spendable := bk.SpendableCoins(ctx, feePayer.GetAddress())

	coins, hasNeg := spendable.SafeSub(msg.Inputs[0].Coins...)
	if !hasNeg {
		fees, err = simtypes.RandomFees(r, ctx, coins)
		if err != nil {
			return err
		}
	}

	txGen := makeTxConfig()
	tx, err := simtestutil.GenSignedMockTx(
		r,
		txGen,
		[]sdk.Msg{msg},
		fees,
		simtestutil.DefaultGenTxGas,
		chainID,
		accountNumbers,
		sequenceNumbers,
		privkeys...,
	)
	if err != nil {
		return err
	}

	_, _, err = app.SimDeliver(txGen.TxEncoder(), tx)
	if err != nil {
		return err
	}

	return nil
}

// randomSendFields returns the sender and recipient simulation accounts as well
// as the transferred amount.
func randomSendFields(
	r *rand.Rand, ctx sdk.Context, accs []simtypes.Account, bk keeper.Keeper, ak banktypes.AccountKeeper,
) (simtypes.Account, simtypes.Account, sdk.Coins, bool) {
	simAccount, _ := simtypes.RandomAcc(r, accs)
	toSimAcc, _ := simtypes.RandomAcc(r, accs)

	// disallow sending money to yourself
	for simAccount.PubKey.Equals(toSimAcc.PubKey) {
		toSimAcc, _ = simtypes.RandomAcc(r, accs)
	}

	acc := ak.GetAccount(ctx, simAccount.Address)
	if acc == nil {
		return simAccount, toSimAcc, nil, true
	}

	spendable := bk.SpendableCoins(ctx, acc.GetAddress())

	sendCoins := simtypes.RandSubsetCoins(r, spendable)
	if sendCoins.Empty() {
		return simAccount, toSimAcc, nil, true
	}

	return simAccount, toSimAcc, sendCoins, false
}

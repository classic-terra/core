package simulation

import (
	"math/rand"
	"strings"

	"github.com/cosmos/cosmos-sdk/baseapp"
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	cryptotypes "github.com/cosmos/cosmos-sdk/crypto/types"
	"github.com/cosmos/cosmos-sdk/std"
	simtestutil "github.com/cosmos/cosmos-sdk/testutil/sims"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/auth/tx"
	"github.com/cosmos/cosmos-sdk/x/bank/keeper"
	banksim "github.com/cosmos/cosmos-sdk/x/bank/simulation"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	"github.com/cosmos/cosmos-sdk/x/simulation"

	simtypes "github.com/cosmos/cosmos-sdk/types/simulation"
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
	ir := codectypes.NewInterfaceRegistry()
	cdc := codec.NewProtoCodec(ir)
	std.RegisterInterfaces(ir)
	std.RegisterLegacyAminoCodec(amino)
	return tx.NewTxConfig(cdc, tx.DefaultSignModes)
}

// SimulateMsgSend tests and runs a single msg send where both
// accounts already exist.
// nolint: funlen
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
// nolint: interfacer
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
		// random number of inputs/outputs between [1, 3]
		inputs := make([]banktypes.Input, 1)
		outputs := make([]banktypes.Output, r.Intn(3)+1)

		// collect signer privKeys
		privs := make([]cryptotypes.PrivKey, len(inputs))

		// use map to check if address already exists as input
		usedAddrs := make(map[string]bool)
		simAccount, _, coins, skip := randomSendFields(r, ctx, accs, bk, ak)

		if skip {
			return simtypes.NoOpMsg(banktypes.ModuleName, sdk.MsgTypeURL(&banktypes.MsgMultiSend{}), "skip all transfers"), nil, nil
		}

		// set input address in used address map
		usedAddrs[simAccount.Address.String()] = true

		// set signer privkey
		privs[0] = simAccount.PrivKey

		// set next input and accumulate total sent coins
		inputs[0] = banktypes.NewInput(simAccount.Address, coins)

		// Check send_enabled status of each sent coin denom
		if err := bk.IsSendEnabledCoins(ctx, coins...); err != nil {
			return simtypes.NoOpMsg(banktypes.ModuleName, sdk.MsgTypeURL(&banktypes.MsgMultiSend{}), err.Error()), nil, nil
		}

		for o := range outputs {
			outAddr, _ := simtypes.RandomAcc(r, accs)

			var outCoins sdk.Coins
			// split total sent coins into random subsets for output
			if o == len(outputs)-1 {
				outCoins = coins
			} else {
				// take random subset of remaining coins for output
				// and update remaining coins
				outCoins = simtypes.RandSubsetCoins(r, coins)
				coins = coins.Sub(outCoins...)
			}

			outputs[o] = banktypes.NewOutput(outAddr.Address, outCoins)
		}

		// remove any output that has no coins
		i := 0
		for i < len(outputs) {
			if outputs[i].Coins.Empty() {
				outputs[i] = outputs[len(outputs)-1]
				outputs = outputs[:len(outputs)-1]
			} else {
				// continue onto next coin
				i++
			}
		}

		msg := &banktypes.MsgMultiSend{
			Inputs:  inputs,
			Outputs: outputs,
		}

		err := sendMsgMultiSend(r, app, bk, ak, msg, ctx, chainID, privs)
		if err != nil {
			if strings.Contains(err.Error(), "insufficient fee") {
				return simtypes.NoOpMsg(banktypes.ModuleName, sdk.MsgTypeURL(&banktypes.MsgMultiSend{}), "skip low fee due to tax"), nil, nil
			}

			return simtypes.NoOpMsg(banktypes.ModuleName, sdk.MsgTypeURL(msg), "invalid transfers"), nil, err
		}

		return simtypes.NewOperationMsg(msg, true, ""), nil, nil
	}
}

// sendMsgMultiSend sends a transaction with a MsgMultiSend from a provided random
// account.
// nolint: interfacer
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
// nolint: interfacer
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

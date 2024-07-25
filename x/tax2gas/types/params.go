package types

import (
	fmt "fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"

	ibcclienttypes "github.com/cosmos/ibc-go/v7/modules/core/02-client/types"
	ibcchanneltypes "github.com/cosmos/ibc-go/v7/modules/core/04-channel/types"
)

var (
	// DefaultMinGasPrices is set at runtime to the staking token with zero amount i.e. "0uatom"
	// see DefaultZeroGlobalFee method in gaia/x/globalfee/ante/fee.go.
	DefaultGasPrices = sdk.NewDecCoins(
		sdk.NewDecCoinFromDec("uluna", sdk.NewDecWithPrec(28325, 3)),
		sdk.NewDecCoinFromDec("usdr", sdk.NewDecWithPrec(52469, 5)),
		sdk.NewDecCoinFromDec("uusd", sdk.NewDecWithPrec(75, 2)),
		sdk.NewDecCoinFromDec("ukrw", sdk.NewDecWithPrec(850, 0)),
		sdk.NewDecCoinFromDec("umnt", sdk.NewDecWithPrec(2142855, 3)),
		sdk.NewDecCoinFromDec("ueur", sdk.NewDecWithPrec(625, 3)),
		sdk.NewDecCoinFromDec("ucny", sdk.NewDecWithPrec(49, 1)),
		sdk.NewDecCoinFromDec("ujpy", sdk.NewDecWithPrec(8185, 2)),
		sdk.NewDecCoinFromDec("ugbp", sdk.NewDecWithPrec(55, 2)),
		sdk.NewDecCoinFromDec("uinr", sdk.NewDecWithPrec(544, 1)),
		sdk.NewDecCoinFromDec("ucad", sdk.NewDecWithPrec(95, 2)),
		sdk.NewDecCoinFromDec("uchf", sdk.NewDecWithPrec(7, 1)),
		sdk.NewDecCoinFromDec("uaud", sdk.NewDecWithPrec(95, 2)),
		sdk.NewDecCoinFromDec("usgd", sdk.NewDec(1)),
		sdk.NewDecCoinFromDec("uthb", sdk.NewDecWithPrec(231, 1)),
		sdk.NewDecCoinFromDec("usek", sdk.NewDecWithPrec(625, 2)),
		sdk.NewDecCoinFromDec("unok", sdk.NewDecWithPrec(625, 2)),
		sdk.NewDecCoinFromDec("udkk", sdk.NewDecWithPrec(45, 1)),
		sdk.NewDecCoinFromDec("uidr", sdk.NewDecWithPrec(10900, 0)),
		sdk.NewDecCoinFromDec("uphp", sdk.NewDecWithPrec(38, 0)),
		sdk.NewDecCoinFromDec("uhkd", sdk.NewDecWithPrec(585, 2)),
		sdk.NewDecCoinFromDec("umyr", sdk.NewDecWithPrec(3, 0)),
		sdk.NewDecCoinFromDec("utwd", sdk.NewDecWithPrec(20, 0)),
	)
	DefaultBypassMinFeeMsgTypes = []string{
		sdk.MsgTypeURL(&ibcclienttypes.MsgUpdateClient{}),
		sdk.MsgTypeURL(&ibcchanneltypes.MsgRecvPacket{}),
		sdk.MsgTypeURL(&ibcchanneltypes.MsgAcknowledgement{}),
		sdk.MsgTypeURL(&ibcchanneltypes.MsgTimeout{}),
		sdk.MsgTypeURL(&ibcchanneltypes.MsgTimeoutOnClose{}),
	}

	// maxTotalBypassMinFeeMsgGasUsage is the allowed maximum gas usage
	// for all the bypass msgs in a transactions.
	// A transaction that contains only bypass message types and the gas usage does not
	// exceed maxTotalBypassMinFeeMsgGasUsage can be accepted with a zero fee.
	// For details, see gaiafeeante.NewFeeDecorator()
	DefaultmaxTotalBypassMinFeeMsgGasUsage uint64 = 1_000_000
)

func NewParams() Params {
	return Params{}
}

// DefaultParams are the default tax2gas module parameters.
func DefaultParams() Params {
	return Params{
		GasPrices:                       DefaultGasPrices,
		Enabled:                         false,
		BypassMinFeeMsgTypes:            DefaultBypassMinFeeMsgTypes,
		MaxTotalBypassMinFeeMsgGasUsage: DefaultmaxTotalBypassMinFeeMsgGasUsage,
	}
}

// Validate validates params.
func (p Params) Validate() error {
	if p.Enabled {
		if len(p.GasPrices) == 0 {
			return fmt.Errorf("must provide at least 1 gas prices")
		}
		if !p.GasPrices.IsAllPositive() {
			return fmt.Errorf("gas prices must be positive")
		}
	}
	return nil
}

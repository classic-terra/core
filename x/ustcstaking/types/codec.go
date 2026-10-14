package types

import (
	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/cosmos-sdk/codec/legacy"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/msgservice"
)

func RegisterInterfaces(registry codectypes.InterfaceRegistry) {
	registry.RegisterImplementations(
		(*sdk.Msg)(nil),
		&MsgStake{},
		&MsgBeginUnbonding{},
		&MsgWithdraw{},
		&MsgClaimRewards{},
		&MsgFundRewards{},
		&MsgUpdateParams{},
		&MsgReturnRewardDust{},
	)
	msgservice.RegisterMsgServiceDesc(registry, &_Msg_serviceDesc)
}

func RegisterLegacyAminoCodec(cdc *codec.LegacyAmino) {
	legacy.RegisterAminoMsg(cdc, &MsgStake{}, "ustcstaking/Stake")
	legacy.RegisterAminoMsg(cdc, &MsgBeginUnbonding{}, "ustcstaking/BeginUnbonding")
	legacy.RegisterAminoMsg(cdc, &MsgWithdraw{}, "ustcstaking/Withdraw")
	legacy.RegisterAminoMsg(cdc, &MsgClaimRewards{}, "ustcstaking/ClaimRewards")
	legacy.RegisterAminoMsg(cdc, &MsgFundRewards{}, "ustcstaking/FundRewards")
	legacy.RegisterAminoMsg(cdc, &MsgUpdateParams{}, "ustcstaking/UpdateParams")
	legacy.RegisterAminoMsg(cdc, &MsgReturnRewardDust{}, "ustcstaking/ReturnRewardDust")
}
